package convert

import (
	"errors"
	"net"
	"reflect"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
)

// TagPolicy controls how struct fields are matched while binding/populating.
type TagPolicy struct {
	Tags            []string
	UseFieldName    bool
	CaseInsensitive bool
}

func DefaultTagPolicy() TagPolicy {
	return TagPolicy{Tags: []string{"convert", "json", "env", "query", "form", "header", "csv"}, UseFieldName: true, CaseInsensitive: true}
}

type PopulateOption func(*populatePolicy)
type populatePolicy struct {
	tag  TagPolicy
	path string
}

func WithTags(tags ...string) PopulateOption {
	return func(p *populatePolicy) {
		if len(tags) > 0 {
			p.tag.Tags = tags
		}
	}
}
func WithTagPolicy(tp TagPolicy) PopulateOption   { return func(p *populatePolicy) { p.tag = tp } }
func WithPopulatePath(path string) PopulateOption { return func(p *populatePolicy) { p.path = path } }

func populatePolicyFrom(opts []PopulateOption) populatePolicy {
	p := populatePolicy{tag: DefaultTagPolicy()}
	for _, opt := range opts {
		if opt != nil {
			opt(&p)
		}
	}
	return p
}

// PopulateWithOptions populates dst from a map/struct using a production tag stack:
// convert, json, env, query, form, header, csv, default, required and validate.
func PopulateWithOptions(dst any, src any, opts ...PopulateOption) error {
	p := populatePolicyFrom(opts)
	return populateReflectPath(reflect.ValueOf(dst), src, p.path, p.tag)
}

func populateReflectPath(dst reflect.Value, src any, path string, tagPolicy TagPolicy) error {
	if !dst.IsValid() || dst.Kind() != reflect.Pointer || dst.IsNil() {
		return PathError(path, KindOf(src), KindStruct, src, ErrUnsupported)
	}
	dv := dst.Elem()
	if dv.Kind() != reflect.Struct {
		return PathError(path, KindOf(src), KindStruct, src, ErrUnsupported)
	}
	m, err := sourceMap(src)
	if err != nil {
		return PathError(path, KindOf(src), KindStruct, src, err)
	}
	dt := dv.Type()
	for i := 0; i < dt.NumField(); i++ {
		f := dt.Field(i)
		if f.PkgPath != "" {
			continue
		}
		names, skip := fieldLookupNames(f, tagPolicy)
		if skip {
			continue
		}
		val, ok := lookupMany(m, names, tagPolicy.CaseInsensitive)
		fieldPath := joinPath(path, firstName(names, snakeName(f.Name)))
		if !ok {
			if def := f.Tag.Get("default"); def != "" {
				val, ok = def, true
			}
		}
		if !ok {
			if isRequired(f) {
				return PathError(fieldPath, KindInvalid, KindOfReflect(dv.Field(i)), nil, ErrEmpty)
			}
			continue
		}
		if err := setReflectValuePath(dv.Field(i), val, fieldPath); err != nil {
			return err
		}
		if err := validateReflectField(dv.Field(i), f, fieldPath); err != nil {
			return err
		}
	}
	return nil
}

func fieldLookupNames(f reflect.StructField, tp TagPolicy) ([]string, bool) {
	out := make([]string, 0, len(tp.Tags)+2)
	for _, tag := range tp.Tags {
		raw := f.Tag.Get(tag)
		if raw == "-" {
			return nil, true
		}
		name := tagName(raw)
		if name != "" {
			out = appendUniqueString(out, name)
		}
	}
	if tp.UseFieldName {
		out = appendUniqueString(out, snakeName(f.Name))
		out = appendUniqueString(out, f.Name)
	}
	return out, false
}
func tagName(raw string) string {
	if raw == "" {
		return ""
	}
	n, _, _ := strings.Cut(raw, ",")
	return strings.TrimSpace(n)
}
func appendUniqueString(in []string, s string) []string {
	if s == "" {
		return in
	}
	for _, x := range in {
		if x == s {
			return in
		}
	}
	return append(in, s)
}
func firstName(names []string, fallback string) string {
	if len(names) > 0 {
		return names[0]
	}
	return fallback
}
func lookupMany(m map[string]any, names []string, ci bool) (any, bool) {
	for _, n := range names {
		if v, ok := lookupName(m, n, ci); ok {
			return v, true
		}
	}
	return nil, false
}
func lookupName(m map[string]any, name string, ci bool) (any, bool) {
	if v, ok := m[name]; ok {
		return v, true
	}
	if ci {
		for k, v := range m {
			if strings.EqualFold(k, name) {
				return v, true
			}
		}
	}
	return nil, false
}
func isRequired(f reflect.StructField) bool {
	return f.Tag.Get("required") == "true" || f.Tag.Get("required") == "1" || strings.Contains(f.Tag.Get("validate"), "required")
}

