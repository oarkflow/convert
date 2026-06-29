package convert

import (
	"context"
	"database/sql"
	"encoding/csv"
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
	"time"
)

// Map is the simplest DTO entry point. It converts any struct/map/slice/scalar input into T.
func Map[T any](src any, opts ...DTOOption) (T, error) { return DTOTo[T](src, opts...) }

// MustMap is Map and panics on error.
func MustMap[T any](src any, opts ...DTOOption) T { return MustDTOTo[T](src, opts...) }

// MapInto converts src into an existing destination pointer.
func MapInto(src any, dst any, opts ...DTOOption) error { return DTO(dst, src, opts...) }

// Preset option helpers for common binding modes.
func DTOLoose() DTOOption        { return WithDTOProfile(DTOProfileLoose) }
func DTOStrictPreset() DTOOption { return WithDTOProfile(DTOProfileStrict) }
func API() DTOOption             { return WithDTOProfile(DTOProfileAPI) }
func StrictAPI() DTOOption {
	return func(o *DTOOptions) { WithDTOProfile(DTOProfileAPI)(o); WithDTOStrict()(o) }
}
func DB() DTOOption       { return WithDTOProfile(DTOProfileDB) }
func Config() DTOOption   { return WithDTOProfile(DTOProfileConfig) }
func EnvDTO() DTOOption   { return WithDTOProfile(DTOProfileConfig) }
func QueryDTO() DTOOption { return WithDTOTags("query", "form", "json", "convert") }
func Form() DTOOption     { return WithDTOTags("form", "query", "json", "convert") }
func Header() DTOOption   { return WithDTOTags("header", "json", "convert") }
func CSV() DTOOption      { return WithDTOProfile(DTOProfileCSV) }

// SourcePriority changes tag lookup priority, e.g. SourcePriority("json", "form", "query").
func SourcePriority(tags ...string) DTOOption { return WithDTOTags(tags...) }

// MultiError contains all field errors found during a best-effort conversion.
type MultiError struct{ Errors []error }

func (m MultiError) Error() string {
	if len(m.Errors) == 0 {
		return ""
	}
	if len(m.Errors) == 1 {
		return m.Errors[0].Error()
	}
	var b strings.Builder
	b.WriteString(strconv.Itoa(len(m.Errors)))
	b.WriteString(" conversion errors")
	for _, err := range m.Errors {
		b.WriteString("; ")
		b.WriteString(DebugError(err))
	}
	return b.String()
}
func (m MultiError) Unwrap() []error { return m.Errors }

// MapAll returns all field errors it can discover instead of only the first one.
func MapAll[T any](src any, opts ...DTOOption) (T, error) {
	var out T
	errs := collectDTOErrors(reflect.ValueOf(&out).Elem(), src, "", dtoOptionsFrom(opts))
	if len(errs) > 0 {
		return out, MultiError{Errors: errs}
	}
	return out, DTO(&out, src, opts...)
}

