package convert

import (
	"database/sql"
	"database/sql/driver"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	ErrEmpty         = errors.New("convert: empty value")
	ErrPrecisionLoss = errors.New("convert: precision loss")
	ErrValidation    = errors.New("convert: validation failed")
)

// Policy controls loose/strict conversion behavior for ToWith and higher-level helpers.
type Policy struct {
	Strict              bool
	TrimSpace           bool
	EmptyStringAsZero   bool
	EmptyStringAsNil    bool
	BoolNumeric         bool
	BoolText            bool
	FloatToInt          bool
	FloatToIntOnlyExact bool
	OverflowCheck       bool
	TimeZone            *time.Location
	TimeLayouts         []string
	DefaultDurationUnit DurationUnit
	UnsafeBytesToString bool
}

type OptionFunc func(*Policy)

func DefaultPolicy() Policy {
	return Policy{BoolNumeric: true, BoolText: true, FloatToInt: true, FloatToIntOnlyExact: true, OverflowCheck: true, TimeZone: time.Local, DefaultDurationUnit: Nanosecond, UnsafeBytesToString: true}
}
func Strict() OptionFunc {
	return func(p *Policy) { p.Strict = true; p.BoolNumeric = false; p.EmptyStringAsZero = false }
}
func Loose() OptionFunc {
	return func(p *Policy) { p.Strict = false; p.BoolNumeric = true; p.BoolText = true; p.FloatToInt = true }
}
func TrimSpace() OptionFunc     { return func(p *Policy) { p.TrimSpace = true } }
func EmptyAsZero() OptionFunc   { return func(p *Policy) { p.EmptyStringAsZero = true } }
func EmptyAsNil() OptionFunc    { return func(p *Policy) { p.EmptyStringAsNil = true } }
func NoBoolNumeric() OptionFunc { return func(p *Policy) { p.BoolNumeric = false } }
func WithLocation(loc *time.Location) OptionFunc {
	return func(p *Policy) {
		if loc != nil {
			p.TimeZone = loc
		}
	}
}
func WithTimeLayouts(layouts ...string) OptionFunc {
	return func(p *Policy) { p.TimeLayouts = append(p.TimeLayouts, layouts...) }
}
func WithDefaultDurationUnit(unit DurationUnit) OptionFunc {
	return func(p *Policy) {
		if unit > 0 {
			p.DefaultDurationUnit = unit
		}
	}
}
func SafeBytesToString() OptionFunc   { return func(p *Policy) { p.UnsafeBytesToString = false } }
func UnsafeBytesToString() OptionFunc { return func(p *Policy) { p.UnsafeBytesToString = true } }

func policyFrom(opts []OptionFunc) Policy {
	p := DefaultPolicy()
	for _, opt := range opts {
		if opt != nil {
			opt(&p)
		}
	}
	return p
}

func ToWith[T Target](v any, opts ...OptionFunc) (T, error) {
	p := policyFrom(opts)
	if s, ok := v.(string); ok {
		v = applyStringPolicy(s, p)
	}
	if b, ok := v.([]byte); ok {
		s := string(b)
		if p.UnsafeBytesToString {
			s = bytesToString(b)
		}
		v = applyStringPolicy(s, p)
	}
	var zero T
	if s, ok := v.(string); ok && s == "" {
		if p.EmptyStringAsZero {
			return zero, nil
		}
		if p.EmptyStringAsNil {
			return zero, ErrNil
		}
	}
	if _, ok := any(zero).(time.Time); ok {
		if len(p.TimeLayouts) > 0 {
			if t, err := ToTimeLayouts(v, p.TimeLayouts...); err == nil {
				return recast[T](t), nil
			}
		}
		if p.TimeZone != nil {
			t, err := ToTimeInLocation(v, p.TimeZone)
			return recast[T](t), err
		}
	}
	if _, ok := any(zero).(time.Duration); ok && p.DefaultDurationUnit != Nanosecond {
		d, err := ToDurationUnit(v, p.DefaultDurationUnit)
		return recast[T](d), err
	}
	return To[T](v)
}

func applyStringPolicy(s string, p Policy) string {
	if p.TrimSpace {
		return strings.TrimSpace(s)
	}
	return s
}

