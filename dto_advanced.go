package convert

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DTOWarning is a non-fatal conversion observation produced by report/batch APIs.
type DTOWarning struct {
	Path    string
	Code    string
	Message string
	Value   any
}

// DTOReport describes a conversion attempt including warnings, defaults and changed paths.
type DTOReportResult[T any] struct {
	Value    T
	Warnings []DTOWarning
	Errors   []error
}

// DTOProfile captures reusable DTO options for API, DB, config, form and CSV workflows.
type DTOProfile struct{ Options []DTOOption }

var (
	DTOProfileLoose  = DTOProfile{}
	DTOProfileStrict = DTOProfile{Options: []DTOOption{WithDTOStrict()}}
	DTOProfileAPI    = DTOProfile{Options: []DTOOption{WithDTOStrict(), WithDTOFlatten(), WithDTOMaxDepth(48), WithDTOMaxSliceLen(100000)}}
	DTOProfileDB     = DTOProfile{Options: []DTOOption{WithDTOTags("db", "json", "convert"), WithDTOFlatten()}}
	DTOProfileConfig = DTOProfile{Options: []DTOOption{WithDTOTags("env", "json", "convert"), WithDTOFlatten()}}
	DTOProfileForm   = DTOProfile{Options: []DTOOption{WithDTOTags("form", "query", "json", "convert"), WithDTOFlatten()}}
	DTOProfileCSV    = DTOProfile{Options: []DTOOption{WithDTOTags("csv", "json", "convert")}}
)

func WithDTOProfile(p DTOProfile) DTOOption {
	return func(o *DTOOptions) {
		for _, opt := range p.Options {
			if opt != nil {
				opt(o)
			}
		}
	}
}

// DTOToReport converts src to T and returns non-fatal warnings such as defaulted fields and unknown inputs.
func DTOToReport[T any](src any, opts ...DTOOption) DTOReportResult[T] {
	var r DTOReportResult[T]
	o := dtoOptionsFrom(opts)
	warns := dtoAnalyzeWarnings(reflect.TypeOf(r.Value), src, o, "")
	err := DTO(&r.Value, src, opts...)
	r.Warnings = warns
	if err != nil {
		r.Errors = append(r.Errors, err)
	}
	return r
}

func DTOReport(dst any, src any, opts ...DTOOption) ([]DTOWarning, error) {
	o := dtoOptionsFrom(opts)
	warnings := dtoAnalyzeWarnings(indirectType(reflect.TypeOf(dst)), src, o, "")
	return warnings, DTO(dst, src, opts...)
}

func dtoAnalyzeWarnings(dst reflect.Type, src any, opt DTOOptions, path string) []DTOWarning {
	if dst == nil {
		return nil
	}
	for dst.Kind() == reflect.Pointer {
		dst = dst.Elem()
	}
	if dst.Kind() != reflect.Struct {
		return nil
	}
	m, ok := toStringAnyMap(src)
	if !ok {
		return nil
	}
	meta := dtoMetaFor(dst, opt)
	used := map[string]struct{}{}
	var out []DTOWarning
	for _, f := range meta.fields {
		_, found, name := dtoLookup(m, f, opt.TagPolicy.CaseInsensitive)
		if !found && opt.Flatten {
			if _, ok := lookupFlattenValue(m, f.names, opt.TagPolicy.CaseInsensitive); ok {
				found, name = true, f.primary
			}
		}
		if found {
			used[name] = struct{}{}
			continue
		}
		if f.defaultValue != "" {
			out = append(out, DTOWarning{Path: joinPath(path, f.primary), Code: "default", Message: "default value applied", Value: f.defaultValue})
		}
	}
	for k, v := range m {
		if _, ok := used[k]; !ok {
			out = append(out, DTOWarning{Path: joinPath(path, k), Code: "unknown", Message: "unknown input field", Value: v})
		}
	}
	return out
}

// DTOBatchOptions controls batch conversion behavior.
type DTOBatchOptions struct {
	CollectErrors bool
	SkipInvalid   bool
	Parallel      bool
	Workers       int
	DTOOptions    []DTOOption
}

