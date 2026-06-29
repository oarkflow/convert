package convert

import (
	"errors"
	"reflect"
	"strings"
	"sync"
	"time"
)

var (
	ErrUnusedField = errors.New("convert: unused input field")
	ErrDecodeHook  = errors.New("convert: decode hook failed")
)

// DTOHook can intercept a conversion before the built-in deep pipeline runs.
// Return handled=false to fall back to the default converter.
type DTOHook func(ctx DTOContext, dst reflect.Value, src any) (handled bool, err error)

// DTOContext describes the current node in a DTO conversion.
type DTOContext struct {
	Path string
	From Kind
	To   Kind
}

// DTOOptions controls map/struct/slice data-to-object conversion.
// The zero value is production-safe and uses convert/json/query/form/header/csv tags,
// case-insensitive matching, comma-splitting for slices, defaults and validation tags.
type DTOOptions struct {
	TagPolicy       TagPolicy
	Split           []SplitOption
	WeaklyTyped     bool
	ZeroMissing     bool
	ErrorUnused     bool
	SquashAnonymous bool
	UseCache        bool
	DecodeHook      DTOHook
	Flatten         bool
	MaxDepth        int
	MaxSliceLen     int
	MaxMapSize      int
	LosslessNumbers bool
}

// DefaultDTOOptions returns the default DTO conversion settings.
func DefaultDTOOptions() DTOOptions {
	return DTOOptions{
		TagPolicy:       DefaultTagPolicy(),
		Split:           []SplitOption{WithTrimSpace()},
		WeaklyTyped:     true,
		UseCache:        true,
		SquashAnonymous: true,
		MaxDepth:        64,
		MaxSliceLen:     1 << 20,
		MaxMapSize:      1 << 20,
	}

}

type DTOOption func(*DTOOptions)

func WithDTOTags(tags ...string) DTOOption {
	return func(o *DTOOptions) {
		if len(tags) > 0 {
			o.TagPolicy.Tags = tags
		}
	}
}
func WithDTOTagPolicy(tp TagPolicy) DTOOption { return func(o *DTOOptions) { o.TagPolicy = tp } }
func WithDTOCaseSensitive() DTOOption {
	return func(o *DTOOptions) { o.TagPolicy.CaseInsensitive = false }
}
func WithDTOSplit(opts ...SplitOption) DTOOption {
	return func(o *DTOOptions) { o.Split = append(o.Split[:0], opts...) }
}
func WithDTOZeroMissing() DTOOption { return func(o *DTOOptions) { o.ZeroMissing = true } }
func WithDTOErrorUnused() DTOOption { return func(o *DTOOptions) { o.ErrorUnused = true } }
func WithDTOFlatten() DTOOption     { return func(o *DTOOptions) { o.Flatten = true } }
func WithDTOMaxDepth(n int) DTOOption {
	return func(o *DTOOptions) {
		if n > 0 {
			o.MaxDepth = n
		}
	}
}
func WithDTOMaxSliceLen(n int) DTOOption {
	return func(o *DTOOptions) {
		if n >= 0 {
			o.MaxSliceLen = n
		}
	}
}
func WithDTOMaxMapSize(n int) DTOOption {
	return func(o *DTOOptions) {
		if n >= 0 {
			o.MaxMapSize = n
		}
	}
}
func WithDTOLosslessNumbers() DTOOption { return func(o *DTOOptions) { o.LosslessNumbers = true } }
func WithDTOStrict() DTOOption {
	return func(o *DTOOptions) { o.ErrorUnused = true; o.LosslessNumbers = true; o.WeaklyTyped = false }
}
func WithDTODecodeHook(h DTOHook) DTOOption {
	return func(o *DTOOptions) { o.DecodeHook = h }
}
func WithoutDTOCache() DTOOption { return func(o *DTOOptions) { o.UseCache = false } }
func WithoutDTOSquashAnonymous() DTOOption {
	return func(o *DTOOptions) { o.SquashAnonymous = false }
}

func dtoOptionsFrom(opts []DTOOption) DTOOptions {
	o := DefaultDTOOptions()
	for _, opt := range opts {
		if opt != nil {
			opt(&o)
		}
	}
	if len(o.Split) == 0 {
		o.Split = []SplitOption{WithTrimSpace()}
	}
	return o
}