// Nullable and Option are lightweight optional value containers.
type Nullable[T any] struct {
	Value T
	Valid bool
}
type Option[T any] struct {
	Value T
	OK    bool
}

func ToPtr[T Target](v any) (*T, error) {
	x, err := To[T](v)
	if err != nil {
		return nil, err
	}
	return &x, nil
}
func ToNullable[T Target](v any) Nullable[T] {
	x, err := To[T](v)
	return Nullable[T]{Value: x, Valid: err == nil}
}
func ToOption[T Target](v any) Option[T] {
	x, err := To[T](v)
	return Option[T]{Value: x, OK: err == nil}
}
func IsZeroValue[T comparable](v T) bool { var z T; return v == z }
func IsNilLike(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return rv.IsNil()
	}
	return false
}

func ToSQLNullString(v any) sql.NullString {
	x, err := ToString(v)
	return sql.NullString{String: x, Valid: err == nil}
}
func ToSQLNullInt64(v any) sql.NullInt64 {
	x, err := ToInt64(v)
	return sql.NullInt64{Int64: x, Valid: err == nil}
}
func ToSQLNullFloat64(v any) sql.NullFloat64 {
	x, err := ToFloat64(v)
	return sql.NullFloat64{Float64: x, Valid: err == nil}
}
func ToSQLNullBool(v any) sql.NullBool {
	x, err := ToBool(v)
	return sql.NullBool{Bool: x, Valid: err == nil}
}
func ToSQLNullTime(v any) sql.NullTime {
	x, err := ToTime(v)
	return sql.NullTime{Time: x, Valid: err == nil}
}

func ScanTo[T Target](src any) (T, error) {
	if src == nil {
		var z T
		return z, ErrNil
	}
	return To[T](src)
}
func ToDriverValue(v any) (driver.Value, error) {
	switch x := v.(type) {
	case nil, int64, float64, bool, []byte, string, time.Time:
		return x, nil
	case driver.Valuer:
		return x.Value()
	case int:
		return int64(x), nil
	case int8:
		return int64(x), nil
	case int16:
		return int64(x), nil
	case int32:
		return int64(x), nil
	case uint, uint8, uint16, uint32, uint64, uintptr:
		u, err := ToUint64(x)
		if err != nil {
			return nil, err
		}
		if u > uint64(^uint64(0)>>1) {
			return nil, ErrOverflow
		}
		return int64(u), nil
	case float32:
		return float64(x), nil
	default:
		return nil, ErrUnsupported
	}
}

type SplitOption func(*splitPolicy)
type splitPolicy struct {
	sep         string
	trim        bool
	ignoreEmpty bool
	unique      bool
}

func WithSeparator(sep string) SplitOption {
	return func(p *splitPolicy) {
		if sep != "" {
			p.sep = sep
		}
	}
}
func WithTrimSpace() SplitOption   { return func(p *splitPolicy) { p.trim = true } }
func WithIgnoreEmpty() SplitOption { return func(p *splitPolicy) { p.ignoreEmpty = true } }
func WithUnique() SplitOption      { return func(p *splitPolicy) { p.unique = true } }
func splitPolicyFrom(opts []SplitOption) splitPolicy {
	p := splitPolicy{sep: ","}
	for _, opt := range opts {
		if opt != nil {
			opt(&p)
		}
	}
	return p
}

func ToSlice[T Target](v any, opts ...SplitOption) ([]T, error) {
	sp := splitPolicyFrom(opts)
	var parts []any
	switch x := v.(type) {
	case string:
		ss := strings.Split(x, sp.sep)
		parts = make([]any, 0, len(ss))
		for _, s := range ss {
			if sp.trim {
				s = strings.TrimSpace(s)
			}
			if sp.ignoreEmpty && s == "" {
				continue
			}
			parts = append(parts, s)
		}
	case []byte:
		return ToSlice[T](bytesToString(x), opts...)
	case []any:
		parts = x
	default:
		rv := reflect.ValueOf(v)
		if rv.IsValid() && (rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array) {
			parts = make([]any, rv.Len())
			for i := 0; i < rv.Len(); i++ {
				parts[i] = rv.Index(i).Interface()
			}
		} else {
			return nil, ErrUnsupported
		}
	}
	out := make([]T, 0, len(parts))
	seen := map[T]struct{}{}
	var z T
	comparable := reflect.TypeOf(z).Comparable()
	for _, part := range parts {
		x, err := To[T](part)
		if err != nil {
			return nil, err
		}
		if sp.unique && comparable {
			if _, ok := seen[x]; ok {
				continue
			}
			seen[x] = struct{}{}
		}
		out = append(out, x)
	}
	return out, nil
}