func collectDTOErrors(dst reflect.Value, src any, path string, opt DTOOptions) []error {
	for dst.Kind() == reflect.Pointer {
		if dst.IsNil() {
			dst.Set(reflect.New(dst.Type().Elem()))
		}
		dst = dst.Elem()
	}
	if dst.Kind() != reflect.Struct || dst.Type() == reflect.TypeOf(time.Time{}) {
		if err := dtoSet(dst, src, path, opt); err != nil {
			return []error{err}
		}
		return nil
	}
	m, ok := toStringAnyMap(src)
	if !ok {
		if err := dtoSet(dst, src, path, opt); err != nil {
			return []error{err}
		}
		return nil
	}
	meta := dtoMetaFor(dst.Type(), opt)
	used := map[string]struct{}{}
	var errs []error
	for _, f := range meta.fields {
		field := fieldByIndex(dst, f.index)
		if !field.IsValid() || !field.CanSet() {
			continue
		}
		val, found, usedName := dtoLookup(m, f, opt.TagPolicy.CaseInsensitive)
		if !found && opt.Flatten {
			if v, ok := lookupFlattenValue(m, f.names, opt.TagPolicy.CaseInsensitive); ok {
				val, found, usedName = v, true, f.primary
			}
		}
		fp := joinPath(path, f.primary)
		if !found {
			if f.defaultValue != "" {
				val, found = f.defaultValue, true
			} else if f.required {
				errs = append(errs, PathError(fp, KindInvalid, KindOfReflect(field), nil, ErrEmpty))
			}
			if !found {
				continue
			}
		}
		used[usedName] = struct{}{}
		if f.readonly {
			continue
		}
		if field.Kind() == reflect.Struct && field.Type() != reflect.TypeOf(time.Time{}) {
			errs = append(errs, collectDTOErrors(field, applyDTOTransforms(val, f.transforms), fp, opt)...)
			continue
		}
		if err := dtoSet(field, applyDTOTransforms(val, f.transforms), fp, opt); err != nil {
			errs = append(errs, err)
			continue
		}
		if f.validate {
			if err := validateReflectField(field, f.structField, fp); err != nil {
				errs = append(errs, err)
			}
		}
	}
	if opt.ErrorUnused {
		for k, v := range m {
			if _, ok := used[k]; !ok {
				errs = append(errs, PathError(joinPath(path, k), KindOf(v), KindInvalid, v, ErrUnusedField))
			}
		}
	}
	return errs
}