type DTOBatchOption func(*DTOBatchOptions)

func DTOBatchCollectErrors() DTOBatchOption {
	return func(o *DTOBatchOptions) { o.CollectErrors = true }
}
func DTOBatchSkipInvalid() DTOBatchOption { return func(o *DTOBatchOptions) { o.SkipInvalid = true } }
func DTOBatchParallel(workers int) DTOBatchOption {
	return func(o *DTOBatchOptions) { o.Parallel = true; o.Workers = workers }
}
func DTOBatchWithOptions(opts ...DTOOption) DTOBatchOption {
	return func(o *DTOBatchOptions) { o.DTOOptions = append(o.DTOOptions, opts...) }
}

type DTOBatchReport[T any] struct {
	Values   []T
	Errors   []error
	Warnings []DTOWarning
}

func DTOBatch[T any](src any, opts ...DTOBatchOption) ([]T, error) {
	r := DTOBatchConvert[T](src, opts...)
	if len(r.Errors) > 0 {
		return r.Values, r.Errors[0]
	}
	return r.Values, nil
}

func DTOBatchConvert[T any](src any, opts ...DTOBatchOption) DTOBatchReport[T] {
	bo := DTOBatchOptions{Workers: 4}
	for _, opt := range opts {
		if opt != nil {
			opt(&bo)
		}
	}
	items, err := collectSliceItems(src, WithTrimSpace())
	if err != nil {
		return DTOBatchReport[T]{Errors: []error{err}}
	}
	out := DTOBatchReport[T]{Values: make([]T, len(items))}
	if !bo.Parallel || len(items) < 2 {
		for i, item := range items {
			v, e := DTOTo[T](item, bo.DTOOptions...)
			if e != nil {
				pe := PathError(indexPath("", i), KindOf(item), KindStruct, item, e)
				out.Errors = append(out.Errors, pe)
				if !bo.CollectErrors && !bo.SkipInvalid {
					out.Values = out.Values[:i]
					return out
				}
				if bo.SkipInvalid {
					continue
				}
			}
			out.Values[i] = v
		}
		return out
	}
	workers := bo.Workers
	if workers <= 0 {
		workers = 4
	}
	type job struct {
		i int
		v any
	}
	jobs := make(chan job)
	var mu sync.Mutex
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				v, e := DTOTo[T](j.v, bo.DTOOptions...)
				mu.Lock()
				if e != nil {
					out.Errors = append(out.Errors, PathError(indexPath("", j.i), KindOf(j.v), KindStruct, j.v, e))
				} else {
					out.Values[j.i] = v
				}
				mu.Unlock()
			}
		}()
	}
	for i, item := range items {
		jobs <- job{i, item}
	}
	close(jobs)
	wg.Wait()
	if bo.SkipInvalid && len(out.Errors) > 0 {
		clean := out.Values[:0]
		bad := map[int]struct{}{}
		for _, e := range out.Errors {
			if d, ok := e.(*ErrorDetail); ok {
				idx := parseLeadingIndex(d.Path)
				if idx >= 0 {
					bad[idx] = struct{}{}
				}
			}
		}
		for i, v := range out.Values {
			if _, ok := bad[i]; !ok {
				clean = append(clean, v)
			}
		}
		out.Values = clean
	}
	return out
}

// FlattenMap converts nested maps into dot-keyed maps.
func FlattenMap(src map[string]any) map[string]any {
	out := make(map[string]any)
	flattenInto(out, "", src)
	return out
}
func flattenInto(out map[string]any, prefix string, v any) {
	m, ok := toStringAnyMap(v)
	if !ok {
		if prefix != "" {
			out[prefix] = v
		}
		return
	}
	for k, val := range m {
		p := k
		if prefix != "" {
			p = prefix + "." + k
		}
		flattenInto(out, p, val)
	}
}

// UnflattenMap converts dot-keyed maps into nested maps.
func UnflattenMap(src map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range src {
		parts := strings.Split(k, ".")
		cur := out
		for i, p := range parts {
			if i == len(parts)-1 {
				cur[p] = v
				break
			}
			next, _ := cur[p].(map[string]any)
			if next == nil {
				next = map[string]any{}
				cur[p] = next
			}
			cur = next
		}
	}
	return out
}

