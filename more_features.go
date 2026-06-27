package convert

import (
	"encoding/csv"
	"errors"
	"go/format"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"
)

// Code is a stable machine-readable error code for callers that do not want
// to parse Error() strings.
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

// PathError annotates conversion failures with a nested path, e.g.
// "server.routes[2].timeout". Error strings are built lazily only when Error
// is called, keeping success paths allocation-free.
type PathError struct {
	Code  Code
	Path  string
	From  Kind
	To    Kind
	Value any
	Cause error
}

func (e *PathError) Error() string {
	if e == nil {
		return ""
	}
	var b strings.Builder
	if e.Path != "" {
		b.WriteString(e.Path)
		b.WriteString(": ")
	}
	b.WriteString("convert: ")
	if e.Cause != nil {
		b.WriteString(e.Cause.Error())
	} else {
		b.WriteString("conversion failed")
	}
	return b.String()
}
func (e *PathError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func codeForError(err error) Code {
	switch {
	case errors.Is(err, ErrUnsupported):
		return CodeUnsupported
	case errors.Is(err, ErrInvalid):
		return CodeInvalid
	case errors.Is(err, ErrOverflow):
		return CodeOverflow
	case errors.Is(err, ErrEmpty):
		return CodeEmpty
	case errors.Is(err, ErrNil):
		return CodeNil
	case errors.Is(err, ErrPrecisionLoss):
		return CodePrecisionLoss
	case errors.Is(err, ErrValidation):
		return CodeValidationFailed
	default:
		return CodeInvalid
	}
}
func wrapPath(path string, from, to Kind, value any, err error) error {
	if err == nil {
		return nil
	}
	return &PathError{Code: codeForError(err), Path: path, From: from, To: to, Value: value, Cause: err}
}

// Precision policy helpers.
func NoPrecisionLoss() OptionFunc {
	return func(p *Policy) { p.FloatToIntOnlyExact = true; p.OverflowCheck = true }
}
func AllowFloatToIntTruncate() OptionFunc {
	return func(p *Policy) { p.FloatToInt = true; p.FloatToIntOnlyExact = false }
}
func AllowFloatToIntExactOnly() OptionFunc {
	return func(p *Policy) { p.FloatToInt = true; p.FloatToIntOnlyExact = true }
}
func RejectLargeIntToFloat() OptionFunc { return func(p *Policy) { p.OverflowCheck = true } }

// ConvertWith applies policy-aware checks on top of the registry/scalar path.
func ConvertWith[T any](v any, opts ...OptionFunc) (T, error) {
	p := policyFrom(opts)
	var z T
	// Enforce exact float->int when requested for reflect-driven conversions.
	if p.FloatToInt && !p.FloatToIntOnlyExact {
		return Convert[T](v, opts...)
	}
	if p.FloatToIntOnlyExact {
		rv := reflect.ValueOf(z)
		sv := reflect.ValueOf(v)
		if rv.IsValid() && sv.IsValid() && isIntKind(rv.Kind()) && isFloatKind(sv.Kind()) {
			f, err := ToFloat64(v)
			if err != nil {
				return z, err
			}
			if float64(int64(f)) != f {
				return z, ErrPrecisionLoss
			}
		}
	}
	return Convert[T](v, opts...)
}
func isIntKind(k reflect.Kind) bool   { return k >= reflect.Int && k <= reflect.Int64 }
func isFloatKind(k reflect.Kind) bool { return k == reflect.Float32 || k == reflect.Float64 }

// Conversion graph. Built-in scalar conversions still use direct functions;
// graph edges are for custom chained domain conversions.
type edgeKey struct{ from, to reflect.Type }
type graphEdge struct{ fn reflect.Value }

var graphRegistry sync.Map // map[edgeKey]graphEdge

func RegisterEdge[From any, To any](fn func(From) (To, error)) {
	var f From
	var t To
	graphRegistry.Store(edgeKey{reflect.TypeOf(f), reflect.TypeOf(t)}, graphEdge{fn: reflect.ValueOf(fn)})
}
func UnregisterEdge[From any, To any]() {
	var f From
	var t To
	graphRegistry.Delete(edgeKey{reflect.TypeOf(f), reflect.TypeOf(t)})
}

func ConvertGraph[T any](v any) (T, error) {
	var z T
	to := reflect.TypeOf(z)
	from := reflect.TypeOf(v)
	if to == nil {
		return any(v).(T), nil
	}
	if from == to {
		return v.(T), nil
	}
	if e, ok := graphRegistry.Load(edgeKey{from, to}); ok {
		res := e.(graphEdge).fn.Call([]reflect.Value{reflect.ValueOf(v)})
		if !res[1].IsNil() {
			return z, res[1].Interface().(error)
		}
		return res[0].Interface().(T), nil
	}
	// Try one-hop chain: from -> mid -> to.
	var foundErr error
	graphRegistry.Range(func(k, val any) bool {
		key := k.(edgeKey)
		if key.from != from {
			return true
		}
		midRes := val.(graphEdge).fn.Call([]reflect.Value{reflect.ValueOf(v)})
		if !midRes[1].IsNil() {
			foundErr = midRes[1].Interface().(error)
			return false
		}
		mid := midRes[0]
		if e2, ok := graphRegistry.Load(edgeKey{key.to, to}); ok {
			res := e2.(graphEdge).fn.Call([]reflect.Value{mid})
			if !res[1].IsNil() {
				foundErr = res[1].Interface().(error)
				return false
			}
			z = res[0].Interface().(T)
			foundErr = nil
			return false
		}
		return true
	})
	if foundErr != nil {
		return z, foundErr
	}
	if !reflect.ValueOf(z).IsZero() {
		return z, nil
	}
	return Convert[T](v)
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

func Env[T any](key string, fallback T, opts ...EnvOption) T {
	p := envPolicyFrom(opts)
	s, ok := os.LookupEnv(p.prefix + key)
	if !ok || s == "" {
		return fallback
	}
	x, err := Convert[T](s)
	if err != nil {
		return fallback
	}
	return x
}
func EnvRequired[T any](key string, opts ...EnvOption) (T, error) {
	p := envPolicyFrom(opts)
	s, ok := os.LookupEnv(p.prefix + key)
	var z T
	if !ok || s == "" {
		return z, ErrEmpty
	}
	return Convert[T](s)
}
func EnvDuration(key string, fallback time.Duration, opts ...EnvOption) time.Duration {
	return Env(key, fallback, opts...)
}
func EnvSlice[T any](key string, fallback []T, opts ...EnvOption) []T {
	p := envPolicyFrom(opts)
	s, ok := os.LookupEnv(p.prefix + key)
	if !ok || s == "" {
		return fallback
	}
	sopts := []SplitOption{WithSeparator(p.sep)}
	if p.trim {
		sopts = append(sopts, WithTrimSpace())
	}
	x, err := ToSlice[T](s, sopts...)
	if err != nil {
		return fallback
	}
	return x
}
func EnvStruct[T any](opts ...EnvOption) (T, error) {
	p := envPolicyFrom(opts)
	var z T
	m := map[string]any{}
	t := reflect.TypeOf(z)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return z, ErrUnsupported
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" {
			continue
		}
		name := f.Tag.Get("env")
		if name == "-" {
			continue
		}
		if name == "" {
			name = strings.ToUpper(snakeName(f.Name))
		}
		if s, ok := os.LookupEnv(p.prefix + name); ok {
			m[snakeName(f.Name)] = s
		}
	}
	return ToStruct[T](m)
}