func joinPath(base, elem string) string {
	if base == "" {
		return elem
	}
	if elem == "" {
		return base
	}
	return base + "." + elem
}
func indexPath(base string, i int) string {
	var b [24]byte
	s := string(appendInt64(b[:0], int64(i)))
	if base == "" {
		return "[" + s + "]"
	}
	return base + "[" + s + "]"
}
func mapPath(base, k string) string {
	if base == "" {
		return k
	}
	if k == "" {
		return base
	}
	return base + "." + k
}

// KindOfReflect returns a Kind for a reflect.Value without forcing Interface on unexported fields.
func KindOfReflect(v reflect.Value) Kind {
	if !v.IsValid() {
		return KindInvalid
	}
	if v.Type() == reflect.TypeOf(time.Time{}) {
		return KindTime
	}
	if v.Type() == reflect.TypeOf(time.Duration(0)) {
		return KindDuration
	}
	switch v.Kind() {
	case reflect.Bool:
		return KindBool
	case reflect.String:
		return KindString
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return KindInt
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return KindUint
	case reflect.Float32, reflect.Float64:
		return KindFloat
	case reflect.Slice, reflect.Array:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			return KindBytes
		}
		return KindSlice
	case reflect.Map:
		return KindMap
	case reflect.Struct:
		return KindStruct
	}
	return KindInvalid
}

// ConvertInto converts src into dst with deep slice/map/struct support and path-aware errors.
func ConvertInto(dst any, src any) error {
	return setReflectValuePath(reflect.ValueOf(dst).Elem(), src, "")
}

func setReflectValuePath(fv reflect.Value, val any, path string) error {
	if !fv.IsValid() || !fv.CanSet() {
		return PathError(path, KindOf(val), KindInvalid, val, ErrUnsupported)
	}
	if fv.Kind() == reflect.Pointer {
		if IsNilLike(val) {
			return nil
		}
		nv := reflect.New(fv.Type().Elem())
		if err := setReflectValuePath(nv.Elem(), val, path); err != nil {
			return err
		}
		fv.Set(nv)
		return nil
	}
	if fv.CanAddr() {
		if fa, ok := fv.Addr().Interface().(FromAny); ok {
			if err := fa.FromAny(val); err != nil {
				return PathError(path, KindOf(val), KindOfReflect(fv), val, err)
			}
			return nil
		}
	}
	if val != nil {
		rv := reflect.ValueOf(val)
		if rv.IsValid() && rv.Type().AssignableTo(fv.Type()) {
			fv.Set(rv)
			return nil
		}
		if rv.IsValid() && rv.Type().ConvertibleTo(fv.Type()) && safeDirectConvertible(fv.Type(), rv.Type()) {
			fv.Set(rv.Convert(fv.Type()))
			return nil
		}
	}
	if fv.Type() == reflect.TypeOf(time.Time{}) {
		x, err := ToTime(val)
		if err != nil {
			return PathError(path, KindOf(val), KindTime, val, err)
		}
		fv.Set(reflect.ValueOf(x))
		return nil
	}
	if fv.Type() == reflect.TypeOf(time.Duration(0)) {
		x, err := ToDuration(val)
		if err != nil {
			return PathError(path, KindOf(val), KindDuration, val, err)
		}
		fv.SetInt(int64(x))
		return nil
	}
	switch fv.Kind() {
	case reflect.Interface:
		if val == nil {
			fv.SetZero()
		} else {
			fv.Set(reflect.ValueOf(val))
		}
		return nil
	case reflect.Bool:
		x, err := ToBool(val)
		if err != nil {
			return PathError(path, KindOf(val), KindBool, val, err)
		}
		fv.SetBool(x)
		return nil
	case reflect.String:
		x, err := ToString(val)
		if err != nil {
			return PathError(path, KindOf(val), KindString, val, err)
		}
		fv.SetString(x)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		x, err := ToInt64(val)
		if err != nil {
			return PathError(path, KindOf(val), KindInt, val, err)
		}
		if fv.OverflowInt(x) {
			return PathError(path, KindOf(val), KindInt, val, ErrOverflow)
		}
		fv.SetInt(x)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		x, err := ToUint64(val)
		if err != nil {
			return PathError(path, KindOf(val), KindUint, val, err)
		}
		if fv.OverflowUint(x) {
			return PathError(path, KindOf(val), KindUint, val, ErrOverflow)
		}
		fv.SetUint(x)
		return nil
	case reflect.Float32, reflect.Float64:
		x, err := ToFloat64(val)
		if err != nil {
			return PathError(path, KindOf(val), KindFloat, val, err)
		}
		if fv.OverflowFloat(x) {
			return PathError(path, KindOf(val), KindFloat, val, ErrOverflow)
		}
		fv.SetFloat(x)
		return nil
	case reflect.Slice:
		return setSliceReflectPath(fv, val, path)
	case reflect.Array:
		return setArrayReflectPath(fv, val, path)
	case reflect.Map:
		return setMapReflectPath(fv, val, path)
	case reflect.Struct:
		return populateReflectPath(fv.Addr(), val, path, DefaultTagPolicy())
	}
	return PathError(path, KindOf(val), KindOfReflect(fv), val, ErrUnsupported)
}
func safeDirectConvertible(dst, src reflect.Type) bool {
	return dst.Kind() == src.Kind() && dst.Kind() != reflect.Struct && dst.Kind() != reflect.Map && dst.Kind() != reflect.Slice && dst.Kind() != reflect.Array
}