func StructToFlatMap(src any, opts ...DTOOption) (map[string]any, error) {
	m, err := StructToMap(src, opts...)
	if err != nil {
		return nil, err
	}
	return FlattenMap(m), nil
}

// ApplyPatch deep-applies non-zero/non-nil patch fields into dst and returns changed paths.
func ApplyPatch(dst any, patch any, opts ...DTOOption) ([]string, error) {
	if dst == nil {
		return nil, ErrNil
	}
	dv := reflect.ValueOf(dst)
	if dv.Kind() != reflect.Pointer || dv.IsNil() {
		return nil, ErrUnsupported
	}
	if m, ok := toStringAnyMap(patch); ok {
		return applyPatchMap(dv.Elem(), m, "", dtoOptionsFrom(opts))
	}
	var pv reflect.Value
	if reflect.TypeOf(patch) != nil && reflect.TypeOf(patch).AssignableTo(dv.Elem().Type()) {
		pv = reflect.ValueOf(patch)
	} else {
		pv = reflect.New(dv.Elem().Type()).Elem()
		if err := DTO(pv.Addr().Interface(), patch, opts...); err != nil {
			return nil, err
		}
	}
	return applyPatchValue(dv.Elem(), indirectValue(pv), "")
}

func applyPatchMap(dst reflect.Value, m map[string]any, path string, opt DTOOptions) ([]string, error) {
	for dst.Kind() == reflect.Pointer {
		if dst.IsNil() {
			dst.Set(reflect.New(dst.Type().Elem()))
		}
		dst = dst.Elem()
	}
	if dst.Kind() != reflect.Struct {
		return nil, ErrUnsupported
	}
	meta := dtoMetaFor(dst.Type(), opt)
	var changed []string
	for _, f := range meta.fields {
		if f.readonly {
			continue
		}
		val, found, _ := dtoLookup(m, f, opt.TagPolicy.CaseInsensitive)
		if !found {
			if v, ok := lookupFlattenValue(m, f.names, opt.TagPolicy.CaseInsensitive); ok {
				val, found = v, true
			}
		}
		if !found {
			continue
		}
		field := fieldByIndex(dst, f.index)
		before := valueInterface(field)
		if err := dtoSet(field, applyDTOTransforms(val, f.transforms), joinPath(path, f.primary), opt); err != nil {
			return changed, err
		}
		if !reflect.DeepEqual(before, valueInterface(field)) {
			changed = append(changed, joinPath(path, f.primary))
		}
	}
	return changed, nil
}

func applyPatchValue(dst, patch reflect.Value, path string) ([]string, error) {
	if !patch.IsValid() {
		return nil, nil
	}
	for patch.Kind() == reflect.Pointer {
		if patch.IsNil() {
			return nil, nil
		}
		patch = patch.Elem()
	}
	if dst.Kind() == reflect.Pointer {
		if dst.IsNil() {
			dst.Set(reflect.New(dst.Type().Elem()))
		}
		dst = dst.Elem()
	}
	if dst.Kind() == reflect.Struct && patch.Kind() == reflect.Struct && dst.Type() == patch.Type() && dst.Type() != reflect.TypeOf(time.Time{}) {
		var changed []string
		meta := dtoMetaFor(dst.Type(), DefaultDTOOptions())
		for _, f := range meta.fields {
			if f.readonly {
				continue
			}
			df := fieldByIndex(dst, f.index)
			pf := fieldByIndex(patch, f.index)
			if !pf.IsValid() || pf.IsZero() {
				continue
			}
			p := joinPath(path, f.primary)
			if df.Kind() == reflect.Struct && pf.Kind() == reflect.Struct && df.Type() != reflect.TypeOf(time.Time{}) {
				c, err := applyPatchValue(df, pf, p)
				if err != nil {
					return changed, err
				}
				changed = append(changed, c...)
				continue
			}
			if !reflect.DeepEqual(valueInterface(df), valueInterface(pf)) {
				df.Set(pf)
				changed = append(changed, p)
			}
		}
		return changed, nil
	}
	if dst.CanSet() && patch.Type().AssignableTo(dst.Type()) && !reflect.DeepEqual(valueInterface(dst), valueInterface(patch)) {
		dst.Set(patch)
		return []string{path}, nil
	}
	return nil, nil
}