// Binding helpers for web and tabular input.
func BindQuery(dst any, values url.Values) error { return bindValues(dst, values, "query") }
func BindForm(dst any, values url.Values) error  { return bindValues(dst, values, "form") }
func BindHeader(dst any, h http.Header) error    { return bindHeader(dst, h) }
func BindCSVRow(dst any, headers, row []string) error {
	if len(headers) != len(row) {
		return ErrInvalid
	}
	m := make(map[string]any, len(headers))
	for i, h := range headers {
		m[h] = row[i]
	}
	return PopulateTagged(dst, m, "csv")
}
func BindCSV(dst any, csvData string) error {
	r := csv.NewReader(strings.NewReader(csvData))
	rows, err := r.ReadAll()
	if err != nil {
		return ErrInvalid
	}
	if len(rows) < 2 {
		return ErrInvalid
	}
	return BindCSVRow(dst, rows[0], rows[1])
}
func bindValues(dst any, values url.Values, tag string) error {
	m := map[string]any{}
	for k, v := range values {
		if len(v) == 1 {
			m[k] = v[0]
		} else {
			a := make([]any, len(v))
			for i := range v {
				a[i] = v[i]
			}
			m[k] = a
		}
	}
	return PopulateTagged(dst, m, tag)
}
func bindHeader(dst any, h http.Header) error {
	m := map[string]any{}
	for k, v := range h {
		if len(v) == 1 {
			m[k] = v[0]
		} else {
			a := make([]any, len(v))
			for i := range v {
				a[i] = v[i]
			}
			m[k] = a
		}
	}
	return PopulateTagged(dst, m, "header")
}
func PopulateTagged(dst any, src any, tag string) error {
	if dst == nil {
		return ErrNil
	}
	dv := reflect.ValueOf(dst)
	if dv.Kind() != reflect.Pointer || dv.IsNil() {
		return ErrUnsupported
	}
	dv = dv.Elem()
	if dv.Kind() != reflect.Struct {
		return ErrUnsupported
	}
	m, err := sourceMap(src)
	if err != nil {
		return err
	}
	dt := dv.Type()
	for i := 0; i < dt.NumField(); i++ {
		f := dt.Field(i)
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
		val, ok := lookupCaseInsensitive(m, name)
		if !ok {
			if def := f.Tag.Get("default"); def != "" {
				val = def
				ok = true
			}
		}
		if !ok {
			continue
		}
		if err := setReflectValue(dv.Field(i), val); err != nil {
			return wrapPath(name, KindOf(val), kindOfReflect(f.Type), val, err)
		}
	}
	return nil
}
func kindOfReflect(t reflect.Type) Kind {
	if t == nil {
		return KindInvalid
	}
	switch t.Kind() {
	case reflect.Bool:
		return KindBool
	case reflect.String:
		return KindString
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if t == reflect.TypeOf(time.Duration(0)) {
			return KindDuration
		}
		return KindInt
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return KindUint
	case reflect.Float32, reflect.Float64:
		return KindFloat
	case reflect.Slice, reflect.Array:
		return KindSlice
	case reflect.Map:
		return KindMap
	case reflect.Struct:
		if t == reflect.TypeOf(time.Time{}) {
			return KindTime
		}
		return KindStruct
	}
	return KindInvalid
}

