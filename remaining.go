package convert

import (
	"bufio"
	"encoding/csv"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"
)

// Code is a stable machine-readable error code for conversion failures.
type Code uint8

const (
	CodeUnsupported Code = iota + 1
	CodeInvalid
	CodeOverflow
	CodeEmpty
	CodeNil
	CodePrecisionLoss
	CodeUnsafe
	CodeValidationFailed
)

// ErrorDetail is a rich, path-aware conversion error. Its Error string is built lazily.
type ErrorDetail struct {
	Code  Code
	Path  string
	From  Kind
	To    Kind
	Value any
	Cause error
}

func (e *ErrorDetail) Error() string {
	if e == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("convert")
	if e.Path != "" {
		b.WriteString(" ")
		b.WriteString(e.Path)
	}
	b.WriteString(": ")
	if e.Cause != nil {
		b.WriteString(e.Cause.Error())
	} else {
		b.WriteString(codeString(e.Code))
	}
	return b.String()
}
func (e *ErrorDetail) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}
func codeString(c Code) string {
	switch c {
	case CodeUnsupported:
		return "unsupported conversion"
	case CodeInvalid:
		return "invalid value"
	case CodeOverflow:
		return "overflow"
	case CodeEmpty:
		return "empty value"
	case CodeNil:
		return "nil value"
	case CodePrecisionLoss:
		return "precision loss"
	case CodeUnsafe:
		return "unsafe conversion"
	case CodeValidationFailed:
		return "validation failed"
	default:
		return "conversion failed"
	}
}
func PathError(path string, from, to Kind, value any, cause error) error {
	c := CodeInvalid
	switch {
	case errors.Is(cause, ErrUnsupported):
		c = CodeUnsupported
	case errors.Is(cause, ErrOverflow):
		c = CodeOverflow
	case errors.Is(cause, ErrEmpty):
		c = CodeEmpty
	case errors.Is(cause, ErrNil):
		c = CodeNil
	case errors.Is(cause, ErrPrecisionLoss):
		c = CodePrecisionLoss
	case errors.Is(cause, ErrValidation):
		c = CodeValidationFailed
	}
	return &ErrorDetail{Code: c, Path: path, From: from, To: to, Value: value, Cause: cause}
}

// Precision and Unix/time policy options.
func NoPrecisionLoss() OptionFunc {
	return func(p *Policy) { p.FloatToIntOnlyExact = true; p.OverflowCheck = true }
}
func AllowFloatToIntTruncate() OptionFunc {
	return func(p *Policy) { p.FloatToInt = true; p.FloatToIntOnlyExact = false }
}
func AllowFloatToIntExactOnly() OptionFunc {
	return func(p *Policy) { p.FloatToInt = true; p.FloatToIntOnlyExact = true }
}
func RejectFloatToInt() OptionFunc { return func(p *Policy) { p.FloatToInt = false } }
func RejectLargeIntToFloat() OptionFunc {
	return func(p *Policy) { p.OverflowCheck = true; p.FloatToIntOnlyExact = true }
}

// ToIntWith applies precision policy around the existing optimized ToInt implementation.
func ToIntWith(v any, opts ...OptionFunc) (int, error)     { return toInt64With[int](v, opts...) }
func ToInt64With(v any, opts ...OptionFunc) (int64, error) { return toInt64With[int64](v, opts...) }
func toInt64With[T ~int | ~int64](v any, opts ...OptionFunc) (T, error) {
	p := policyFrom(opts)
	switch x := v.(type) {
	case float32:
		f := float64(x)
		if !p.FloatToInt {
			return 0, ErrPrecisionLoss
		}
		if p.FloatToIntOnlyExact && mathTrunc(f) != f {
			return 0, ErrPrecisionLoss
		}
	case float64:
		if !p.FloatToInt {
			return 0, ErrPrecisionLoss
		}
		if p.FloatToIntOnlyExact && mathTrunc(x) != x {
			return 0, ErrPrecisionLoss
		}
	}
	var i int64
	switch x := v.(type) {
	case float32:
		i = int64(x)
	case float64:
		i = int64(x)
	default:
		var err error
		i, err = ToInt64(v)
		if err != nil {
			return 0, err
		}
	}
	if reflect.TypeOf(*new(T)).Kind() == reflect.Int && (i < int64(^uint(0)>>1)*-1-1 || i > int64(^uint(0)>>1)) {
		return 0, ErrOverflow
	}
	return T(i), nil
}
func mathTrunc(f float64) float64 { return float64(int64(f)) }