func ToStringSlice(v any, opts ...SplitOption) ([]string, error) { return ToSlice[string](v, opts...) }

type MapKey interface {
	Target
	comparable
}

func ToMap[K MapKey, V Target](v any) (map[K]V, error) {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() || rv.Kind() != reflect.Map {
		return nil, ErrUnsupported
	}
	out := make(map[K]V, rv.Len())
	for _, key := range rv.MapKeys() {
		k, err := To[K](key.Interface())
		if err != nil {
			return nil, err
		}
		val, err := To[V](rv.MapIndex(key).Interface())
		if err != nil {
			return nil, err
		}
		out[k] = val
	}
	return out, nil
}
func ToStringMap(v any) (map[string]string, error) { return ToMap[string, string](v) }
func ToAnyMap(v any) (map[string]any, error) {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() || rv.Kind() != reflect.Map {
		return nil, ErrUnsupported
	}
	out := make(map[string]any, rv.Len())
	for _, k := range rv.MapKeys() {
		ks, err := ToString(k.Interface())
		if err != nil {
			return nil, err
		}
		out[ks] = rv.MapIndex(k).Interface()
	}
	return out, nil
}

func ToStruct[T any](src any) (T, error) { var out T; err := Populate(&out, src); return out, err }
func Populate(dst any, src any) error    { return PopulateWithOptions(dst, src) }
func sourceMap(src any) (map[string]any, error) {
	switch x := src.(type) {
	case map[string]any:
		return x, nil
	case map[string]string:
		m := make(map[string]any, len(x))
		for k, v := range x {
			m[k] = v
		}
		return m, nil
	}
	rv := reflect.ValueOf(src)
	if rv.IsValid() && rv.Kind() == reflect.Map {
		m := make(map[string]any, rv.Len())
		for _, k := range rv.MapKeys() {
			ks, err := ToString(k.Interface())
			if err != nil {
				return nil, err
			}
			m[ks] = rv.MapIndex(k).Interface()
		}
		return m, nil
	}
	return nil, ErrUnsupported
}
func lookupCaseInsensitive(m map[string]any, k string) (any, bool) {
	if v, ok := m[k]; ok {
		return v, true
	}
	lk := strings.ToLower(k)
	for mk, mv := range m {
		if strings.ToLower(mk) == lk {
			return mv, true
		}
	}
	return nil, false
}
func snakeName(s string) string {
	var b strings.Builder
	for i, c := range s {
		if c >= 'A' && c <= 'Z' {
			if i > 0 {
				b.WriteByte('_')
			}
			c += 'a' - 'A'
		}
		b.WriteRune(c)
	}
	return b.String()
}
func setReflectValue(fv reflect.Value, val any) error { return setReflectValuePath(fv, val, "") }
func setSliceReflect(fv reflect.Value, val any) error {
	rv := reflect.ValueOf(val)
	var items []any
	switch x := val.(type) {
	case string:
		ss := strings.Split(x, ",")
		items = make([]any, len(ss))
		for i, s := range ss {
			items[i] = strings.TrimSpace(s)
		}
	case []any:
		items = x
	default:
		if rv.IsValid() && (rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array) {
			items = make([]any, rv.Len())
			for i := 0; i < rv.Len(); i++ {
				items[i] = rv.Index(i).Interface()
			}
		} else {
			return ErrUnsupported
		}
	}
	out := reflect.MakeSlice(fv.Type(), 0, len(items))
	for _, it := range items {
		elem := reflect.New(fv.Type().Elem()).Elem()
		if err := setReflectValue(elem, it); err != nil {
			return err
		}
		out = reflect.Append(out, elem)
	}
	fv.Set(out)
	return nil
}

// Converter registry for domain-specific types.
type Context struct {
	Policy Policy
	Path   string
}
type converterEntry struct{ fn reflect.Value }