type DiffChange struct {
	Path string
	Old  any
	New  any
	Kind string
}

func Diff(a, b any, opts ...DTOOption) []DiffChange {
	return diffValue(reflect.ValueOf(a), reflect.ValueOf(b), "")
}
func diffValue(a, b reflect.Value, path string) []DiffChange {
	if !a.IsValid() || !b.IsValid() {
		if a.IsValid() != b.IsValid() {
			return []DiffChange{{Path: path, Old: valueInterface(a), New: valueInterface(b), Kind: "change"}}
		}
		return nil
	}
	for a.Kind() == reflect.Pointer {
		if a.IsNil() {
			break
		}
		a = a.Elem()
	}
	for b.Kind() == reflect.Pointer {
		if b.IsNil() {
			break
		}
		b = b.Elem()
	}
	if a.IsValid() && b.IsValid() && a.Kind() == reflect.Struct && b.Kind() == reflect.Struct && a.Type() == b.Type() && a.Type() != reflect.TypeOf(time.Time{}) {
		var out []DiffChange
		meta := dtoMetaFor(a.Type(), DefaultDTOOptions())
		for _, f := range meta.fields {
			out = append(out, diffValue(fieldByIndex(a, f.index), fieldByIndex(b, f.index), joinPath(path, f.primary))...)
		}
		return out
	}
	if !reflect.DeepEqual(valueInterface(a), valueInterface(b)) {
		return []DiffChange{{Path: path, Old: valueInterface(a), New: valueInterface(b), Kind: "change"}}
	}
	return nil
}

func Merge[T any](base T, overlays ...any) (T, error) {
	out := base
	for _, ov := range overlays {
		if _, err := ApplyPatch(&out, ov); err != nil {
			return out, err
		}
	}
	return out, nil
}

// Schema is a lightweight schema generated from DTO tags.
type Schema struct {
	Type     string        `json:"type"`
	Fields   []SchemaField `json:"fields,omitempty"`
	Required []string      `json:"required,omitempty"`
}
type SchemaField struct {
	Name       string        `json:"name"`
	Type       string        `json:"type"`
	Required   bool          `json:"required,omitempty"`
	Default    string        `json:"default,omitempty"`
	Sensitive  bool          `json:"sensitive,omitempty"`
	ReadOnly   bool          `json:"readonly,omitempty"`
	WriteOnly  bool          `json:"writeonly,omitempty"`
	Validation string        `json:"validation,omitempty"`
	Elem       *SchemaField  `json:"elem,omitempty"`
	Fields     []SchemaField `json:"fields,omitempty"`
}

func SchemaOf[T any](opts ...DTOOption) Schema { var z T; return SchemaFor(reflect.TypeOf(z), opts...) }
func SchemaFor(t reflect.Type, opts ...DTOOption) Schema {
	o := dtoOptionsFrom(opts)
	t = indirectType(t)
	s := Schema{Type: schemaType(t)}
	if t == nil || t.Kind() != reflect.Struct {
		return s
	}
	meta := dtoMetaFor(t, o)
	for _, f := range meta.fields {
		sf := schemaFieldFor(f)
		s.Fields = append(s.Fields, sf)
		if sf.Required {
			s.Required = append(s.Required, sf.Name)
		}
	}
	return s
}
func schemaFieldFor(f dtoFieldMeta) SchemaField {
	ft := indirectType(f.structField.Type)
	sf := SchemaField{Name: f.primary, Type: schemaType(ft), Required: f.required, Default: f.defaultValue, Sensitive: f.sensitive, ReadOnly: f.readonly, WriteOnly: f.writeonly, Validation: f.structField.Tag.Get("validate")}
	if ft != nil && (ft.Kind() == reflect.Slice || ft.Kind() == reflect.Array) {
		et := indirectType(ft.Elem())
		sf.Elem = &SchemaField{Type: schemaType(et)}
	}
	if ft != nil && ft.Kind() == reflect.Struct && ft != reflect.TypeOf(time.Time{}) {
		sub := SchemaFor(ft)
		sf.Fields = sub.Fields
	}
	return sf
}
func schemaType(t reflect.Type) string {
	if t == nil {
		return "any"
	}
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t == reflect.TypeOf(time.Time{}) {
		return "time"
	}
	if t == reflect.TypeOf(time.Duration(0)) {
		return "duration"
	}
	switch t.Kind() {
	case reflect.Bool:
		return "bool"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "integer"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return "unsigned"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.String:
		return "string"
	case reflect.Slice, reflect.Array:
		return "array"
	case reflect.Map:
		return "object"
	case reflect.Struct:
		return "object"
	default:
		return "any"
	}
}