// ToAnySlice converts strings, []byte, []any, arrays and slices into []any.
func ToAnySlice(v any, opts ...SplitOption) ([]any, error) {
	sp := splitPolicyFrom(opts)
	switch x := v.(type) {
	case nil:
		return nil, ErrNil
	case []any:
		if !sp.trim && !sp.ignoreEmpty && !sp.unique {
			return x, nil
		}
		out := make([]any, 0, len(x))
		seen := make(map[string]struct{}, len(x))
		for _, it := range x {
			if sp.unique {
				k := anyKey(it)
				if _, ok := seen[k]; ok {
					continue
				}
				seen[k] = struct{}{}
			}
			out = append(out, it)
		}
		return out, nil
	case string:
		ss := strings.Split(x, sp.sep)
		out := make([]any, 0, len(ss))
		seen := map[string]struct{}{}
		for _, s := range ss {
			if sp.trim {
				s = strings.TrimSpace(s)
			}
			if sp.ignoreEmpty && s == "" {
				continue
			}
			if sp.unique {
				if _, ok := seen[s]; ok {
					continue
				}
				seen[s] = struct{}{}
			}
			out = append(out, s)
		}
		return out, nil
	case []byte:
		return ToAnySlice(bytesToString(x), opts...)
	}
	rv := reflect.ValueOf(v)
	if !rv.IsValid() || (rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array) {
		return nil, ErrUnsupported
	}
	out := make([]any, 0, rv.Len())
	seen := map[string]struct{}{}
	for i := 0; i < rv.Len(); i++ {
		val := rv.Index(i).Interface()
		if sp.unique {
			k := anyKey(val)
			if _, ok := seen[k]; ok {
				continue
			}
			seen[k] = struct{}{}
		}
		out = append(out, val)
	}
	return out, nil
}
func anyKey(v any) string { return reflect.TypeOf(v).String() + ":" + ToDebugString(v) }
func ToDebugString(v any) string {
	if s, err := ToString(v); err == nil {
		return s
	}
	return reflect.ValueOf(v).String()
}

// ToTypedSlice converts to []T for any T. Built-in scalar values use the zero-allocation fast paths;
// custom types use registered converters, FromAny, structs and maps through Convert.
func ToTypedSlice[T any](v any, opts ...SplitOption) ([]T, error) {
	parts, err := ToAnySlice(v, opts...)
	if err != nil {
		return nil, err
	}
	out := make([]T, 0, len(parts))
	sp := splitPolicyFrom(opts)
	seen := map[string]struct{}{}
	for i, part := range parts {
		var x T
		if err := setReflectValuePath(reflect.ValueOf(&x).Elem(), part, indexPath("", i)); err != nil {
			return nil, err
		}
		if sp.unique {
			k := anyKey(x)
			if _, ok := seen[k]; ok {
				continue
			}
			seen[k] = struct{}{}
		}
		out = append(out, x)
	}
	return out, nil
}

// Env helpers.
type EnvOption func(*envPolicy)
type envPolicy struct {
	prefix   string
	required bool
	sep      string
	trim     bool
}