var convRegistry sync.Map // map[reflect.Type]converterEntry
func Register[T any](fn func(any, Context) (T, error)) {
	var z T
	convRegistry.Store(reflect.TypeOf(z), converterEntry{fn: reflect.ValueOf(fn)})
}
func Unregister[T any]()        { var z T; convRegistry.Delete(reflect.TypeOf(z)) }
func HasConverter[T any]() bool { var z T; _, ok := convRegistry.Load(reflect.TypeOf(z)); return ok }
func Convert[T any](v any, opts ...OptionFunc) (T, error) {
	var z T
	typ := reflect.TypeOf(z)
	if e, ok := convRegistry.Load(typ); ok {
		res := e.(converterEntry).fn.Call([]reflect.Value{reflect.ValueOf(v), reflect.ValueOf(Context{Policy: policyFrom(opts)})})
		if !res[1].IsNil() {
			return z, res[1].Interface().(error)
		}
		return res[0].Interface().(T), nil
	}
	if err := setReflectValue(reflect.ValueOf(&z).Elem(), v); err != nil {
		return z, err
	}
	return z, nil
}

type Validator[T any] func(T) error

func ToValidated[T Target](v any, validators ...Validator[T]) (T, error) {
	x, err := To[T](v)
	if err != nil {
		return x, err
	}
	for _, fn := range validators {
		if fn != nil {
			if err := fn(x); err != nil {
				return x, err
			}
		}
	}
	return x, nil
}
func Min[T ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr | ~float32 | ~float64](min T) Validator[T] {
	return func(v T) error {
		if v < min {
			return ErrValidation
		}
		return nil
	}
}
func Max[T ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr | ~float32 | ~float64](max T) Validator[T] {
	return func(v T) error {
		if v > max {
			return ErrValidation
		}
		return nil
	}
}
func Range[T ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr | ~float32 | ~float64](min, max T) Validator[T] {
	return func(v T) error {
		if v < min || v > max {
			return ErrValidation
		}
		return nil
	}
}
func OneOf[T comparable](vals ...T) Validator[T] {
	return func(v T) error {
		for _, x := range vals {
			if v == x {
				return nil
			}
		}
		return ErrValidation
	}
}
func NotEmpty() Validator[string] {
	return func(v string) error {
		if v == "" {
			return ErrValidation
		}
		return nil
	}
}
func Len(n int) Validator[string] {
	return func(v string) error {
		if len(v) != n {
			return ErrValidation
		}
		return nil
	}
}
func MinLen(n int) Validator[string] {
	return func(v string) error {
		if len(v) < n {
			return ErrValidation
		}
		return nil
	}
}
func MaxLen(n int) Validator[string] {
	return func(v string) error {
		if len(v) > n {
			return ErrValidation
		}
		return nil
	}
}
func Regex(expr string) Validator[string] {
	re := regexp.MustCompile(expr)
	return func(v string) error {
		if !re.MatchString(v) {
			return ErrValidation
		}
		return nil
	}
}
func Email() Validator[string] { return Regex(`^[^@\s]+@[^@\s]+\.[^@\s]+$`) }
func URLValidator() Validator[string] {
	return func(v string) error {
		u, err := url.ParseRequestURI(v)
		if err != nil || u.Scheme == "" {
			return ErrValidation
		}
		return nil
	}
}
func Hostname() Validator[string] {
	return func(v string) error {
		if v == "" || strings.ContainsAny(v, " /\t\n\r") {
			return ErrValidation
		}
		return nil
	}
}
func IP() Validator[string] {
	return func(v string) error {
		if net.ParseIP(v) == nil {
			return ErrValidation
		}
		return nil
	}
}
func CIDR() Validator[string] {
	return func(v string) error {
		_, _, err := net.ParseCIDR(v)
		if err != nil {
			return ErrValidation
		}
		return nil
	}
}
func UUID() Validator[string] {
	return func(v string) error {
		if !ValidUUID(v) {
			return ErrValidation
		}
		return nil
	}
}

var enumRegistry sync.Map