// DTO converts src into dst. dst must be a non-nil pointer. It supports:
// scalar <- scalar/string/bytes, struct <- map/struct, map <- map/struct,
// slice/array <- slice/array/string, pointers, interfaces, time.Time,
// time.Duration, FromAny hooks, default/required/validate tags and path errors.
func DTO(dst any, src any, opts ...DTOOption) error {
	if dst == nil {
		return ErrNil
	}
	rv := reflect.ValueOf(dst)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return PathError("", KindOf(src), KindInvalid, src, ErrUnsupported)
	}
	return dtoSet(rv.Elem(), src, "", dtoOptionsFrom(opts))
}

// DTOTo converts src to T through the cached deep DTO pipeline.
func DTOTo[T any](src any, opts ...DTOOption) (T, error) {
	var out T
	err := DTO(&out, src, opts...)
	return out, err
}

// MustDTOTo is like DTOTo but panics on error.
func MustDTOTo[T any](src any, opts ...DTOOption) T {
	out, err := DTOTo[T](src, opts...)
	if err != nil {
		panic(err)
	}
	return out
}

// DTOMap converts src to map[K]V. Struct sources are read from exported fields and tags.
func DTOMap[K comparable, V any](src any, opts ...DTOOption) (map[K]V, error) {
	var out map[K]V
	err := DTO(&out, src, opts...)
	return out, err
}

// DTOSlice converts src to []T. String sources are split according to WithDTOSplit.
func DTOSlice[T any](src any, opts ...DTOOption) ([]T, error) {
	var out []T
	err := DTO(&out, src, opts...)
	return out, err
}

func dtoSet(dst reflect.Value, src any, path string, opt DTOOptions) error {
	if opt.MaxDepth > 0 && dtoPathDepth(path) > opt.MaxDepth {
		return PathError(path, KindOf(src), KindOfReflect(dst), src, ErrUnsupported)
	}
	if !dst.IsValid() || !dst.CanSet() {
		return PathError(path, KindOf(src), KindInvalid, src, ErrUnsupported)
	}
	if opt.DecodeHook != nil {
		handled, err := opt.DecodeHook(DTOContext{Path: path, From: KindOf(src), To: KindOfReflect(dst)}, dst, src)
		if err != nil {
			return PathError(path, KindOf(src), KindOfReflect(dst), src, err)
		}
		if handled {
			return nil
		}
	}
	if dst.Kind() == reflect.Pointer {
		if IsNilLike(src) {
			if opt.ZeroMissing {
				dst.SetZero()
			}
			return nil
		}
		if dst.IsNil() {
			dst.Set(reflect.New(dst.Type().Elem()))
		}
		return dtoSet(dst.Elem(), src, path, opt)
	}
	if dst.CanAddr() {
		if fa, ok := dst.Addr().Interface().(FromAny); ok {
			if err := fa.FromAny(src); err != nil {
				return PathError(path, KindOf(src), KindOfReflect(dst), src, err)
			}
			return nil
		}
	}
	if src != nil {
		sv := reflect.ValueOf(src)
		if sv.IsValid() && sv.Type().AssignableTo(dst.Type()) {
			dst.Set(sv)
			return nil
		}
		if sv.IsValid() && sv.Type().ConvertibleTo(dst.Type()) && safeDirectConvertible(dst.Type(), sv.Type()) {
			dst.Set(sv.Convert(dst.Type()))
			return nil
		}
	}
	if dst.Type() == reflect.TypeOf(time.Time{}) {
		x, err := ToTime(src)
		if err != nil {
			return PathError(path, KindOf(src), KindTime, src, err)
		}
		dst.Set(reflect.ValueOf(x))
		return nil
	}
	if dst.Type() == reflect.TypeOf(time.Duration(0)) {
		x, err := ToDuration(src)
		if err != nil {
			return PathError(path, KindOf(src), KindDuration, src, err)
		}
		dst.SetInt(int64(x))
		return nil
	}
	switch dst.Kind() {
	case reflect.Interface:
		if src == nil {
			dst.SetZero()
		} else {
			dst.Set(reflect.ValueOf(src))
		}
		return nil
	case reflect.Bool:
		x, err := ToBool(src)
		if err != nil {
			return PathError(path, KindOf(src), KindBool, src, err)
		}
		dst.SetBool(x)
		return nil
	case reflect.String:
		x, err := ToString(src)
		if err != nil {
			return PathError(path, KindOf(src), KindString, src, err)
		}
		dst.SetString(x)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		x, err := ToInt64(src)
		if opt.LosslessNumbers {
			x, err = ToInt64With(src, NoPrecisionLoss())
		}
		if err != nil {
			return PathError(path, KindOf(src), KindInt, src, err)
		}
		if dst.OverflowInt(x) {
			return PathError(path, KindOf(src), KindInt, src, ErrOverflow)
		}
		dst.SetInt(x)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		x, err := ToUint64(src)
		if opt.LosslessNumbers {
			if f, ok := src.(float64); ok && mathTrunc(f) != f {
				err = ErrPrecisionLoss
			}
		}
		if err != nil {
			return PathError(path, KindOf(src), KindUint, src, err)
		}
		if dst.OverflowUint(x) {
			return PathError(path, KindOf(src), KindUint, src, ErrOverflow)
		}
		dst.SetUint(x)
		return nil
	case reflect.Float32, reflect.Float64:
		x, err := ToFloat64(src)
		if err != nil {
			return PathError(path, KindOf(src), KindFloat, src, err)
		}
		if dst.OverflowFloat(x) {
			return PathError(path, KindOf(src), KindFloat, src, ErrOverflow)
		}
		dst.SetFloat(x)
		return nil
	case reflect.Slice:
		return dtoSetSlice(dst, src, path, opt)
	case reflect.Array:
		return dtoSetArray(dst, src, path, opt)
	case reflect.Map:
		return dtoSetMap(dst, src, path, opt)
	case reflect.Struct:
		return dtoSetStruct(dst, src, path, opt)
	}
	return PathError(path, KindOf(src), KindOfReflect(dst), src, ErrUnsupported)
}