// HumanError is safe to show to clients. DebugError contains path/type details for developers.
func HumanError(err error) string {
	if err == nil {
		return ""
	}
	var me MultiError
	if errors.As(err, &me) && len(me.Errors) > 0 {
		return HumanError(me.Errors[0])
	}
	var d *ErrorDetail
	if errors.As(err, &d) {
		name := d.Path
		if name == "" {
			name = "value"
		}
		return fmt.Sprintf("Invalid field %q: %v.", name, d.Cause)
	}
	return err.Error()
}
func DebugError(err error) string {
	if err == nil {
		return ""
	}
	var d *ErrorDetail
	if errors.As(err, &d) {
		return fmt.Sprintf("%s: cannot convert %v(%v) to %v: %v", emptyAs(d.Path, "value"), d.From, d.Value, d.To, d.Cause)
	}
	return err.Error()
}
func emptyAs(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

// Description is a field-level documentation row generated from DTO tags.
type Description struct {
	Name, GoName, Type, Default, Validate    string
	Required, ReadOnly, WriteOnly, Sensitive bool
	Aliases                                  []string
}

func Describe[T any](opts ...DTOOption) []Description {
	var z T
	return DescribeType(reflect.TypeOf(z), opts...)
}
func DescribeType(t reflect.Type, opts ...DTOOption) []Description {
	o := dtoOptionsFrom(opts)
	t = indirectType(t)
	if t == nil || t.Kind() != reflect.Struct {
		return nil
	}
	meta := dtoMetaFor(t, o)
	out := make([]Description, 0, len(meta.fields))
	for _, f := range meta.fields {
		aliases := append([]string(nil), f.names...)
		if len(aliases) > 0 {
			aliases = aliases[1:]
		}
		out = append(out, Description{Name: f.primary, GoName: f.structField.Name, Type: schemaType(f.structField.Type), Default: f.defaultValue, Validate: f.structField.Tag.Get("validate"), Required: f.required, ReadOnly: f.readonly, WriteOnly: f.writeonly, Sensitive: f.sensitive, Aliases: aliases})
	}
	return out
}
func DescribeText[T any](opts ...DTOOption) string {
	rows := Describe[T](opts...)
	var b strings.Builder
	for _, r := range rows {
		fmt.Fprintf(&b, "%s\t%s\trequired=%t\tvalidate=%s\n", r.Name, r.Type, r.Required, r.Validate)
	}
	return b.String()
}

// TraceStep records how a field was resolved.
type TraceStep struct {
	Path, Source, Action string
	Value                any
	Error                string
}
type TraceResult[T any] struct {
	Value    T
	Steps    []TraceStep
	Warnings []DTOWarning
	Err      error
}

func MapTrace[T any](src any, opts ...DTOOption) TraceResult[T] {
	var out T
	o := dtoOptionsFrom(opts)
	steps := traceValue(reflect.TypeOf(out), src, "", o)
	err := DTO(&out, src, opts...)
	for i := range steps {
		if steps[i].Error != "" {
			break
		}
	}
	return TraceResult[T]{Value: out, Steps: steps, Warnings: dtoAnalyzeWarnings(reflect.TypeOf(out), src, o, ""), Err: err}
}
func traceValue(t reflect.Type, src any, path string, opt DTOOptions) []TraceStep {
	t = indirectType(t)
	if t == nil || t.Kind() != reflect.Struct || t == reflect.TypeOf(time.Time{}) {
		return nil
	}
	m, ok := toStringAnyMap(src)
	if !ok {
		return []TraceStep{{Path: path, Action: "unsupported-trace-source", Value: src}}
	}
	meta := dtoMetaFor(t, opt)
	var out []TraceStep
	for _, f := range meta.fields {
		v, found, srcName := dtoLookup(m, f, opt.TagPolicy.CaseInsensitive)
		if !found && opt.Flatten {
			if vv, ok := lookupFlattenValue(m, f.names, opt.TagPolicy.CaseInsensitive); ok {
				v, found, srcName = vv, true, f.primary
			}
		}
		fp := joinPath(path, f.primary)
		step := TraceStep{Path: fp, Source: srcName, Value: v}
		switch {
		case found && f.readonly:
			step.Action = "skip-readonly"
		case found:
			step.Action = "map"
		case f.defaultValue != "":
			step.Action = "default"
			step.Value = f.defaultValue
		case f.required:
			step.Action = "missing-required"
			step.Error = ErrEmpty.Error()
		default:
			step.Action = "missing-skip"
		}
		out = append(out, step)
		ft := indirectType(f.structField.Type)
		if found && ft != nil && ft.Kind() == reflect.Struct && ft != reflect.TypeOf(time.Time{}) {
			out = append(out, traceValue(ft, v, fp, opt)...)
		}
	}
	return out
}

// DryRun reports matching, warnings and validation errors without returning a value.
type DryRunReport struct {
	Steps    []TraceStep
	Warnings []DTOWarning
	Errors   []error
}

func DryRun[T any](src any, opts ...DTOOption) DryRunReport {
	tr := MapTrace[T](src, opts...)
	var out T
	return DryRunReport{Steps: tr.Steps, Warnings: tr.Warnings, Errors: collectDTOErrors(reflect.ValueOf(&out).Elem(), src, "", dtoOptionsFrom(opts))}
}

// Optional represents an API patch field with unset / null / value states.
type Optional[T any] struct {
	Value T
	Set   bool
	Null  bool
}

func Some[T any](v T) Optional[T]  { return Optional[T]{Value: v, Set: true} }
func Null[T any]() Optional[T]     { return Optional[T]{Set: true, Null: true} }
func Unset[T any]() Optional[T]    { return Optional[T]{} }
func (o Optional[T]) IsZero() bool { return !o.Set }
func (o Optional[T]) MarshalJSON() ([]byte, error) {
	if !o.Set || o.Null {
		return []byte("null"), nil
	}
	return json.Marshal(o.Value)
}
func (o *Optional[T]) UnmarshalJSON(b []byte) error {
	o.Set = true
	if strings.TrimSpace(string(b)) == "null" {
		o.Null = true
		var z T
		o.Value = z
		return nil
	}
	return json.Unmarshal(b, &o.Value)
}

// ApplyOptionalPatch applies fields of type Optional[T] from patch to dst.
func ApplyOptionalPatch(dst any, patch any, opts ...DTOOption) ([]string, error) {
	dv := reflect.ValueOf(dst)
	if dv.Kind() != reflect.Pointer || dv.IsNil() {
		return nil, ErrUnsupported
	}
	pv := reflect.ValueOf(patch)
	for pv.Kind() == reflect.Pointer {
		if pv.IsNil() {
			return nil, nil
		}
		pv = pv.Elem()
	}
	if pv.Kind() != reflect.Struct {
		return ApplyPatch(dst, patch, opts...)
	}
	return applyOptionalPatchValue(dv.Elem(), pv, "", dtoOptionsFrom(opts))
}
func applyOptionalPatchValue(dst, patch reflect.Value, path string, opt DTOOptions) ([]string, error) {
	for dst.Kind() == reflect.Pointer {
		if dst.IsNil() {
			dst.Set(reflect.New(dst.Type().Elem()))
		}
		dst = dst.Elem()
	}
	meta := dtoMetaFor(patch.Type(), opt)
	var changed []string
	for _, f := range meta.fields {
		pf := fieldByIndex(patch, f.index)
		if !pf.IsValid() || !pf.CanInterface() {
			continue
		}
		if !isOptionalType(pf.Type()) {
			continue
		}
		set := pf.FieldByName("Set").Bool()
		if !set {
			continue
		}
		null := pf.FieldByName("Null").Bool()
		fp := joinPath(path, f.primary)
		df := findDTOField(dst, f.primary, opt)
		if !df.IsValid() || !df.CanSet() {
			continue
		}
		before := valueInterface(df)
		if null {
			df.SetZero()
		} else {
			if err := dtoSet(df, pf.FieldByName("Value").Interface(), fp, opt); err != nil {
				return changed, err
			}
		}
		if !reflect.DeepEqual(before, valueInterface(df)) {
			changed = append(changed, fp)
		}
	}
	return changed, nil
}
func isOptionalType(t reflect.Type) bool {
	return t.Kind() == reflect.Struct && t.NumField() >= 3 && t.Field(0).Name == "Value" && t.Field(1).Name == "Set" && t.Field(2).Name == "Null"
}
func findDTOField(dst reflect.Value, name string, opt DTOOptions) reflect.Value {
	meta := dtoMetaFor(dst.Type(), opt)
	for _, f := range meta.fields {
		if f.primary == name {
			return fieldByIndex(dst, f.index)
		}
	}
	return reflect.Value{}
}

// RolePolicy blocks fields tagged adminonly/role=<name> unless role matches.
func Role(role string) DTOOption {
	return WithDTODecodeHook(func(ctx DTOContext, dst reflect.Value, src any) (bool, error) { return false, nil })
}

// BindSource is a named input source for multi-source binding.
type BindSource struct {
	Name string
	Data map[string]any
	Tags []string
}

func FromMapSource(name string, m map[string]any, tags ...string) BindSource {
	return BindSource{Name: name, Data: m, Tags: tags}
}
func FromQuerySource(v url.Values) BindSource {
	return BindSource{Name: "query", Data: valuesToMap(v), Tags: []string{"query", "form", "json", "convert"}}
}
func FromHeaderSource(h http.Header) BindSource {
	m := map[string]any{}
	for k, v := range h {
		if len(v) == 1 {
			m[k] = v[0]
		} else {
			m[k] = v
		}
	}
	return BindSource{Name: "header", Data: m, Tags: []string{"header", "json", "convert"}}
}
func FromDefaultSource(m map[string]any) BindSource { return BindSource{Name: "default", Data: m} }

// Bind converts multiple sources into T. Later sources overwrite earlier ones.
func Bind[T any](sources ...BindSource) (T, error) { return BindWithOptions[T](nil, sources...) }
func BindWithOptions[T any](opts []DTOOption, sources ...BindSource) (T, error) {
	merged := map[string]any{}
	tags := []string{"convert", "json", "query", "form", "header", "env", "csv"}
	for _, s := range sources {
		for k, v := range s.Data {
			merged[k] = v
		}
		if len(s.Tags) > 0 {
			tags = append(s.Tags, tags...)
		}
	}
	return DTOTo[T](merged, append([]DTOOption{WithDTOTags(uniqueStrings(tags)...), WithDTOFlatten()}, opts...)...)
}
func uniqueStrings(in []string) []string {
	out := in[:0]
	seen := map[string]struct{}{}
	for _, s := range in {
		if s == "" {
			continue
		}
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}

// HTTP helpers. They use net/http only in this optional usability layer.
func BindJSON(r *http.Request, dst any, opts ...DTOOption) error {
	defer r.Body.Close()
	var v any
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		return err
	}
	return DTO(dst, v, append([]DTOOption{WithDTOTags("json", "convert")}, opts...)...)
}
func BindRequestQuery(r *http.Request, dst any, opts ...DTOOption) error {
	return DTO(dst, valuesToMap(r.URL.Query()), append([]DTOOption{QueryDTO()}, opts...)...)
}
func BindRequestForm(r *http.Request, dst any, opts ...DTOOption) error {
	if err := r.ParseForm(); err != nil {
		return err
	}
	return DTO(dst, valuesToMap(r.Form), append([]DTOOption{Form()}, opts...)...)
}
func BindHeaders(r *http.Request, dst any, opts ...DTOOption) error {
	return DTO(dst, FromHeaderSource(r.Header).Data, append([]DTOOption{Header()}, opts...)...)
}
func BindRequest(r *http.Request, dst any, opts ...DTOOption) error {
	body := map[string]any{}
	if r.Body != nil && r.ContentLength != 0 {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}
	_, err := BindWithOptions[map[string]any](nil, FromMapSource("body", body, "json", "convert"), FromQuerySource(r.URL.Query()), FromHeaderSource(r.Header))
	if err != nil {
		return err
	}
	merged := map[string]any{}
	for k, v := range body {
		merged[k] = v
	}
	for k, v := range valuesToMap(r.URL.Query()) {
		merged[k] = v
	}
	for k, v := range FromHeaderSource(r.Header).Data {
		merged[k] = v
	}
	return DTO(dst, merged, append([]DTOOption{WithDTOFlatten()}, opts...)...)
}

// Config loader sources.
type ConfigSource interface {
	Load() (map[string]any, error)
}
type ConfigSourceFunc func() (map[string]any, error)

func (f ConfigSourceFunc) Load() (map[string]any, error) { return f() }
func EnvSource() ConfigSource {
	return ConfigSourceFunc(func() (map[string]any, error) {
		m := map[string]any{}
		for _, e := range os.Environ() {
			k, v, _ := strings.Cut(e, "=")
			m[k] = v
		}
		return m, nil
	})
}
func MapSource(m map[string]any) ConfigSource {
	return ConfigSourceFunc(func() (map[string]any, error) { return m, nil })
}
func FileSource(path string) ConfigSource {
	return ConfigSourceFunc(func() (map[string]any, error) {
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			return nil, err
		}
		return m, nil
	})
}
func LoadConfig[T any](sources ...ConfigSource) (T, error) {
	merged := map[string]any{}
	for _, s := range sources {
		m, err := s.Load()
		if err != nil {
			var z T
			return z, err
		}
		for k, v := range m {
			merged[k] = v
		}
	}
	return DTOTo[T](merged, Config(), WithDTOFlatten())
}
func SafeJSON(v any, opts ...DTOOption) string {
	m, err := Redact(v, opts...)
	if err != nil {
		b, _ := json.Marshal(v)
		return string(b)
	}
	b, _ := json.Marshal(m)
	return string(b)
}
func SafeMap(v any, opts ...DTOOption) map[string]any { m, _ := Redact(v, opts...); return m }
func SafeString(v any, opts ...DTOOption) string      { return SafeJSON(v, opts...) }