func FromQuery[T any](v url.Values, opts ...DTOOption) (T, error) {
	return DTOTo[T](valuesToMap(v), append([]DTOOption{WithDTOTags("query", "form", "json", "convert")}, opts...)...)
}
func FromForm[T any](v url.Values, opts ...DTOOption) (T, error) {
	return DTOTo[T](valuesToMap(v), append([]DTOOption{WithDTOTags("form", "query", "json", "convert")}, opts...)...)
}
func FromHeaders[T any](h http.Header, opts ...DTOOption) (T, error) {
	m := map[string]any{}
	for k, v := range h {
		if len(v) == 1 {
			m[k] = v[0]
		} else {
			m[k] = v
		}
	}
	return DTOTo[T](m, append([]DTOOption{WithDTOTags("header", "json", "convert")}, opts...)...)
}
func FromEnv[T any](opts ...DTOOption) (T, error) {
	m := map[string]any{}
	for _, e := range os.Environ() {
		k, v, _ := strings.Cut(e, "=")
		m[k] = v
	}
	return DTOTo[T](m, append([]DTOOption{WithDTOTags("env", "json", "convert")}, opts...)...)
}
func FromCSVRow[T any](headers []string, row []string, opts ...DTOOption) (T, error) {
	m := map[string]any{}
	for i, h := range headers {
		if i < len(row) {
			m[h] = row[i]
		}
	}
	return DTOTo[T](m, append([]DTOOption{WithDTOTags("csv", "json", "convert")}, opts...)...)
}

func ToQuery(src any, opts ...DTOOption) (url.Values, error) {
	m, err := StructToFlatMap(src, opts...)
	if err != nil {
		return nil, err
	}
	q := url.Values{}
	for k, v := range m {
		if v == nil {
			continue
		}
		q.Set(k, ToDebugString(v))
	}
	return q, nil
}
func ToHeaders(src any, opts ...DTOOption) (http.Header, error) {
	m, err := StructToMap(src, append([]DTOOption{WithDTOTags("header", "json", "convert")}, opts...)...)
	if err != nil {
		return nil, err
	}
	h := http.Header{}
	for k, v := range m {
		if v != nil {
			h.Set(k, ToDebugString(v))
		}
	}
	return h, nil
}
func ToEnv(src any, opts ...DTOOption) (map[string]string, error) {
	m, err := StructToMap(src, append([]DTOOption{WithDTOTags("env", "json", "convert")}, opts...)...)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for k, v := range m {
		if v != nil {
			out[k] = ToDebugString(v)
		}
	}
	return out, nil
}

func Redact(src any, opts ...DTOOption) (map[string]any, error) {
	o := dtoOptionsFrom(opts)
	rv := reflect.ValueOf(src)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil, ErrNil
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, ErrUnsupported
	}
	meta := dtoMetaFor(rv.Type(), o)
	out := map[string]any{}
	for _, f := range meta.fields {
		if f.writeonly {
			continue
		}
		fv := fieldByIndex(rv, f.index)
		if !fv.IsValid() || !fv.CanInterface() {
			continue
		}
		if f.sensitive {
			out[f.primary] = "[REDACTED]"
		} else {
			out[f.primary] = fv.Interface()
		}
	}
	return out, nil
}