func dtoSetSlice(dst reflect.Value, src any, path string, opt DTOOptions) error {
	items, err := collectSliceItems(src, opt.Split...)
	if err != nil {
		return PathError(path, KindOf(src), KindSlice, src, err)
	}
	if opt.MaxSliceLen >= 0 && len(items) > opt.MaxSliceLen {
		return PathError(path, KindOf(src), KindSlice, src, ErrOverflow)
	}
	out := reflect.MakeSlice(dst.Type(), 0, len(items))
	for i, item := range items {
		elem := reflect.New(dst.Type().Elem()).Elem()
		if err := dtoSet(elem, item, indexPath(path, i), opt); err != nil {
			return err
		}
		out = reflect.Append(out, elem)
	}
	dst.Set(out)
	return nil
}

func dtoSetArray(dst reflect.Value, src any, path string, opt DTOOptions) error {
	items, err := collectSliceItems(src, opt.Split...)
	if err != nil {
		return PathError(path, KindOf(src), KindSlice, src, err)
	}
	if len(items) != dst.Len() {
		return PathError(path, KindOf(src), KindSlice, src, ErrInvalid)
	}
	for i, item := range items {
		if err := dtoSet(dst.Index(i), item, indexPath(path, i), opt); err != nil {
			return err
		}
	}
	return nil
}

func dtoSetMap(dst reflect.Value, src any, path string, opt DTOOptions) error {
	m, err := dtoSourceEntries(src, opt)
	if err != nil {
		return PathError(path, KindOf(src), KindMap, src, err)
	}
	if opt.MaxMapSize >= 0 && len(m) > opt.MaxMapSize {
		return PathError(path, KindOf(src), KindMap, src, ErrOverflow)
	}
	out := reflect.MakeMapWithSize(dst.Type(), len(m))
	for _, e := range m {
		key := reflect.New(dst.Type().Key()).Elem()
		if err := dtoSet(key, e.key, mapPath(path, "<key>"), opt); err != nil {
			return err
		}
		elem := reflect.New(dst.Type().Elem()).Elem()
		if err := dtoSet(elem, e.value, mapPath(path, e.name), opt); err != nil {
			return err
		}
		out.SetMapIndex(key, elem)
	}
	dst.Set(out)
	return nil
}

type dtoEntry struct {
	key   any
	name  string
	value any
}