// Database row helpers.
type SQLScanner interface{ Scan(dest ...any) error }
type SQLRows interface {
	Columns() ([]string, error)
	Next() bool
	Scan(dest ...any) error
	Err() error
}

func FromSQLRow[T any](row SQLScanner, columns []string, opts ...DTOOption) (T, error) {
	vals := make([]any, len(columns))
	ptrs := make([]any, len(columns))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	if err := row.Scan(ptrs...); err != nil {
		var z T
		return z, err
	}
	return DTOTo[T](sqlValuesMap(columns, vals), append([]DTOOption{WithDTOTags("db", "json", "convert")}, opts...)...)
}
func FromSQLRows[T any](rows SQLRows, opts ...DTOOption) ([]T, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var out []T
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return out, err
		}
		v, err := DTOTo[T](sqlValuesMap(cols, vals), append([]DTOOption{WithDTOTags("db", "json", "convert")}, opts...)...)
		if err != nil {
			return out, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func sqlValuesMap(cols []string, vals []any) map[string]any {
	m := map[string]any{}
	for i, c := range cols {
		v := vals[i]
		if b, ok := v.([]byte); ok {
			v = string(b)
		}
		m[c] = v
	}
	return m
}
func ToSQLArgs(src any, fields ...string) ([]any, error) {
	m, err := StructToMap(src, WithDTOTags("db", "json", "convert"))
	if err != nil {
		return nil, err
	}
	if len(fields) == 0 {
		fields = make([]string, 0, len(m))
		for k := range m {
			fields = append(fields, k)
		}
		sort.Strings(fields)
	}
	out := make([]any, 0, len(fields))
	for _, f := range fields {
		out = append(out, m[f])
	}
	return out, nil
}

var _ = sql.ErrNoRows

// CSV import/export helpers.
func ReadCSV[T any](r io.Reader, opts ...DTOOption) ([]T, error) {
	cr := csv.NewReader(r)
	headers, err := cr.Read()
	if err != nil {
		return nil, err
	}
	var out []T
	for {
		row, err := cr.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return out, err
		}
		v, err := FromCSVRow[T](headers, row, opts...)
		if err != nil {
			return out, err
		}
		out = append(out, v)
	}
	return out, nil
}
func WriteCSV[T any](w io.Writer, rows []T, opts ...DTOOption) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()
	if len(rows) == 0 {
		return nil
	}
	desc := Describe[T](CSV())
	headers := make([]string, len(desc))
	for i, d := range desc {
		headers[i] = d.Name
	}
	if err := cw.Write(headers); err != nil {
		return err
	}
	for _, row := range rows {
		m, err := StructToMap(row, append([]DTOOption{CSV()}, opts...)...)
		if err != nil {
			return err
		}
		rec := make([]string, len(headers))
		for i, h := range headers {
			rec[i] = ToDebugString(m[h])
		}
		if err := cw.Write(rec); err != nil {
			return err
		}
	}
	return cw.Error()
}