func RegisterEnum[T comparable](m map[string]T) { var z T; enumRegistry.Store(reflect.TypeOf(z), m) }
func EnumValue[T comparable](name string) (T, error) {
	var z T
	if e, ok := enumRegistry.Load(reflect.TypeOf(z)); ok {
		if v, ok := e.(map[string]T)[name]; ok {
			return v, nil
		}
	}
	return z, ErrInvalid
}
func EnumValid[T comparable](v T) bool {
	if e, ok := enumRegistry.Load(reflect.TypeOf(v)); ok {
		for _, x := range e.(map[string]T) {
			if x == v {
				return true
			}
		}
	}
	return false
}
func EnumName[T comparable](v T) (string, bool) {
	if e, ok := enumRegistry.Load(reflect.TypeOf(v)); ok {
		for n, x := range e.(map[string]T) {
			if x == v {
				return n, true
			}
		}
	}
	return "", false
}

func ToBytes(v any) ([]byte, error) {
	switch x := v.(type) {
	case []byte:
		return x, nil
	case string:
		return []byte(x), nil
	default:
		var b [64]byte
		out, err := AppendString(b[:0], v)
		if err != nil {
			return nil, err
		}
		cp := make([]byte, len(out))
		copy(cp, out)
		return cp, nil
	}
}
func ToHex(v any) (string, error) {
	b, err := ToBytes(v)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
func FromHex(v any) ([]byte, error) {
	s, err := ToString(v)
	if err != nil {
		return nil, err
	}
	return hex.DecodeString(s)
}
func ToBase64(v any) (string, error) {
	b, err := ToBytes(v)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}
func FromBase64(v any) ([]byte, error) {
	s, err := ToString(v)
	if err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(s)
}
func Uint64ToBytes(v uint64, order binary.ByteOrder) []byte {
	b := make([]byte, 8)
	order.PutUint64(b, v)
	return b
}
func BytesToUint64(b []byte, order binary.ByteOrder) (uint64, error) {
	if len(b) < 8 {
		return 0, ErrInvalid
	}
	return order.Uint64(b), nil
}
func BigEndian() binary.ByteOrder    { return binary.BigEndian }
func LittleEndian() binary.ByteOrder { return binary.LittleEndian }

func ToBytesSize(v any) (uint64, error) {
	switch x := v.(type) {
	case uint64:
		return x, nil
	case int:
		if x < 0 {
			return 0, ErrOverflow
		}
		return uint64(x), nil
	case string:
		return parseSizeString(x)
	case []byte:
		return parseSizeString(bytesToString(x))
	default:
		return ToUint64(v)
	}
}
func parseSizeString(s string) (uint64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, ErrInvalid
	}
	i := 0
	for i < len(s) && (s[i] >= '0' && s[i] <= '9') {
		i++
	}
	if i == 0 {
		return 0, ErrInvalid
	}
	n, err := parseUint64String(s[:i])
	if err != nil {
		return 0, err
	}
	unit := strings.ToLower(strings.TrimSpace(s[i:]))
	mul := uint64(1)
	switch unit {
	case "", "b":
		mul = 1
	case "kb":
		mul = 1000
	case "mb":
		mul = 1000 * 1000
	case "gb":
		mul = 1000 * 1000 * 1000
	case "tb":
		mul = 1000 * 1000 * 1000 * 1000
	case "pb":
		mul = 1000 * 1000 * 1000 * 1000 * 1000
	case "kib":
		mul = 1 << 10
	case "mib":
		mul = 1 << 20
	case "gib":
		mul = 1 << 30
	case "tib":
		mul = 1 << 40
	case "pib":
		mul = 1 << 50
	default:
		return 0, ErrInvalid
	}
	if n > ^uint64(0)/mul {
		return 0, ErrOverflow
	}
	return n * mul, nil
}
func AppendSize(dst []byte, n uint64) []byte { return append(appendUint64(dst, n), 'B') }
func ToSizeString(n uint64) string           { var b [32]byte; return string(AppendSize(b[:0], n)) }

type Decimal interface{ String() string }
type DecimalParser interface{ ParseDecimal(string) (any, error) }

var decimalParser DecimalParser

func RegisterDecimalParser(p DecimalParser) { decimalParser = p }
func ToDecimal(v any) (any, error) {
	if decimalParser == nil {
		return nil, ErrUnsupported
	}
	s, err := ToString(v)
	if err != nil {
		return nil, err
	}
	return decimalParser.ParseDecimal(s)
}