func collectSliceItems(v any, opts ...SplitOption) ([]any, error) {
	sp := splitPolicyFrom(opts)
	switch x := v.(type) {
	case nil:
		return nil, ErrNil
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
		return collectSliceItems(bytesToString(x), opts...)
	case []any:
		out := make([]any, 0, len(x))
		seen := map[string]struct{}{}
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
	}
	rv := reflect.ValueOf(v)
	if !rv.IsValid() || (rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array) {
		return nil, ErrUnsupported
	}
	out := make([]any, 0, rv.Len())
	seen := map[string]struct{}{}
	for i := 0; i < rv.Len(); i++ {
		it := rv.Index(i).Interface()
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
}

func setSliceReflectPath(fv reflect.Value, val any, path string) error {
	items, err := collectSliceItems(val, WithTrimSpace())
	if err != nil {
		return PathError(path, KindOf(val), KindSlice, val, err)
	}
	out := reflect.MakeSlice(fv.Type(), 0, len(items))
	for i, it := range items {
		elem := reflect.New(fv.Type().Elem()).Elem()
		if err := setReflectValuePath(elem, it, indexPath(path, i)); err != nil {
			return err
		}
		out = reflect.Append(out, elem)
	}
	fv.Set(out)
	return nil
}
func setArrayReflectPath(fv reflect.Value, val any, path string) error {
	items, err := collectSliceItems(val, WithTrimSpace())
	if err != nil {
		return PathError(path, KindOf(val), KindSlice, val, err)
	}
	if len(items) != fv.Len() {
		return PathError(path, KindOf(val), KindSlice, val, ErrInvalid)
	}
	for i, it := range items {
		if err := setReflectValuePath(fv.Index(i), it, indexPath(path, i)); err != nil {
			return err
		}
	}
	return nil
}
func setMapReflectPath(fv reflect.Value, val any, path string) error {
	rv := reflect.ValueOf(val)
	if !rv.IsValid() || rv.Kind() != reflect.Map {
		return PathError(path, KindOf(val), KindMap, val, ErrUnsupported)
	}
	out := reflect.MakeMapWithSize(fv.Type(), rv.Len())
	for _, mk := range rv.MapKeys() {
		key := reflect.New(fv.Type().Key()).Elem()
		if err := setReflectValuePath(key, mk.Interface(), mapPath(path, "<key>")); err != nil {
			return err
		}
		elem := reflect.New(fv.Type().Elem()).Elem()
		ks := ToDebugString(mk.Interface())
		if err := setReflectValuePath(elem, rv.MapIndex(mk).Interface(), mapPath(path, ks)); err != nil {
			return err
		}
		out.SetMapIndex(key, elem)
	}
	fv.Set(out)
	return nil
}

// ToDeepSlice converts any slice/array/string/[]any into []T. T can be scalar, struct, map, slice or a registered custom type.
func ToDeepSlice[T any](v any, opts ...SplitOption) ([]T, error) { return ToTypedSlice[T](v, opts...) }

// ToSliceAny is an alias for ToDeepSlice for discoverability.
func ToSliceAny[T any](v any, opts ...SplitOption) ([]T, error) { return ToTypedSlice[T](v, opts...) }

// ToTypedMap converts maps deeply. Keys and values can be scalar, custom, struct, map or slice types.
func ToTypedMap[K comparable, V any](v any) (map[K]V, error) {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() || rv.Kind() != reflect.Map {
		return nil, ErrUnsupported
	}
	out := make(map[K]V, rv.Len())
	for _, mk := range rv.MapKeys() {
		var k K
		if err := setReflectValuePath(reflect.ValueOf(&k).Elem(), mk.Interface(), "<key>"); err != nil {
			return nil, err
		}
		var elem V
		if err := setReflectValuePath(reflect.ValueOf(&elem).Elem(), rv.MapIndex(mk).Interface(), ToDebugString(mk.Interface())); err != nil {
			return nil, err
		}
		out[k] = elem
	}
	return out, nil
}

// DeepConvert converts src to T using the deep reflection pipeline and path-aware errors.
func DeepConvert[T any](src any) (T, error) {
	var out T
	err := setReflectValuePath(reflect.ValueOf(&out).Elem(), src, "")
	return out, err
}

// Additional validators.
func Slug() Validator[string] { return Regex(`^[a-z0-9]+(?:-[a-z0-9]+)*$`) }
func SemVer() Validator[string] {
	return Regex(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)
}
func Domain() Validator[string] {
	return func(v string) error {
		if validDomain(v, false) {
			return nil
		}
		return ErrValidation
	}
}
func FQDN() Validator[string] {
	return func(v string) error {
		if validDomain(v, true) {
			return nil
		}
		return ErrValidation
	}
}
func MACAddress() Validator[string] {
	return func(v string) error {
		if _, err := net.ParseMAC(v); err == nil {
			return nil
		}
		return ErrValidation
	}
}
func Phone() Validator[string] {
	return func(v string) error {
		digits := 0
		for i, r := range v {
			if unicode.IsDigit(r) {
				digits++
				continue
			}
			if strings.ContainsRune("+-.() ", r) && !(r == '+' && i > 0) {
				continue
			}
			return ErrValidation
		}
		if digits >= 7 && digits <= 15 {
			return nil
		}
		return ErrValidation
	}
}
func CountryCode() Validator[string] {
	return func(v string) error {
		if len(v) == 2 && isASCIIAlpha(v[0]) && isASCIIAlpha(v[1]) {
			return nil
		}
		return ErrValidation
	}
}
func CurrencyCode() Validator[string] {
	return func(v string) error {
		if len(v) == 3 && isASCIIUpper(v[0]) && isASCIIUpper(v[1]) && isASCIIUpper(v[2]) {
			return nil
		}
		return ErrValidation
	}
}
func TimeZoneName() Validator[string] {
	return func(v string) error {
		if _, err := time.LoadLocation(v); err == nil {
			return nil
		}
		return ErrValidation
	}
}
func SafeFilePath() Validator[string] {
	return func(v string) error {
		if v == "" || strings.Contains(v, "\x00") || strings.Contains(v, "..") || strings.HasPrefix(v, "/") || strings.HasPrefix(v, "~") {
			return ErrValidation
		}
		return nil
	}
}
func Charset(chars string) Validator[string] {
	allowed := map[rune]struct{}{}
	for _, r := range chars {
		allowed[r] = struct{}{}
	}
	return func(v string) error {
		for _, r := range v {
			if _, ok := allowed[r]; !ok {
				return ErrValidation
			}
		}
		return nil
	}
}
func ReservedUsername(names ...string) Validator[string] {
	set := map[string]struct{}{}
	if len(names) == 0 {
		names = []string{"admin", "root", "system", "support", "null", "undefined", "api", "www"}
	}
	for _, n := range names {
		set[strings.ToLower(n)] = struct{}{}
	}
	return func(v string) error {
		if _, ok := set[strings.ToLower(v)]; ok {
			return ErrValidation
		}
		return nil
	}
}
func isASCIIAlpha(c byte) bool { return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') }
func isASCIIUpper(c byte) bool { return c >= 'A' && c <= 'Z' }
func validDomain(s string, fqdn bool) bool {
	s = strings.TrimSuffix(s, ".")
	if fqdn && !strings.Contains(s, ".") {
		return false
	}
	if len(s) < 1 || len(s) > 253 {
		return false
	}
	labels := strings.Split(s, ".")
	for _, l := range labels {
		if len(l) < 1 || len(l) > 63 || l[0] == '-' || l[len(l)-1] == '-' {
			return false
		}
		for i := 0; i < len(l); i++ {
			c := l[i]
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-') {
				return false
			}
		}
	}
	return true
}

// Locale controls locale-aware number parsing.
type Locale struct {
	Decimal  byte
	Group    byte
	Numerals string
}

var LocaleEN = Locale{Decimal: '.', Group: ',', Numerals: "0123456789"}
var LocaleEU = Locale{Decimal: ',', Group: '.', Numerals: "0123456789"}
var LocaleSpace = Locale{Decimal: '.', Group: ' ', Numerals: "0123456789"}

func ParseLocaleFloat64(s string, loc Locale) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, ErrEmpty
	}
	neg := false
	i := 0
	if s[0] == '-' || s[0] == '+' {
		neg = s[0] == '-'
		i++
	}
	var intPart uint64
	var fracPart float64
	fracScale := 0.1
	seenDigit := false
	seenDecimal := false
	for i < len(s) {
		r, sz := rune(s[i]), 1
		if s[i] >= 0x80 {
			r, sz = utf8.DecodeRuneInString(s[i:])
		}
		if byteOrZero(r) == loc.Group && loc.Group != 0 {
			i += sz
			continue
		}
		if byteOrZero(r) == loc.Decimal && loc.Decimal != 0 {
			if seenDecimal {
				return 0, ErrInvalid
			}
			seenDecimal = true
			i += sz
			continue
		}
		d := digitValue(r, loc.Numerals)
		if d < 0 {
			return 0, ErrInvalid
		}
		seenDigit = true
		if !seenDecimal {
			intPart = intPart*10 + uint64(d)
		} else {
			fracPart += float64(d) * fracScale
			fracScale *= 0.1
		}
		i += sz
	}
	if !seenDigit {
		return 0, ErrInvalid
	}
	out := float64(intPart) + fracPart
	if neg {
		out = -out
	}
	return out, nil
}
func ParseLocaleInt64(s string, loc Locale) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, ErrEmpty
	}
	neg := false
	i := 0
	if s[0] == '-' || s[0] == '+' {
		neg = s[0] == '-'
		i++
	}
	var u uint64
	seenDigit := false
	for i < len(s) {
		r, sz := rune(s[i]), 1
		if s[i] >= 0x80 {
			r, sz = utf8.DecodeRuneInString(s[i:])
		}
		if byteOrZero(r) == loc.Group && loc.Group != 0 {
			i += sz
			continue
		}
		if byteOrZero(r) == loc.Decimal && loc.Decimal != 0 {
			return 0, ErrInvalid
		}
		d := digitValue(r, loc.Numerals)
		if d < 0 {
			return 0, ErrInvalid
		}
		seenDigit = true
		u = u*10 + uint64(d)
		i += sz
	}
	if !seenDigit {
		return 0, ErrInvalid
	}
	if neg {
		if u > 1<<63 {
			return 0, ErrOverflow
		}
		return -int64(u), nil
	}
	if u > uint64(^uint64(0)>>1) {
		return 0, ErrOverflow
	}
	return int64(u), nil
}
func ToFloat64Locale(v any, loc Locale) (float64, error) {
	s, err := ToString(v)
	if err != nil {
		return 0, err
	}
	return ParseLocaleFloat64(s, loc)
}
func ToInt64Locale(v any, loc Locale) (int64, error) {
	s, err := ToString(v)
	if err != nil {
		return 0, err
	}
	return ParseLocaleInt64(s, loc)
}
func byteOrZero(r rune) byte {
	if r >= 0 && r < 128 {
		return byte(r)
	}
	return 0
}
func digitValue(r rune, numerals string) int {
	if r >= '0' && r <= '9' {
		return int(r - '0')
	}
	return localizedDigit(r, numerals)
}
func normalizeLocaleNumber(s string, loc Locale, allowDecimal bool) (string, error) {
	// Kept for compatibility with internal callers; parsing APIs avoid this allocation.
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ErrEmpty
	}
	var b strings.Builder
	b.Grow(len(s))
	decimals := 0
	for _, r := range s {
		c := byteOrZero(r)
		switch {
		case c == loc.Group && loc.Group != 0:
			continue
		case c == loc.Decimal && loc.Decimal != 0:
			if !allowDecimal {
				return "", ErrInvalid
			}
			decimals++
			if decimals > 1 {
				return "", ErrInvalid
			}
			b.WriteByte('.')
		case r == '-' || r == '+':
			if b.Len() != 0 {
				return "", ErrInvalid
			}
			b.WriteRune(r)
		default:
			d := digitValue(r, loc.Numerals)
			if d < 0 {
				return "", ErrInvalid
			}
			b.WriteByte(byte('0' + d))
		}
	}
	return b.String(), nil
}
func localizedDigit(r rune, numerals string) int {
	if numerals == "" {
		return -1
	}
	i := 0
	for _, nr := range numerals {
		if nr == r {
			return i
		}
		i++
	}
	return -1
}