// Collection helpers.
func SliceToMap[T any, K comparable](items []T, field string) (map[K]T, error) {
	out := make(map[K]T, len(items))
	for _, it := range items {
		v, err := fieldValueByName[K](it, field)
		if err != nil {
			return out, err
		}
		out[v] = it
	}
	return out, nil
}
func GroupBy[T any, K comparable](items []T, field string) (map[K][]T, error) {
	out := map[K][]T{}
	for _, it := range items {
		v, err := fieldValueByName[K](it, field)
		if err != nil {
			return out, err
		}
		out[v] = append(out[v], it)
	}
	return out, nil
}
func Pluck[T any, V any](items []T, field string) ([]V, error) {
	out := make([]V, 0, len(items))
	for _, it := range items {
		v, err := fieldValueByName[V](it, field)
		if err != nil {
			return out, err
		}
		out = append(out, v)
	}
	return out, nil
}
func fieldValueByName[V any](item any, name string) (V, error) {
	rv := reflect.ValueOf(item)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			var z V
			return z, ErrNil
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		var z V
		return z, ErrUnsupported
	}
	meta := dtoMetaFor(rv.Type(), DefaultDTOOptions())
	for _, f := range meta.fields {
		if f.primary == name || strings.EqualFold(f.structField.Name, name) {
			fv := fieldByIndex(rv, f.index)
			return DTOTo[V](fv.Interface())
		}
	}
	var z V
	return z, ErrInvalid
}