// DTOStreamJSONL reads JSON lines, converts each record, and writes JSON lines.
func DTOStreamJSONL[T any](ctx context.Context, r io.Reader, w io.Writer, opts ...DTOOption) error {
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	enc := json.NewEncoder(w)
	for s.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		line := strings.TrimSpace(s.Text())
		if line == "" {
			continue
		}
		var m any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			return err
		}
		v, err := DTOTo[T](m, opts...)
		if err != nil {
			return err
		}
		if err := enc.Encode(v); err != nil {
			return err
		}
	}
	return s.Err()
}

// Type-pair converter registry for hot domain mappings.
type PairConverter[S any, T any] func(S) (T, error)
type pairKey struct{ s, t reflect.Type }

var pairConverters sync.Map

func RegisterPair[S any, T any](fn PairConverter[S, T]) {
	var s S
	var t T
	pairConverters.Store(pairKey{reflect.TypeOf(s), reflect.TypeOf(t)}, fn)
}
func UnregisterPair[S any, T any]() {
	var s S
	var t T
	pairConverters.Delete(pairKey{reflect.TypeOf(s), reflect.TypeOf(t)})
}
func DTOConvertPair[S any, T any](src S, opts ...DTOOption) (T, error) {
	var z T
	if fn, ok := pairConverters.Load(pairKey{reflect.TypeOf(src), reflect.TypeOf(z)}); ok {
		return fn.(PairConverter[S, T])(src)
	}
	return DTOTo[T](src, opts...)
}

func dtoTagOptions(f reflect.StructField) (transforms []string, readonly, writeonly, sensitive bool) {
	for _, tag := range []string{"convert", "json", "env", "query", "form", "header", "csv"} {
		raw := f.Tag.Get(tag)
		if raw == "" {
			continue
		}
		parts := strings.Split(raw, ",")
		for _, p := range parts[1:] {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			switch {
			case p == "trim" || p == "lower" || p == "upper" || p == "title" || p == "snake" || p == "camel" || p == "kebab":
				transforms = append(transforms, p)
			case p == "readonly":
				readonly = true
			case p == "writeonly":
				writeonly = true
			case p == "sensitive" || p == "secret":
				sensitive = true
			case strings.HasPrefix(p, "source=") || strings.HasPrefix(p, "alias="):
			}
		}
	}
	if f.Tag.Get("sensitive") == "true" || f.Tag.Get("secret") == "true" {
		sensitive = true
	}
	return
}

func applyDTOTransforms(v any, transforms []string) any {
	if len(transforms) == 0 {
		return v
	}
	s, ok := v.(string)
	if !ok {
		return v
	}
	for _, tr := range transforms {
		switch tr {
		case "trim":
			s = strings.TrimSpace(s)
		case "lower":
			s = strings.ToLower(s)
		case "upper":
			s = strings.ToUpper(s)
		case "title":
			s = strings.Title(strings.ToLower(s))
		case "snake":
			s = snakeName(s)
		case "kebab":
			s = strings.ReplaceAll(snakeName(s), "_", "-")
		case "camel":
			s = toCamel(s)
		}
	}
	return s
}
func toCamel(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == '_' || r == '-' || r == ' ' })
	if len(parts) == 0 {
		return s
	}
	for i := 1; i < len(parts); i++ {
		if parts[i] != "" {
			parts[i] = strings.ToUpper(parts[i][:1]) + strings.ToLower(parts[i][1:])
		}
	}
	parts[0] = strings.ToLower(parts[0])
	return strings.Join(parts, "")
}