func ToURL(v any) (url.URL, error) {
	s, err := ToString(v)
	if err != nil {
		return url.URL{}, err
	}
	u, err := url.Parse(s)
	if err != nil {
		return url.URL{}, ErrInvalid
	}
	return *u, nil
}
func ToIP(v any) (net.IP, error) {
	s, err := ToString(v)
	if err != nil {
		return nil, err
	}
	ip := net.ParseIP(s)
	if ip == nil {
		return nil, ErrInvalid
	}
	return ip, nil
}
func ToIPNet(v any) (net.IPNet, error) {
	s, err := ToString(v)
	if err != nil {
		return net.IPNet{}, err
	}
	_, n, err := net.ParseCIDR(s)
	if err != nil {
		return net.IPNet{}, ErrInvalid
	}
	return *n, nil
}
func ToAddr(v any) (netip.Addr, error) {
	s, err := ToString(v)
	if err != nil {
		return netip.Addr{}, err
	}
	a, err := netip.ParseAddr(s)
	if err != nil {
		return netip.Addr{}, ErrInvalid
	}
	return a, nil
}
func ToPrefix(v any) (netip.Prefix, error) {
	s, err := ToString(v)
	if err != nil {
		return netip.Prefix{}, err
	}
	p, err := netip.ParsePrefix(s)
	if err != nil {
		return netip.Prefix{}, ErrInvalid
	}
	return p, nil
}
func ToAddrPort(v any) (netip.AddrPort, error) {
	s, err := ToString(v)
	if err != nil {
		return netip.AddrPort{}, err
	}
	ap, err := netip.ParseAddrPort(s)
	if err != nil {
		return netip.AddrPort{}, ErrInvalid
	}
	return ap, nil
}