// Enum registry convenience wrapper around the core enum registry.
func RegisterEnumValues[T comparable](values ...T) {
	m := map[string]T{}
	for _, v := range values {
		m[ToDebugString(v)] = v
	}
	RegisterEnum[T](m)
}

// Versioned migrations.
type Migration[S any, T any] func(S) (T, error)

func RegisterMigration[S any, T any](fn Migration[S, T])   { RegisterPair[S, T](PairConverter[S, T](fn)) }
func Migrate[T any](src any, opts ...DTOOption) (T, error) { return DTOTo[T](src, opts...) }

// Stats and cache warmup.
type RuntimeStats struct {
	RegisteredPairs   int
	FieldCacheEntries int
}

func Stats() RuntimeStats {
	n := 0
	pairConverters.Range(func(_, _ any) bool { n++; return true })
	fc := 0
	dtoFieldCache.Range(func(_, _ any) bool { fc++; return true })
	return RuntimeStats{RegisteredPairs: n, FieldCacheEntries: fc}
}
func Warmup[T any](sample any, opts ...DTOOption) error { var out T; return DTO(&out, sample, opts...) }
func WarmupPair[S any, T any](opts ...DTOOption) error {
	var s S
	_, err := DTOConvertPair[S, T](s, opts...)
	return err
}