func dtoSourceEntries(src any, opt DTOOptions) ([]dtoEntry, error) {
	if src == nil {
		return nil, ErrNil
	}
	rv := reflect.ValueOf(src)
	if !rv.IsValid() {
		return nil, ErrNil
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil, ErrNil
		}
		rv = rv.Elem()
	}
	switch rv.Kind() {
	case reflect.Map:
		out := make([]dtoEntry, 0, rv.Len())
		for _, k := range rv.MapKeys() {
			name := ToDebugString(k.Interface())
			out = append(out, dtoEntry{key: k.Interface(), name: name, value: rv.MapIndex(k).Interface()})
		}
		return out, nil
	case reflect.Struct:
		meta := dtoMetaFor(rv.Type(), opt)
		out := make([]dtoEntry, 0, len(meta.fields))
		for _, f := range meta.fields {
			fv := fieldByIndex(rv, f.index)
			if !fv.IsValid() || !fv.CanInterface() {
				continue
			}
			if f.writeonly {
				continue
			}
			out = append(out, dtoEntry{key: f.primary, name: f.primary, value: fv.Interface()})
		}
		return out, nil
	}
	return nil, ErrUnsupported
}

func dtoSetStruct(dst reflect.Value, src any, path string, opt DTOOptions) error {
	rv := reflect.ValueOf(src)
	if !rv.IsValid() {
		return PathError(path, KindInvalid, KindStruct, src, ErrNil)
	}
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return PathError(path, KindInvalid, KindStruct, src, ErrNil)
		}
		rv = rv.Elem()
	}
	if rv.Kind() == reflect.Struct && rv.Type().AssignableTo(dst.Type()) {
		dst.Set(rv)
		return nil
	}
	meta := dtoMetaFor(dst.Type(), opt)
	used := map[string]struct{}{}
	if rv.Kind() == reflect.Map {
		return dtoStructFromMap(dst, rv, path, opt, meta, used)
	}
	if rv.Kind() == reflect.Struct {
		entries, err := dtoSourceEntries(rv.Interface(), opt)
		if err != nil {
			return PathError(path, KindOf(src), KindStruct, src, err)
		}
		m := make(map[string]any, len(entries))
		for _, e := range entries {
			m[e.name] = e.value
		}
		return dtoStructFromStringMap(dst, m, path, opt, meta, used)
	}
	return PathError(path, KindOf(src), KindStruct, src, ErrUnsupported)
}

func dtoStructFromMap(dst reflect.Value, srcMap reflect.Value, path string, opt DTOOptions, meta *dtoStructMeta, used map[string]struct{}) error {
	m := make(map[string]any, srcMap.Len())
	for _, k := range srcMap.MapKeys() {
		ks, err := ToString(k.Interface())
		if err != nil {
			return PathError(mapPath(path, "<key>"), KindOf(k.Interface()), KindString, k.Interface(), err)
		}
		m[ks] = srcMap.MapIndex(k).Interface()
	}
	return dtoStructFromStringMap(dst, m, path, opt, meta, used)
}

func dtoStructFromStringMap(dst reflect.Value, m map[string]any, path string, opt DTOOptions, meta *dtoStructMeta, used map[string]struct{}) error {
	for _, fm := range meta.fields {
		field := fieldByIndex(dst, fm.index)
		if !field.IsValid() || !field.CanSet() {
			continue
		}
		val, found, usedName := dtoLookup(m, fm, opt.TagPolicy.CaseInsensitive)
		fieldPath := joinPath(path, fm.primary)
		if !found {
			if fm.defaultValue != "" {
				val, found = fm.defaultValue, true
			}
		}
		if !found && opt.Flatten {
			if v, ok := lookupFlattenValue(m, fm.names, opt.TagPolicy.CaseInsensitive); ok {
				val, found, usedName = v, true, fm.primary
			}
		}
		if !found {
			if opt.ZeroMissing {
				field.SetZero()
			}
			if fm.required {
				return PathError(fieldPath, KindInvalid, KindOfReflect(field), nil, ErrEmpty)
			}
			continue
		}
		if fm.readonly {
			used[usedName] = struct{}{}
			continue
		}
		used[usedName] = struct{}{}
		val = applyDTOTransforms(val, fm.transforms)
		if err := dtoSet(field, val, fieldPath, opt); err != nil {
			return err
		}
		if fm.validate {
			if err := validateReflectField(field, fm.structField, fieldPath); err != nil {
				return err
			}
		}
	}
	if opt.ErrorUnused {
		for k := range m {
			if _, ok := used[k]; !ok {
				return PathError(mapPath(path, k), KindOf(m[k]), KindInvalid, m[k], ErrUnusedField)
			}
		}
	}
	return nil
}