func lookupFlattenValue(m map[string]any, names []string, ci bool) (any, bool) {
	flat := FlattenMap(m)
	for _, n := range names {
		if v, ok := lookupName(flat, n, ci); ok {
			return v, true
		}
	}
	return nil, false
}
func dtoPathDepth(path string) int {
	if path == "" {
		return 0
	}
	n := 1
	for _, r := range path {
		if r == '.' || r == '[' {
			n++
		}
	}
	return n
}
func toStringAnyMap(src any) (map[string]any, bool) {
	if src == nil {
		return nil, false
	}
	switch x := src.(type) {
	case map[string]any:
		return x, true
	case map[string]string:
		m := map[string]any{}
		for k, v := range x {
			m[k] = v
		}
		return m, true
	}
	rv := reflect.ValueOf(src)
	for rv.IsValid() && rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil, false
		}
		rv = rv.Elem()
	}
	if !rv.IsValid() {
		return nil, false
	}
	if rv.Kind() == reflect.Map {
		m := map[string]any{}
		for _, k := range rv.MapKeys() {
			ks, err := ToString(k.Interface())
			if err == nil {
				m[ks] = rv.MapIndex(k).Interface()
			}
		}
		return m, true
	}
	return nil, false
}
func valuesToMap(v url.Values) map[string]any {
	m := map[string]any{}
	for k, vv := range v {
		if len(vv) == 1 {
			m[k] = vv[0]
		} else {
			m[k] = vv
		}
	}
	return m
}
func indirectType(t reflect.Type) reflect.Type {
	if t == nil {
		return nil
	}
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}
func indirectValue(v reflect.Value) reflect.Value {
	for v.IsValid() && v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return v
		}
		v = v.Elem()
	}
	return v
}
func valueInterface(v reflect.Value) any {
	if !v.IsValid() {
		return nil
	}
	if v.CanInterface() {
		return v.Interface()
	}
	return nil
}
func parseLeadingIndex(p string) int {
	if !strings.HasPrefix(p, "[") {
		return -1
	}
	end := strings.IndexByte(p, ']')
	if end < 0 {
		return -1
	}
	i, _ := strconv.Atoi(p[1:end])
	return i
}

// Flatten/unflatten-friendly lookup names from convert tag source=/alias= options.
func dtoExtendedNames(f reflect.StructField) []string {
	var names []string
	for _, tag := range []string{"convert", "json", "env", "query", "form", "header", "csv"} {
		raw := f.Tag.Get(tag)
		if raw == "" {
			continue
		}
		for _, p := range strings.Split(raw, ",")[1:] {
			p = strings.TrimSpace(p)
			if strings.HasPrefix(p, "source=") || strings.HasPrefix(p, "alias=") {
				_, rhs, _ := strings.Cut(p, "=")
				for _, n := range strings.Split(rhs, "|") {
					names = appendUniqueString(names, strings.TrimSpace(n))
				}
			}
		}
	}
	return names
}

// StableJSONSchema returns an indented JSON representation of SchemaOf[T].
func StableJSONSchema[T any](opts ...DTOOption) string {
	b, _ := json.MarshalIndent(SchemaOf[T](opts...), "", "  ")
	return string(b)
}

// SortedDiff returns Diff sorted by path for deterministic tests/reports.
func SortedDiff(a, b any) []DiffChange {
	d := Diff(a, b)
	sort.Slice(d, func(i, j int) bool { return d[i].Path < d[j].Path })
	return d
}

var ErrStream = errors.New("convert: stream conversion failed")

func errf(format string, args ...any) error { return fmt.Errorf(format, args...) }

// Plugin installs related converters/hooks into the package-level registries.
type Plugin interface{ Register() error }
type PluginFunc func() error

func (f PluginFunc) Register() error { return f() }
func Use(plugins ...Plugin) error {
	for _, p := range plugins {
		if p == nil {
			continue
		}
		if err := p.Register(); err != nil {
			return err
		}
	}
	return nil
}

// DTOPlan is a reusable compiled conversion facade for a source/destination pair.
type DTOPlan[S any, T any] struct{ opts []DTOOption }

func CompileDTOPlan[S any, T any](opts ...DTOOption) DTOPlan[S, T] {
	return DTOPlan[S, T]{opts: append([]DTOOption(nil), opts...)}
}
func (p DTOPlan[S, T]) Convert(src S) (T, error) { return DTOConvertPair[S, T](src, p.opts...) }
func (p DTOPlan[S, T]) ConvertSlice(src []S) ([]T, error) {
	out := make([]T, len(src))
	for i, v := range src {
		x, err := p.Convert(v)
		if err != nil {
			return out, PathError(indexPath("", i), KindOf(v), KindInvalid, v, err)
		}
		out[i] = x
	}
	return out, nil
}