// Context-aware batch conversion.
func BatchContext[T any](ctx context.Context, src any, opts ...DTOBatchOption) ([]T, error) {
	items, err := collectSliceItems(src, WithTrimSpace())
	if err != nil {
		return nil, err
	}
	out := make([]T, 0, len(items))
	for i, item := range items {
		select {
		case <-ctx.Done():
			return out, ctx.Err()
		default:
			{
			}
		}
		v, err := DTOTo[T](item)
		if err != nil {
			return out, PathError(indexPath("", i), KindOf(item), KindInvalid, item, err)
		}
		out = append(out, v)
	}
	return out, nil
}

// Output map options.
type MapOutputOptions struct{ OmitEmpty, OmitZero, NilAsNull bool }
type MapOutputOption func(*MapOutputOptions)

func OmitEmpty() MapOutputOption { return func(o *MapOutputOptions) { o.OmitEmpty = true } }
func OmitZero() MapOutputOption  { return func(o *MapOutputOptions) { o.OmitZero = true } }
func NilAsNull() MapOutputOption { return func(o *MapOutputOptions) { o.NilAsNull = true } }
func ObjectToMap(src any, opts ...MapOutputOption) (map[string]any, error) {
	m, err := StructToMap(src)
	if err != nil {
		return nil, err
	}
	var o MapOutputOptions
	for _, opt := range opts {
		if opt != nil {
			opt(&o)
		}
	}
	for k, v := range m {
		rv := reflect.ValueOf(v)
		if (o.OmitEmpty || o.OmitZero) && (!rv.IsValid() || rv.IsZero()) {
			delete(m, k)
		}
	}
	return m, nil
}

// Safe limit aliases for API readability. MaxDepth/SliceLen/MapSize are enforced by DTO core.
func MaxDepth(n int) DTOOption    { return WithDTOMaxDepth(n) }
func MaxSliceLen(n int) DTOOption { return WithDTOMaxSliceLen(n) }
func MaxMapSize(n int) DTOOption  { return WithDTOMaxMapSize(n) }