func WithPrefix(prefix string) EnvOption { return func(p *envPolicy) { p.prefix = prefix } }
func WithRequired() EnvOption            { return func(p *envPolicy) { p.required = true } }
func WithEnvSeparator(sep string) EnvOption {
	return func(p *envPolicy) {
		if sep != "" {
			p.sep = sep
		}
	}
}
func WithEnvTrimSpace() EnvOption { return func(p *envPolicy) { p.trim = true } }
func envPolicyFrom(opts []EnvOption) envPolicy {
	p := envPolicy{sep: ","}
	for _, opt := range opts {
		if opt != nil {
			opt(&p)
		}
	}
	return p
}
func Env[T Target](key string, fallback T, opts ...EnvOption) T {
	p := envPolicyFrom(opts)
	s, ok := os.LookupEnv(p.prefix + key)
	if !ok || s == "" {
		return fallback
	}
	if x, err := To[T](s); err == nil {
		return x
	}
	return fallback
}
func EnvRequired[T Target](key string, opts ...EnvOption) (T, error) {
	p := envPolicyFrom(opts)
	s, ok := os.LookupEnv(p.prefix + key)
	if !ok || s == "" {
		var z T
		return z, ErrEmpty
	}
	return To[T](s)
}
func EnvDuration(key string, fallback time.Duration, opts ...EnvOption) time.Duration {
	return Env[time.Duration](key, fallback, opts...)
}
func EnvSlice[T Target](key string, opts ...EnvOption) ([]T, error) {
	p := envPolicyFrom(opts)
	s, ok := os.LookupEnv(p.prefix + key)
	if !ok || s == "" {
		if p.required {
			return nil, ErrEmpty
		}
		return nil, nil
	}
	splitOpts := []SplitOption{WithSeparator(p.sep)}
	if p.trim {
		splitOpts = append(splitOpts, WithTrimSpace())
	}
	return ToSlice[T](s, splitOpts...)
}
func EnvStruct[T any](opts ...EnvOption) (T, error) {
	p := envPolicyFrom(opts)
	m := map[string]any{}
	for _, e := range os.Environ() {
		k, v, ok := strings.Cut(e, "=")
		if !ok {
			continue
		}
		if p.prefix != "" {
			if !strings.HasPrefix(k, p.prefix) {
				continue
			}
			k = strings.TrimPrefix(k, p.prefix)
		}
		m[strings.ToLower(k)] = v
	}
	return ToStruct[T](m)
}

// Binding helpers.
func BindQuery(dst any, values url.Values) error { return bindValues(dst, values, "query") }
func BindForm(dst any, values url.Values) error  { return bindValues(dst, values, "form") }
func BindHeader(dst any, h http.Header) error {
	vals := url.Values{}
	for k, v := range h {
		vals[k] = v
	}
	return bindValues(dst, vals, "header")
}
func bindValues(dst any, values url.Values, tag string) error {
	rv := reflect.ValueOf(dst)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return ErrUnsupported
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return ErrUnsupported
	}
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if f.PkgPath != "" {
			continue
		}
		name := f.Tag.Get(tag)
		if name == "-" {
			continue
		}
		if name == "" {
			name = f.Tag.Get("convert")
		}
		if name == "" {
			name = snakeName(f.Name)
		}
		vals, ok := lookupValues(values, name)
		if !ok || len(vals) == 0 || vals[0] == "" {
			if def := f.Tag.Get("default"); def != "" {
				if err := setReflectValue(rv.Field(i), def); err != nil {
					return PathError(name, KindString, KindOf(rv.Field(i).Interface()), def, err)
				}
			}
			continue
		}
		var val any = vals[0]
		if rv.Field(i).Kind() == reflect.Slice {
			val = vals
			if len(vals) == 1 {
				val = vals[0]
			}
		}
		if err := setReflectValue(rv.Field(i), val); err != nil {
			return PathError(name, KindString, KindOf(rv.Field(i).Interface()), val, err)
		}
	}
	return nil
}
func lookupValues(v url.Values, key string) ([]string, bool) {
	if x, ok := v[key]; ok {
		return x, true
	}
	alt := ""
	if strings.HasSuffix(key, "s") && len(key) > 1 {
		alt = strings.TrimSuffix(key, "s")
	}
	for k, x := range v {
		if strings.EqualFold(k, key) || (alt != "" && strings.EqualFold(k, alt)) {
			return x, true
		}
	}
	return nil, false
}
func BindCSVRow(dst any, headers, row []string) error {
	if len(headers) != len(row) {
		return ErrInvalid
	}
	vals := url.Values{}
	for i, h := range headers {
		vals.Add(h, row[i])
	}
	if err := bindValues(dst, vals, "csv"); err != nil {
		return err
	}
	return nil
}
func BindCSV(dst any, r io.Reader) error {
	cr := csv.NewReader(bufio.NewReader(r))
	headers, err := cr.Read()
	if err != nil {
		return err
	}
	row, err := cr.Read()
	if err != nil {
		return err
	}
	return BindCSVRow(dst, headers, row)
}

// Conversion graph. Direct built-ins remain faster; the graph is for extensibility/chaining.
type edgeKey struct{ from, to reflect.Type }
type graphEdge struct{ fn reflect.Value }

var graph sync.Map