// Unsafe audit controls. Existing ToString([]byte) is optimized zero-copy; callers
// that require a copy can use ToStringCopy or ToWith[string](b, SafeBytesToString()).
var unsafeByteStringEnabled = true

func SafeMode()             { unsafeByteStringEnabled = false }
func UnsafeMode()           { unsafeByteStringEnabled = true }
func IsUnsafeEnabled() bool { return unsafeByteStringEnabled }
func ToStringCopy(v any) (string, error) {
	if b, ok := v.([]byte); ok {
		return string(b), nil
	}
	return ToString(v)
}

// ConversionMatrix describes built-in high-level support. It is also used to
// generate CONVERSION_MATRIX.md.
func ConversionMatrix() [][]string {
	return [][]string{
		{"from/to", "bool", "int", "uint", "float", "string", "duration", "time", "bytes", "slice", "map", "struct"},
		{"string", "yes", "yes", "yes", "yes", "yes", "yes", "yes", "yes", "yes", "no", "yes"},
		{"[]byte", "yes", "yes", "yes", "yes", "yes", "yes", "yes", "yes", "yes", "no", "no"},
		{"[]any", "no", "no", "no", "no", "no", "no", "no", "no", "yes", "no", "no"},
		{"int", "yes", "yes", "yes", "yes", "yes", "yes", "yes", "yes", "no", "no", "no"},
		{"uint", "yes", "yes", "yes", "yes", "yes", "yes", "yes", "yes", "no", "no", "no"},
		{"float", "yes", "exact", "exact", "yes", "yes", "yes", "yes", "yes", "no", "no", "no"},
		{"bool", "yes", "yes", "yes", "yes", "yes", "yes", "no", "yes", "no", "no", "no"},
		{"time.Time", "yes", "unix", "unix", "unix", "yes", "no", "yes", "yes", "no", "no", "no"},
		{"duration", "yes", "ns", "ns", "ns", "yes", "yes", "no", "yes", "no", "no", "no"},
		{"map", "no", "no", "no", "no", "yes", "no", "no", "no", "no", "yes", "yes"},
		{"struct", "no", "no", "no", "no", "no", "no", "no", "no", "no", "no", "yes"},
	}
}
func MatrixMarkdown() string {
	m := ConversionMatrix()
	var b strings.Builder
	b.WriteString("# Conversion Matrix\n\n")
	for i, row := range m {
		b.WriteByte('|')
		for _, c := range row {
			b.WriteByte(' ')
			b.WriteString(c)
			b.WriteString(" |")
		}
		b.WriteByte('\n')
		if i == 0 {
			b.WriteByte('|')
			for range row {
				b.WriteString(" --- |")
			}
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// Tiny generation helper used by cmd/convertgen tests/examples. It intentionally
// emits straightforward code that delegates field conversions to this package.
func GenerateStructConverter(pkg, typeName string, fields []string) ([]byte, error) {
	sort.Strings(fields)
	var b strings.Builder
	b.WriteString("package ")
	b.WriteString(pkg)
	b.WriteString("\n\n")
	b.WriteString("import convert \"github.com/oarkflow/convert\"\n\n")
	b.WriteString("func Populate")
	b.WriteString(typeName)
	b.WriteString("(dst *")
	b.WriteString(typeName)
	b.WriteString(", src map[string]any) error {\n")
	for _, f := range fields {
		b.WriteString("\tif v, ok := src[\"")
		b.WriteString(snakeName(f))
		b.WriteString("\"]; ok { if err := convert.AsInto(v, &dst.")
		b.WriteString(f)
		b.WriteString("); err != nil { return err } }\n")
	}
	b.WriteString("\treturn nil\n}\n")
	return format.Source([]byte(b.String()))
}