// Typed decimal adapters keep the core dependency-free while allowing exact decimal support.
type DecimalAdapter[T any] struct {
	Parse  func(string) (T, error)
	Format func(T) string
}

var typedDecimal sync.Map

func RegisterDecimalAdapter[T any](a DecimalAdapter[T]) {
	var z T
	typedDecimal.Store(reflect.TypeOf(z), a)
}
func UnregisterDecimalAdapter[T any]() { var z T; typedDecimal.Delete(reflect.TypeOf(z)) }
func ToDecimalTyped[T any](v any) (T, error) {
	var z T
	if e, ok := typedDecimal.Load(reflect.TypeOf(z)); ok {
		s, err := ToString(v)
		if err != nil {
			return z, err
		}
		return e.(DecimalAdapter[T]).Parse(s)
	}
	return z, ErrUnsupported
}
func DecimalToString[T any](v T) (string, error) {
	if e, ok := typedDecimal.Load(reflect.TypeOf(v)); ok {
		return e.(DecimalAdapter[T]).Format(v), nil
	}
	if d, ok := any(v).(Decimal); ok {
		return d.String(), nil
	}
	return "", ErrUnsupported
}

// ValidateStruct applies validate tags on already-populated structs.
func ValidateStruct(v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return ErrNil
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return ErrUnsupported
	}
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if f.PkgPath != "" {
			continue
		}
		if err := validateReflectField(rv.Field(i), f, snakeName(f.Name)); err != nil {
			return err
		}
	}
	return nil
}
func validateReflectField(v reflect.Value, f reflect.StructField, path string) error {
	tag := f.Tag.Get("validate")
	if tag == "" {
		return nil
	}
	if strings.Contains(tag, "required") && isZeroReflect(v) {
		return PathError(path, KindInvalid, KindOfReflect(v), nil, ErrEmpty)
	}
	if !v.CanInterface() {
		return nil
	}
	s, _ := ToString(v.Interface())
	for _, rule := range strings.Split(tag, ",") {
		rule = strings.TrimSpace(rule)
		if rule == "" || rule == "required" {
			continue
		}
		var err error
		switch rule {
		case "email":
			err = Email()(s)
		case "url":
			err = URLValidator()(s)
		case "hostname":
			err = Hostname()(s)
		case "ip":
			err = IP()(s)
		case "cidr":
			err = CIDR()(s)
		case "uuid":
			err = UUID()(s)
		case "slug":
			err = Slug()(s)
		case "semver":
			err = SemVer()(s)
		case "domain":
			err = Domain()(s)
		case "fqdn":
			err = FQDN()(s)
		case "mac":
			err = MACAddress()(s)
		case "phone":
			err = Phone()(s)
		case "country":
			err = CountryCode()(s)
		case "currency":
			err = CurrencyCode()(s)
		case "timezone":
			err = TimeZoneName()(s)
		case "safe_path":
			err = SafeFilePath()(s)
		}
		if err != nil {
			return PathError(path, KindString, KindString, s, err)
		}
	}
	return nil
}
func isZeroReflect(v reflect.Value) bool { return !v.IsValid() || v.IsZero() }

// IsValidation reports validation errors regardless of wrapping.
func IsValidation(err error) bool { return errors.Is(err, ErrValidation) }