func dtoLookup(m map[string]any, f dtoFieldMeta, ci bool) (any, bool, string) {
	for _, n := range f.names {
		if v, ok := m[n]; ok {
			return v, true, n
		}
	}
	if ci {
		for _, n := range f.names {
			for k, v := range m {
				if strings.EqualFold(k, n) {
					return v, true, k
				}
			}
		}
	}
	return nil, false, ""
}

type dtoStructMeta struct{ fields []dtoFieldMeta }
type dtoFieldMeta struct {
	index        []int
	names        []string
	primary      string
	defaultValue string
	required     bool
	validate     bool
	structField  reflect.StructField
	transforms   []string
	readonly     bool
	writeonly    bool
	sensitive    bool
}

type dtoCacheKey struct {
	typ     reflect.Type
	tags    string
	useName bool
	ci      bool
	squash  bool
}

var dtoFieldCache sync.Map

func dtoMetaFor(t reflect.Type, opt DTOOptions) *dtoStructMeta {
	key := dtoCacheKey{typ: t, tags: strings.Join(opt.TagPolicy.Tags, "\x00"), useName: opt.TagPolicy.UseFieldName, ci: opt.TagPolicy.CaseInsensitive, squash: opt.SquashAnonymous}
	if opt.UseCache {
		if v, ok := dtoFieldCache.Load(key); ok {
			return v.(*dtoStructMeta)
		}
	}
	m := &dtoStructMeta{}
	buildDTOMeta(t, nil, opt, &m.fields)
	if opt.UseCache {
		actual, _ := dtoFieldCache.LoadOrStore(key, m)
		return actual.(*dtoStructMeta)
	}
	return m
}

func buildDTOMeta(t reflect.Type, prefix []int, opt DTOOptions, out *[]dtoFieldMeta) {
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" && !f.Anonymous {
			continue
		}
		names, skip := fieldLookupNames(f, opt.TagPolicy)
		for _, extra := range dtoExtendedNames(f) {
			names = appendUniqueString(names, extra)
		}
		if skip {
			continue
		}
		idx := append(append([]int(nil), prefix...), f.Index...)
		ft := f.Type
		for ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}
		if opt.SquashAnonymous && f.Anonymous && ft.Kind() == reflect.Struct && tagName(f.Tag.Get("convert")) == "" && tagName(f.Tag.Get("json")) == "" {
			buildDTOMeta(ft, idx, opt, out)
			continue
		}
		primary := firstName(names, snakeName(f.Name))
		transforms, readonly, writeonly, sensitive := dtoTagOptions(f)
		*out = append(*out, dtoFieldMeta{index: idx, names: names, primary: primary, defaultValue: f.Tag.Get("default"), required: isRequired(f), validate: f.Tag.Get("validate") != "", structField: f, transforms: transforms, readonly: readonly, writeonly: writeonly, sensitive: sensitive})
	}
}

func fieldByIndex(v reflect.Value, index []int) reflect.Value {
	for _, i := range index {
		if v.Kind() == reflect.Pointer {
			if v.IsNil() {
				if !v.CanSet() {
					return reflect.Value{}
				}
				v.Set(reflect.New(v.Type().Elem()))
			}
			v = v.Elem()
		}
		if v.Kind() != reflect.Struct || i >= v.NumField() {
			return reflect.Value{}
		}
		v = v.Field(i)
	}
	return v
}

// StructToMap converts exported struct fields to map[string]any using DTO tag names.
func StructToMap(src any, opts ...DTOOption) (map[string]any, error) {
	o := dtoOptionsFrom(opts)
	entries, err := dtoSourceEntries(src, o)
	if err != nil {
		return nil, err
	}
	out := make(map[string]any, len(entries))
	for _, e := range entries {
		out[e.name] = e.value
	}
	return out, nil
}