func ToUUIDString(v any) (string, error) {
	s, err := ToString(v)
	if err != nil {
		return "", err
	}
	if !ValidUUID(s) {
		return "", ErrInvalid
	}
	return s, nil
}
func ValidUUID(s string) bool {
	if len(s) == 32 {
		for i := 0; i < 32; i++ {
			if !isHex(s[i]) {
				return false
			}
		}
		return true
	}
	if len(s) != 36 {
		return false
	}
	for i := 0; i < 36; i++ {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if s[i] != '-' {
				return false
			}
			continue
		}
		if !isHex(s[i]) {
			return false
		}
	}
	return true
}
func isHex(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func ToTimeLayout(v any, layout string) (time.Time, error) { return ToTimeLayouts(v, layout) }
func ToTimeLayouts(v any, layouts ...string) (time.Time, error) {
	if t, ok := v.(time.Time); ok {
		return t, nil
	}
	s, err := ToString(v)
	if err != nil {
		return time.Time{}, err
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, ErrInvalid
}
func ToUnixNano(v any) (int64, error) {
	t, err := ToTime(v)
	if err != nil {
		return 0, err
	}
	return t.UnixNano(), nil
}
func ToUnixMicro(v any) (int64, error) {
	t, err := ToTime(v)
	if err != nil {
		return 0, err
	}
	return t.UnixMicro(), nil
}

func Get[T Target](data any, path string) (T, error) {
	v, err := GetAny(data, path)
	if err != nil {
		var z T
		return z, err
	}
	return To[T](v)
}
func GetAny(data any, path string) (any, error) {
	cur := reflect.ValueOf(data)
	for _, part := range strings.Split(path, ".") {
		if !cur.IsValid() {
			return nil, ErrNil
		}
		for cur.Kind() == reflect.Interface || cur.Kind() == reflect.Pointer {
			if cur.IsNil() {
				return nil, ErrNil
			}
			cur = cur.Elem()
		}
		switch cur.Kind() {
		case reflect.Map:
			key := reflect.ValueOf(part)
			if key.Type() != cur.Type().Key() {
				if key.Type().ConvertibleTo(cur.Type().Key()) {
					key = key.Convert(cur.Type().Key())
				} else {
					return nil, ErrUnsupported
				}
			}
			cur = cur.MapIndex(key)
		case reflect.Struct:
			cur = cur.FieldByNameFunc(func(n string) bool { return strings.EqualFold(n, part) || strings.EqualFold(snakeName(n), part) })
		case reflect.Slice, reflect.Array:
			idx, err := ToInt(part)
			if err != nil || idx < 0 || idx >= cur.Len() {
				return nil, ErrInvalid
			}
			cur = cur.Index(idx)
		default:
			return nil, ErrUnsupported
		}
	}
	return cur.Interface(), nil
}
func Set(data any, path string, val any) error {
	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return ErrInvalid
	}
	cur := reflect.ValueOf(data)
	if cur.Kind() != reflect.Pointer || cur.IsNil() {
		return ErrUnsupported
	}
	cur = cur.Elem()
	for _, part := range parts[:len(parts)-1] {
		for cur.Kind() == reflect.Pointer {
			cur = cur.Elem()
		}
		switch cur.Kind() {
		case reflect.Map:
			key := reflect.ValueOf(part).Convert(cur.Type().Key())
			cur = cur.MapIndex(key)
		case reflect.Struct:
			cur = cur.FieldByNameFunc(func(n string) bool { return strings.EqualFold(n, part) || strings.EqualFold(snakeName(n), part) })
		default:
			return ErrUnsupported
		}
	}
	return setReflectValue(cur.FieldByNameFunc(func(n string) bool {
		return strings.EqualFold(n, parts[len(parts)-1]) || strings.EqualFold(snakeName(n), parts[len(parts)-1])
	}), val)
}

func Query[T Target](values url.Values, key string, fallback T) T {
	if s := values.Get(key); s != "" {
		if x, err := To[T](s); err == nil {
			return x
		}
	}
	return fallback
}
func QueryInt(v url.Values, k string, d int) int    { return Query[int](v, k, d) }
func QueryBool(v url.Values, k string, d bool) bool { return Query[bool](v, k, d) }
func QueryDuration(v url.Values, k string, d time.Duration) time.Duration {
	return Query[time.Duration](v, k, d)
}
func FormDuration(v url.Values, k string, d time.Duration) time.Duration {
	return QueryDuration(v, k, d)
}
func HeaderInt(h http.Header, k string, d int) int {
	if s := h.Get(k); s != "" {
		if x, err := ToInt(s); err == nil {
			return x
		}
	}
	return d
}

func JSONNumberToInt64(n json.Number) (int64, error)     { return ToInt64(n) }
func JSONNumberToFloat64(n json.Number) (float64, error) { return ToFloat64(n) }
func JSONMapToStruct[T any](m map[string]any) (T, error) { return ToStruct[T](m) }
func NormalizeJSONNumbers(v any) any {
	switch x := v.(type) {
	case json.Number:
		if i, err := x.Int64(); err == nil {
			return i
		}
		if f, err := x.Float64(); err == nil {
			return f
		}
		return string(x)
	case map[string]any:
		for k, v := range x {
			x[k] = NormalizeJSONNumbers(v)
		}
		return x
	case []any:
		for i, v := range x {
			x[i] = NormalizeJSONNumbers(v)
		}
		return x
	default:
		return v
	}
}

type Kind uint8

const (
	KindInvalid Kind = iota
	KindBool
	KindString
	KindBytes
	KindInt
	KindUint
	KindFloat
	KindTime
	KindDuration
	KindSlice
	KindMap
	KindStruct
)

func KindOf(v any) Kind {
	switch v.(type) {
	case bool:
		return KindBool
	case string:
		return KindString
	case []byte:
		return KindBytes
	case int, int8, int16, int32, int64:
		return KindInt
	case uint, uint8, uint16, uint32, uint64, uintptr:
		return KindUint
	case float32, float64:
		return KindFloat
	case time.Time:
		return KindTime
	case time.Duration:
		return KindDuration
	}
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return KindInvalid
	}
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		return KindSlice
	case reflect.Map:
		return KindMap
	case reflect.Struct:
		return KindStruct
	}
	return KindInvalid
}
func CanConvert[T Target](v any) bool     { _, err := To[T](v); return err == nil }
func CanConvertTo(v any, target any) bool { return AsInto(v, target) == nil }

type FromAny interface{ FromAny(any) error }
type ToAny interface{ ToAny() any }