func RegisterEdge[S any, T any](fn func(S) (T, error)) {
	var s S
	var t T
	graph.Store(edgeKey{reflect.TypeOf(s), reflect.TypeOf(t)}, graphEdge{reflect.ValueOf(fn)})
}
func ConvertGraph[T any](v any) (T, error) {
	var z T
	to := reflect.TypeOf(z)
	if to == nil {
		return z, ErrUnsupported
	}
	if v == nil {
		return z, ErrNil
	}
	from := reflect.TypeOf(v)
	if from.AssignableTo(to) {
		return v.(T), nil
	}
	if e, ok := graph.Load(edgeKey{from, to}); ok {
		return callEdge[T](e.(graphEdge), v)
	}
	// Try one-hop via registered intermediate edges.
	var result T
	var found bool
	var foundErr error
	graph.Range(func(k, val any) bool {
		ek := k.(edgeKey)
		if ek.from != from {
			return true
		}
		midRes := val.(graphEdge).fn.Call([]reflect.Value{reflect.ValueOf(v)})
		if !midRes[1].IsNil() {
			return true
		}
		mid := midRes[0].Interface()
		if e2, ok := graph.Load(edgeKey{reflect.TypeOf(mid), to}); ok {
			r, err := callEdge[T](e2.(graphEdge), mid)
			if err != nil {
				foundErr = err
			} else {
				result = r
				found = true
			}
			return false
		}
		return true
	})
	if found {
		return result, nil
	}
	if foundErr != nil {
		return z, foundErr
	}
	return Convert[T](v)
}
func callEdge[T any](e graphEdge, v any) (T, error) {
	var z T
	res := e.fn.Call([]reflect.Value{reflect.ValueOf(v)})
	if !res[1].IsNil() {
		return z, res[1].Interface().(error)
	}
	return res[0].Interface().(T), nil
}

// ConversionMatrix returns a markdown compatibility matrix for documentation/tests.
func ConversionMatrix() string {
	rows := [][]string{{"from \\ to", "bool", "int", "uint", "float", "string", "duration", "time", "bytes", "slice"}, {"string", "yes", "yes", "yes", "yes", "yes", "yes", "yes", "yes", "yes"}, {"[]byte", "yes", "yes", "yes", "yes", "yes", "yes", "yes", "yes", "yes"}, {"int", "yes", "yes", "yes", "yes", "yes", "yes", "yes", "yes", "no"}, {"uint", "yes", "yes", "yes", "yes", "yes", "yes", "yes", "yes", "no"}, {"float", "yes", "yes*", "yes*", "yes", "yes", "yes", "yes", "yes", "no"}, {"bool", "yes", "yes", "yes", "yes", "yes", "no", "no", "yes", "no"}, {"time.Time", "no", "yes", "yes", "no", "yes", "no", "yes", "no", "no"}, {"time.Duration", "no", "yes", "yes", "no", "yes", "yes", "no", "yes", "no"}, {"[]any/slice", "no", "no", "no", "no", "no", "no", "no", "no", "yes"}}
	widths := make([]int, len(rows[0]))
	for _, r := range rows {
		for i, c := range r {
			if len(c) > widths[i] {
				widths[i] = len(c)
			}
		}
	}
	var b strings.Builder
	for i, r := range rows {
		b.WriteByte('|')
		for j, c := range r {
			b.WriteByte(' ')
			b.WriteString(c)
			for k := len(c); k < widths[j]; k++ {
				b.WriteByte(' ')
			}
			b.WriteByte(' ')
			b.WriteByte('|')
		}
		b.WriteByte('\n')
		if i == 0 {
			b.WriteByte('|')
			for _, w := range widths {
				b.WriteByte(' ')
				b.WriteString(strings.Repeat("-", w))
				b.WriteByte(' ')
				b.WriteByte('|')
			}
			b.WriteByte('\n')
		}
	}
	return b.String()
}
func sortedKeys(m map[string]any) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

// parseStructTypes is shared by convertgen tests and the command implementation.
func parseStructTypes(filename string) (map[string]*ast.StructType, string, error) {
	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, "", err
	}
	out := map[string]*ast.StructType{}
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			st, ok := ts.Type.(*ast.StructType)
			if ok {
				out[ts.Name.Name] = st
			}
		}
	}
	return out, f.Name.Name, nil
}
