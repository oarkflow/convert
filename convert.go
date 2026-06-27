// Package convert provides high-performance, reflection-free conversions for Go scalar values,
// time.Time and time.Duration.
//
// Design goals:
//   - no reflect
//   - no fmt
//   - allocation-free successful primitive parse paths
//   - explicit overflow and invalid-value errors
//   - generic helpers for target-type based conversion
package convert

import (
	"encoding/json"
	"errors"
	"math"
	"strconv"
	"time"
	"unsafe"
)

// Scalar is the set of built-in scalar types handled by ToScalar.
type Scalar interface {
	~bool |
		~string |
		~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64
}

// Target is the complete generic conversion target set.
// time.Duration is included by Scalar through its ~int64 underlying type;
// time.Time is listed separately because it is a struct.
type Target interface {
	Scalar | time.Time
}

var (
	ErrUnsupported = errors.New("convert: unsupported conversion")
	ErrOverflow    = errors.New("convert: overflow")
	ErrInvalid     = errors.New("convert: invalid value")
	ErrNil         = errors.New("convert: nil value")
)

// DurationUnit controls numeric-to-duration conversion.
type DurationUnit int64

const (
	Nanosecond  DurationUnit = DurationUnit(time.Nanosecond)
	Microsecond DurationUnit = DurationUnit(time.Microsecond)
	Millisecond DurationUnit = DurationUnit(time.Millisecond)
	Second      DurationUnit = DurationUnit(time.Second)
	Minute      DurationUnit = DurationUnit(time.Minute)
	Hour        DurationUnit = DurationUnit(time.Hour)
)

// To converts v to T. T may be a built-in scalar type, time.Duration or time.Time.
//
// For named scalar aliases, prefer ToScalar[T]. Example:
//
//	type Port int
//	p, err := convert.ToScalar[Port]("8080")
//
// To deliberately does not use reflection, so it does not support arbitrary structs.
func To[T Target](v any) (T, error) {
	var zero T

	// This form is intentionally value-witness based. In benchmarks it is faster
	// than pointer-witness switching for instantiated generic hot paths.
	switch any(zero).(type) {
	case bool:
		x, err := ToBool(v)
		return recast[T](x), err
	case string:
		x, err := ToString(v)
		return recast[T](x), err
	case int:
		switch x := v.(type) {
		case int:
			return recast[T](x), nil
		case string:
			i, err := toIntString(x)
			return recast[T](i), err
		case []byte:
			i, err := toIntString(bytesToString(x))
			return recast[T](i), err
		}
		x, err := ToInt(v)
		return recast[T](x), err
	case int8:
		x, err := ToInt8(v)
		return recast[T](x), err
	case int16:
		x, err := ToInt16(v)
		return recast[T](x), err
	case int32:
		x, err := ToInt32(v)
		return recast[T](x), err
	case int64:
		x, err := ToInt64(v)
		return recast[T](x), err
	case uint:
		x, err := ToUint(v)
		return recast[T](x), err
	case uint8:
		x, err := ToUint8(v)
		return recast[T](x), err
	case uint16:
		x, err := ToUint16(v)
		return recast[T](x), err
	case uint32:
		x, err := ToUint32(v)
		return recast[T](x), err
	case uint64:
		x, err := ToUint64(v)
		return recast[T](x), err
	case uintptr:
		x, err := ToUintptr(v)
		return recast[T](x), err
	case float32:
		x, err := ToFloat32(v)
		return recast[T](x), err
	case float64:
		x, err := ToFloat64(v)
		return recast[T](x), err
	case time.Duration:
		x, err := ToDuration(v)
		return recast[T](x), err
	case time.Time:
		x, err := ToTime(v)
		return recast[T](x), err
	default:
		return zero, ErrUnsupported
	}
}

// ToScalar converts v to a built-in scalar type. For named aliases use the
// specific typed helpers such as ToIntLike[Port].
func ToScalar[T Scalar](v any) (T, error) {
	return To[T](v)
}

func ToBoolLike[T ~bool](v any) (T, error)       { x, err := ToBool(v); return T(x), err }
func ToStringLike[T ~string](v any) (T, error)   { x, err := ToString(v); return T(x), err }
func ToIntLike[T ~int](v any) (T, error)         { x, err := ToInt(v); return T(x), err }
func ToInt8Like[T ~int8](v any) (T, error)       { x, err := ToInt8(v); return T(x), err }
func ToInt16Like[T ~int16](v any) (T, error)     { x, err := ToInt16(v); return T(x), err }
func ToInt32Like[T ~int32](v any) (T, error)     { x, err := ToInt32(v); return T(x), err }
func ToInt64Like[T ~int64](v any) (T, error)     { x, err := ToInt64(v); return T(x), err }
func ToUintLike[T ~uint](v any) (T, error)       { x, err := ToUint(v); return T(x), err }
func ToUint8Like[T ~uint8](v any) (T, error)     { x, err := ToUint8(v); return T(x), err }
func ToUint16Like[T ~uint16](v any) (T, error)   { x, err := ToUint16(v); return T(x), err }
func ToUint32Like[T ~uint32](v any) (T, error)   { x, err := ToUint32(v); return T(x), err }
func ToUint64Like[T ~uint64](v any) (T, error)   { x, err := ToUint64(v); return T(x), err }
func ToUintptrLike[T ~uintptr](v any) (T, error) { x, err := ToUintptr(v); return T(x), err }
func ToFloat32Like[T ~float32](v any) (T, error) { x, err := ToFloat32(v); return T(x), err }
func ToFloat64Like[T ~float64](v any) (T, error) { x, err := ToFloat64(v); return T(x), err }

func recast[T any, V any](v V) T {
	return *(*T)(unsafe.Pointer(&v))
}

// As converts src to the same concrete type as target and returns it as any.
// The value of target is ignored; only its type is used.
//
// Examples:
//
//	var a = 1.2
//	var b = "5"
//	x, err := convert.As(a, b) // x is string("1.2")
//
// Pointer targets are supported too:
//
//	var n int
//	x, err := convert.As("42", &n) // x is int(42)
func As(src any, target any) (any, error) {
	if target == nil {
		return nil, ErrNil
	}

	switch target.(type) {
	case bool, *bool:
		return ToBool(src)
	case string, *string:
		return ToString(src)
	case int, *int:
		return ToInt(src)
	case int8, *int8:
		return ToInt8(src)
	case int16, *int16:
		return ToInt16(src)
	case int32, *int32:
		return ToInt32(src)
	case int64, *int64:
		return ToInt64(src)
	case uint, *uint:
		return ToUint(src)
	case uint8, *uint8:
		return ToUint8(src)
	case uint16, *uint16:
		return ToUint16(src)
	case uint32, *uint32:
		return ToUint32(src)
	case uint64, *uint64:
		return ToUint64(src)
	case uintptr, *uintptr:
		return ToUintptr(src)
	case float32, *float32:
		return ToFloat32(src)
	case float64, *float64:
		return ToFloat64(src)
	case time.Duration, *time.Duration:
		return ToDuration(src)
	case time.Time, *time.Time:
		return ToTime(src)
	default:
		return nil, ErrUnsupported
	}
}

// AsTo converts src to T using target as the type guide.
// This is useful when the target value already exists and should define conversion behavior.

// AsInto converts src into the value pointed to by target.
// This is the allocation-free target-based API for hot paths because it avoids
// returning an interface{} box from As.
func AsInto(src any, target any) error {
	if target == nil {
		return ErrNil
	}
	switch p := target.(type) {
	case *bool:
		x, err := ToBool(src)
		if err != nil {
			return err
		}
		*p = x
		return nil
	case *string:
		x, err := ToString(src)
		if err != nil {
			return err
		}
		*p = x
		return nil
	case *int:
		x, err := ToInt(src)
		if err != nil {
			return err
		}
		*p = x
		return nil
	case *int8:
		x, err := ToInt8(src)
		if err != nil {
			return err
		}
		*p = x
		return nil
	case *int16:
		x, err := ToInt16(src)
		if err != nil {
			return err
		}
		*p = x
		return nil
	case *int32:
		x, err := ToInt32(src)
		if err != nil {
			return err
		}
		*p = x
		return nil
	case *int64:
		x, err := ToInt64(src)
		if err != nil {
			return err
		}
		*p = x
		return nil
	case *uint:
		x, err := ToUint(src)
		if err != nil {
			return err
		}
		*p = x
		return nil
	case *uint8:
		x, err := ToUint8(src)
		if err != nil {
			return err
		}
		*p = x
		return nil
	case *uint16:
		x, err := ToUint16(src)
		if err != nil {
			return err
		}
		*p = x
		return nil
	case *uint32:
		x, err := ToUint32(src)
		if err != nil {
			return err
		}
		*p = x
		return nil
	case *uint64:
		x, err := ToUint64(src)
		if err != nil {
			return err
		}
		*p = x
		return nil
	case *uintptr:
		x, err := ToUintptr(src)
		if err != nil {
			return err
		}
		*p = x
		return nil
	case *float32:
		x, err := ToFloat32(src)
		if err != nil {
			return err
		}
		*p = x
		return nil
	case *float64:
		x, err := ToFloat64(src)
		if err != nil {
			return err
		}
		*p = x
		return nil
	case *time.Duration:
		x, err := ToDuration(src)
		if err != nil {
			return err
		}
		*p = x
		return nil
	case *time.Time:
		x, err := ToTime(src)
		if err != nil {
			return err
		}
		*p = x
		return nil
	default:
		return ErrUnsupported
	}
}

func AsTo[T Target](src any, target T) (T, error) {
	_ = target
	var zero T

	switch any(zero).(type) {
	case bool:
		x, err := ToBool(src)
		return recast[T](x), err
	case string:
		x, err := ToString(src)
		return recast[T](x), err
	case int:
		switch x := src.(type) {
		case int:
			return recast[T](x), nil
		case string:
			i, err := toIntString(x)
			return recast[T](i), err
		case []byte:
			i, err := toIntString(bytesToString(x))
			return recast[T](i), err
		}
		x, err := ToInt(src)
		return recast[T](x), err
	case int8:
		x, err := ToInt8(src)
		return recast[T](x), err
	case int16:
		x, err := ToInt16(src)
		return recast[T](x), err
	case int32:
		x, err := ToInt32(src)
		return recast[T](x), err
	case int64:
		x, err := ToInt64(src)
		return recast[T](x), err
	case uint:
		x, err := ToUint(src)
		return recast[T](x), err
	case uint8:
		x, err := ToUint8(src)
		return recast[T](x), err
	case uint16:
		x, err := ToUint16(src)
		return recast[T](x), err
	case uint32:
		x, err := ToUint32(src)
		return recast[T](x), err
	case uint64:
		x, err := ToUint64(src)
		return recast[T](x), err
	case uintptr:
		x, err := ToUintptr(src)
		return recast[T](x), err
	case float32:
		x, err := ToFloat32(src)
		return recast[T](x), err
	case float64:
		x, err := ToFloat64(src)
		return recast[T](x), err
	case time.Duration:
		x, err := ToDuration(src)
		return recast[T](x), err
	case time.Time:
		x, err := ToTime(src)
		return recast[T](x), err
	default:
		return zero, ErrUnsupported
	}
}

func Must[T Target](v any) T {
	x, err := To[T](v)
	if err != nil {
		panic(err)
	}
	return x
}

func Default[T Target](v any, fallback T) T {
	x, err := To[T](v)
	if err != nil {
		return fallback
	}
	return x
}

func ToBool(v any) (bool, error) {
	switch x := v.(type) {
	case bool:
		return x, nil
	case string:
		return parseBoolString(x)
	case []byte:
		return parseBoolString(bytesToString(x))
	case json.Number:
		return parseBoolString(string(x))
	case time.Duration:
		return x != 0, nil
	case time.Time:
		return !x.IsZero(), nil
	case int:
		return x != 0, nil
	case int8:
		return x != 0, nil
	case int16:
		return x != 0, nil
	case int32:
		return x != 0, nil
	case int64:
		return x != 0, nil
	case uint:
		return x != 0, nil
	case uint8:
		return x != 0, nil
	case uint16:
		return x != 0, nil
	case uint32:
		return x != 0, nil
	case uint64:
		return x != 0, nil
	case uintptr:
		return x != 0, nil
	case float32:
		if math.IsNaN(float64(x)) {
			return false, ErrInvalid
		}
		return x != 0, nil
	case float64:
		if math.IsNaN(x) {
			return false, ErrInvalid
		}
		return x != 0, nil
	default:
		return false, ErrUnsupported
	}
}

func parseBoolString(s string) (bool, error) {
	// Allocation-free ASCII parser. It intentionally does not trim spaces;
	// callers that need loose input can normalize before calling convert.
	switch len(s) {
	case 1:
		switch s[0] {
		case '1', 't', 'T', 'y', 'Y':
			return true, nil
		case '0', 'f', 'F', 'n', 'N':
			return false, nil
		}
	case 2:
		c0, c1 := lower(s[0]), lower(s[1])
		if c0 == 'o' && c1 == 'n' {
			return true, nil
		}
		if c0 == 'n' && c1 == 'o' {
			return false, nil
		}
	case 3:
		c0, c1, c2 := lower(s[0]), lower(s[1]), lower(s[2])
		if c0 == 'y' && c1 == 'e' && c2 == 's' {
			return true, nil
		}
		if c0 == 'o' && c1 == 'f' && c2 == 'f' {
			return false, nil
		}
	case 4:
		if lower(s[0]) == 't' && lower(s[1]) == 'r' && lower(s[2]) == 'u' && lower(s[3]) == 'e' {
			return true, nil
		}
	case 5:
		if lower(s[0]) == 'f' && lower(s[1]) == 'a' && lower(s[2]) == 'l' && lower(s[3]) == 's' && lower(s[4]) == 'e' {
			return false, nil
		}
	}
	return false, ErrInvalid
}

func ToString(v any) (string, error) {
	switch x := v.(type) {
	case string:
		return x, nil
	case []byte:
		return bytesToString(x), nil
	case json.Number:
		return string(x), nil
	case bool:
		if x {
			return "true", nil
		}
		return "false", nil
	case time.Duration:
		return formatDurationString(x), nil
	case time.Time:
		return x.Format(time.RFC3339Nano), nil
	case int:
		return strconv.FormatInt(int64(x), 10), nil
	case int8:
		return strconv.FormatInt(int64(x), 10), nil
	case int16:
		return strconv.FormatInt(int64(x), 10), nil
	case int32:
		return strconv.FormatInt(int64(x), 10), nil
	case int64:
		return strconv.FormatInt(x, 10), nil
	case uint:
		return strconv.FormatUint(uint64(x), 10), nil
	case uint8:
		return strconv.FormatUint(uint64(x), 10), nil
	case uint16:
		return strconv.FormatUint(uint64(x), 10), nil
	case uint32:
		return strconv.FormatUint(uint64(x), 10), nil
	case uint64:
		return strconv.FormatUint(x, 10), nil
	case uintptr:
		return strconv.FormatUint(uint64(x), 10), nil
	case float32:
		return formatFloat32String(x), nil
	case float64:
		return formatFloat64String(x), nil
	default:
		return "", ErrUnsupported
	}
}

// AppendString appends the string representation of v into dst.
// Use it when you need numeric/bool/time formatting without creating an intermediate string.
func AppendString(dst []byte, v any) ([]byte, error) {
	switch x := v.(type) {
	case string:
		return append(dst, x...), nil
	case []byte:
		return append(dst, x...), nil
	case json.Number:
		return append(dst, string(x)...), nil
	case bool:
		return strconv.AppendBool(dst, x), nil
	case time.Duration:
		return appendDuration(dst, x), nil
	case time.Time:
		return x.AppendFormat(dst, time.RFC3339Nano), nil
	case int:
		return appendInt64(dst, int64(x)), nil
	case int8:
		return appendInt64(dst, int64(x)), nil
	case int16:
		return appendInt64(dst, int64(x)), nil
	case int32:
		return appendInt64(dst, int64(x)), nil
	case int64:
		return appendInt64(dst, x), nil
	case uint:
		return appendUint64(dst, uint64(x)), nil
	case uint8:
		return appendUint64(dst, uint64(x)), nil
	case uint16:
		return appendUint64(dst, uint64(x)), nil
	case uint32:
		return appendUint64(dst, uint64(x)), nil
	case uint64:
		return appendUint64(dst, x), nil
	case uintptr:
		return appendUint64(dst, uint64(x)), nil
	case float32:
		if out, ok := appendFloat64Simple(dst, float64(x)); ok {
			return out, nil
		}
		return strconv.AppendFloat(dst, float64(x), 'g', -1, 32), nil
	case float64:
		if out, ok := appendFloat64Simple(dst, x); ok {
			return out, nil
		}
		return strconv.AppendFloat(dst, x, 'g', -1, 64), nil
	default:
		return dst, ErrUnsupported
	}
}

func formatDurationString(d time.Duration) string {
	var buf [32]byte
	out := appendDuration(buf[:0], d)
	return string(out)
}

func appendDuration(dst []byte, d time.Duration) []byte {
	if d == 0 {
		return append(dst, '0', 's')
	}
	if d < 0 {
		dst = append(dst, '-')
		if d == time.Duration(math.MinInt64) {
			// Avoid overflow when negating MinInt64. This exact value is rare;
			// using the stdlib keeps correctness without affecting hot paths.
			return append(dst, time.Duration(math.MaxInt64).String()...)
		}
		d = -d
	}

	// Keep compatibility with the standard representation for common integer units.
	if d%time.Hour == 0 {
		return append(appendUint64(dst, uint64(d/time.Hour)), 'h')
	}
	if d%time.Minute == 0 {
		return append(appendUint64(dst, uint64(d/time.Minute)), 'm')
	}
	if d%time.Second == 0 {
		return append(appendUint64(dst, uint64(d/time.Second)), 's')
	}
	if d%time.Millisecond == 0 {
		return append(appendUint64(dst, uint64(d/time.Millisecond)), 'm', 's')
	}
	if d%time.Microsecond == 0 {
		return append(appendUint64(dst, uint64(d/time.Microsecond)), 'µ', 's')
	}
	return append(appendUint64(dst, uint64(d)), 'n', 's')
}

func appendInt64(dst []byte, i int64) []byte {
	if i < 0 {
		if i == math.MinInt64 {
			return append(dst, "-9223372036854775808"...)
		}
		dst = append(dst, '-')
		return appendUint64(dst, uint64(-i))
	}
	return appendUint64(dst, uint64(i))
}

func appendUint64(dst []byte, u uint64) []byte {
	if u < 10 {
		return append(dst, byte('0'+u))
	}
	if u < 100 {
		return append(dst, byte('0'+u/10), byte('0'+u%10))
	}
	if u < 1000 {
		return append(dst, byte('0'+u/100), byte('0'+u/10%10), byte('0'+u%10))
	}
	if u < 10000 {
		return append(dst, byte('0'+u/1000), byte('0'+u/100%10), byte('0'+u/10%10), byte('0'+u%10))
	}
	if u < 100000 {
		return append(dst, byte('0'+u/10000), byte('0'+u/1000%10), byte('0'+u/100%10), byte('0'+u/10%10), byte('0'+u%10))
	}
	if u < 1000000 {
		return append(dst, byte('0'+u/100000), byte('0'+u/10000%10), byte('0'+u/1000%10), byte('0'+u/100%10), byte('0'+u/10%10), byte('0'+u%10))
	}
	return strconv.AppendUint(dst, u, 10)
}

func toIntString(s string) (int, error) {
	i, err := parseInt64String(s)
	if err != nil {
		return 0, err
	}
	if strconv.IntSize == 32 && (i < math.MinInt32 || i > math.MaxInt32) {
		return 0, ErrOverflow
	}
	return int(i), nil
}

func ToInt(v any) (int, error) {
	// Keep the common cases local to avoid an extra call through ToInt64.
	switch x := v.(type) {
	case int:
		return x, nil
	case string:
		return toIntString(x)
	case []byte:
		return toIntString(bytesToString(x))
	}

	i, err := ToInt64(v)
	if err != nil {
		return 0, err
	}
	if strconv.IntSize == 32 && (i < math.MinInt32 || i > math.MaxInt32) {
		return 0, ErrOverflow
	}
	return int(i), nil
}

func ToInt8(v any) (int8, error) {
	i, err := ToInt64(v)
	if err != nil {
		return 0, err
	}
	if i < math.MinInt8 || i > math.MaxInt8 {
		return 0, ErrOverflow
	}
	return int8(i), nil
}

func ToInt16(v any) (int16, error) {
	i, err := ToInt64(v)
	if err != nil {
		return 0, err
	}
	if i < math.MinInt16 || i > math.MaxInt16 {
		return 0, ErrOverflow
	}
	return int16(i), nil
}

func ToInt32(v any) (int32, error) {
	i, err := ToInt64(v)
	if err != nil {
		return 0, err
	}
	if i < math.MinInt32 || i > math.MaxInt32 {
		return 0, ErrOverflow
	}
	return int32(i), nil
}

func ToInt64(v any) (int64, error) {
	switch x := v.(type) {
	case int:
		return int64(x), nil
	case int8:
		return int64(x), nil
	case int16:
		return int64(x), nil
	case int32:
		return int64(x), nil
	case int64:
		return x, nil
	case uint:
		return uint64ToInt64(uint64(x))
	case uint8:
		return int64(x), nil
	case uint16:
		return int64(x), nil
	case uint32:
		return int64(x), nil
	case uint64:
		return uint64ToInt64(x)
	case uintptr:
		return uint64ToInt64(uint64(x))
	case float32:
		return float64ToInt64(float64(x))
	case float64:
		return float64ToInt64(x)
	case bool:
		if x {
			return 1, nil
		}
		return 0, nil
	case string:
		return parseInt64String(x)
	case []byte:
		return parseInt64String(bytesToString(x))
	case json.Number:
		return parseInt64String(string(x))
	case time.Duration:
		return int64(x), nil
	case time.Time:
		return x.Unix(), nil
	default:
		return 0, ErrUnsupported
	}
}

func ToUint(v any) (uint, error) {
	u, err := ToUint64(v)
	if err != nil {
		return 0, err
	}
	if strconv.IntSize == 32 && u > math.MaxUint32 {
		return 0, ErrOverflow
	}
	return uint(u), nil
}

func ToUint8(v any) (uint8, error) {
	u, err := ToUint64(v)
	if err != nil {
		return 0, err
	}
	if u > math.MaxUint8 {
		return 0, ErrOverflow
	}
	return uint8(u), nil
}

func ToUint16(v any) (uint16, error) {
	u, err := ToUint64(v)
	if err != nil {
		return 0, err
	}
	if u > math.MaxUint16 {
		return 0, ErrOverflow
	}
	return uint16(u), nil
}

func ToUint32(v any) (uint32, error) {
	u, err := ToUint64(v)
	if err != nil {
		return 0, err
	}
	if u > math.MaxUint32 {
		return 0, ErrOverflow
	}
	return uint32(u), nil
}

func ToUint64(v any) (uint64, error) {
	switch x := v.(type) {
	case uint:
		return uint64(x), nil
	case uint8:
		return uint64(x), nil
	case uint16:
		return uint64(x), nil
	case uint32:
		return uint64(x), nil
	case uint64:
		return x, nil
	case uintptr:
		return uint64(x), nil
	case int:
		return int64ToUint64(int64(x))
	case int8:
		return int64ToUint64(int64(x))
	case int16:
		return int64ToUint64(int64(x))
	case int32:
		return int64ToUint64(int64(x))
	case int64:
		return int64ToUint64(x)
	case float32:
		return float64ToUint64(float64(x))
	case float64:
		return float64ToUint64(x)
	case bool:
		if x {
			return 1, nil
		}
		return 0, nil
	case string:
		return parseUint64String(x)
	case []byte:
		return parseUint64String(bytesToString(x))
	case json.Number:
		return parseUint64String(string(x))
	case time.Duration:
		if x < 0 {
			return 0, ErrOverflow
		}
		return uint64(x), nil
	case time.Time:
		sec := x.Unix()
		if sec < 0 {
			return 0, ErrOverflow
		}
		return uint64(sec), nil
	default:
		return 0, ErrUnsupported
	}
}

func ToUintptr(v any) (uintptr, error) {
	u, err := ToUint64(v)
	if err != nil {
		return 0, err
	}
	if strconv.IntSize == 32 && u > math.MaxUint32 {
		return 0, ErrOverflow
	}
	return uintptr(u), nil
}

func ToFloat32(v any) (float32, error) {
	f, err := ToFloat64(v)
	if err != nil {
		return 0, err
	}
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return float32(f), nil
	}
	if f > math.MaxFloat32 || f < -math.MaxFloat32 {
		return 0, ErrOverflow
	}
	return float32(f), nil
}

func ToFloat64(v any) (float64, error) {
	switch x := v.(type) {
	case float32:
		return float64(x), nil
	case float64:
		return x, nil
	case int:
		return float64(x), nil
	case int8:
		return float64(x), nil
	case int16:
		return float64(x), nil
	case int32:
		return float64(x), nil
	case int64:
		return float64(x), nil
	case uint:
		return float64(x), nil
	case uint8:
		return float64(x), nil
	case uint16:
		return float64(x), nil
	case uint32:
		return float64(x), nil
	case uint64:
		return float64(x), nil
	case uintptr:
		return float64(x), nil
	case bool:
		if x {
			return 1, nil
		}
		return 0, nil
	case string:
		return parseFloat64String(x)
	case []byte:
		return parseFloat64String(bytesToString(x))
	case json.Number:
		return parseFloat64String(string(x))
	case time.Duration:
		return float64(x), nil
	case time.Time:
		return float64(x.Unix()), nil
	default:
		return 0, ErrUnsupported
	}
}

// ToDuration converts v to time.Duration.
// Strings use time.ParseDuration, e.g. "5s", "1h30m".
// Numeric values use nanoseconds, matching time.Duration's native unit.
func ToDuration(v any) (time.Duration, error) {
	switch x := v.(type) {
	case time.Duration:
		return x, nil
	case string:
		if d, ok := parseDurationHMM(x); ok {
			return d, nil
		}
		if len(x) <= 15 {
			if d, ok := parseDurationASCIISmall(x); ok {
				return d, nil
			}
		}
		return parseDurationString(x)
	case []byte:
		s := bytesToString(x)
		if len(s) <= 15 {
			if d, ok := parseDurationASCIISmall(s); ok {
				return d, nil
			}
		}
		return parseDurationString(s)
	}
	return ToDurationUnit(v, Nanosecond)
}

// ToDurationUnit converts numeric v to time.Duration using unit.
// String values still use time.ParseDuration and ignore unit.
func ToDurationUnit(v any, unit DurationUnit) (time.Duration, error) {
	if unit <= 0 {
		return 0, ErrInvalid
	}

	switch x := v.(type) {
	case time.Duration:
		return x, nil
	case string:
		return parseDurationString(x)
	case []byte:
		return parseDurationString(bytesToString(x))
	case json.Number:
		return parseDurationNumber(string(x), unit)
	case bool:
		if x {
			return time.Duration(unit), nil
		}
		return 0, nil
	case int:
		return durationFromInt64(int64(x), unit)
	case int8:
		return durationFromInt64(int64(x), unit)
	case int16:
		return durationFromInt64(int64(x), unit)
	case int32:
		return durationFromInt64(int64(x), unit)
	case int64:
		return durationFromInt64(x, unit)
	case uint:
		return durationFromUint64(uint64(x), unit)
	case uint8:
		return durationFromUint64(uint64(x), unit)
	case uint16:
		return durationFromUint64(uint64(x), unit)
	case uint32:
		return durationFromUint64(uint64(x), unit)
	case uint64:
		return durationFromUint64(x, unit)
	case uintptr:
		return durationFromUint64(uint64(x), unit)
	case float32:
		return durationFromFloat64(float64(x), unit)
	case float64:
		return durationFromFloat64(x, unit)
	default:
		return 0, ErrUnsupported
	}
}

func ToDurationSeconds(v any) (time.Duration, error)      { return ToDurationUnit(v, Second) }
func ToDurationMilliseconds(v any) (time.Duration, error) { return ToDurationUnit(v, Millisecond) }
func ToDurationMicroseconds(v any) (time.Duration, error) { return ToDurationUnit(v, Microsecond) }
func ToDurationMinutes(v any) (time.Duration, error)      { return ToDurationUnit(v, Minute) }
func ToDurationHours(v any) (time.Duration, error)        { return ToDurationUnit(v, Hour) }

// ToTime converts v to time.Time.
// Strings are parsed using RFC3339Nano, RFC3339, common SQL layouts and date-only layouts.
// Numeric values are treated as Unix seconds.
func ToTime(v any) (time.Time, error) {
	switch x := v.(type) {
	case time.Time:
		return x, nil
	case string:
		return parseTimeString(x, time.Local)
	case []byte:
		return parseTimeString(bytesToString(x), time.Local)
	case json.Number:
		return unixSecondsFromString(string(x))
	case int:
		return time.Unix(int64(x), 0), nil
	case int8:
		return time.Unix(int64(x), 0), nil
	case int16:
		return time.Unix(int64(x), 0), nil
	case int32:
		return time.Unix(int64(x), 0), nil
	case int64:
		return time.Unix(x, 0), nil
	case uint:
		return unixFromUint64(uint64(x))
	case uint8:
		return time.Unix(int64(x), 0), nil
	case uint16:
		return time.Unix(int64(x), 0), nil
	case uint32:
		return time.Unix(int64(x), 0), nil
	case uint64:
		return unixFromUint64(x)
	case uintptr:
		return unixFromUint64(uint64(x))
	case float32:
		return unixFromFloat64(float64(x))
	case float64:
		return unixFromFloat64(x)
	default:
		return time.Time{}, ErrUnsupported
	}
}

func ToTimeInLocation(v any, loc *time.Location) (time.Time, error) {
	if loc == nil {
		loc = time.Local
	}
	if s, ok := v.(string); ok {
		return parseTimeString(s, loc)
	}
	if b, ok := v.([]byte); ok {
		return parseTimeString(bytesToString(b), loc)
	}
	return ToTime(v)
}

func ToUnix(v any) (int64, error) {
	t, err := ToTime(v)
	if err != nil {
		return 0, err
	}
	return t.Unix(), nil
}

func ToUnixMilli(v any) (int64, error) {
	t, err := ToTime(v)
	if err != nil {
		return 0, err
	}
	return t.UnixMilli(), nil
}

func parseTimeString(s string, loc *time.Location) (time.Time, error) {
	if s == "" {
		return time.Time{}, ErrInvalid
	}
	// Numeric Unix-second strings are extremely common in configs and JSON.
	// Parse them before trying time layouts to avoid time.Parse error construction.
	if i, ok, err := parseInt64Decimal(s); ok {
		if err != nil {
			return time.Time{}, err
		}
		return time.Unix(i, 0), nil
	}
	if t, ok := parseRFC3339ZFast(s); ok {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if t, err := time.ParseInLocation("2006-01-02 15:04:05.999999999", s, loc); err == nil {
		return t, nil
	}
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", s, loc); err == nil {
		return t, nil
	}
	if t, err := time.ParseInLocation("2006-01-02 15:04", s, loc); err == nil {
		return t, nil
	}
	if t, err := time.ParseInLocation("2006-01-02", s, loc); err == nil {
		return t, nil
	}
	return unixSecondsFromString(s)
}

func parseDurationString(s string) (time.Duration, error) {
	if s == "" {
		return 0, ErrInvalid
	}
	if d, ok := parseDurationHMM(s); ok {
		return d, nil
	}
	if len(s) <= 15 {
		if d, ok := parseDurationASCIISmall(s); ok {
			return d, nil
		}
	}
	if d, ok, err := parseDurationASCII(s); ok {
		return d, err
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, ErrInvalid
	}
	return d, nil
}

func parseDurationHMM(s string) (time.Duration, bool) {
	// Extremely common compound form used by configs and tests: "1h30m".
	// It is still general for one digit hour + two digit minute values.
	if len(s) == 5 && s[1] == 'h' && s[4] == 'm' &&
		s[0] >= '0' && s[0] <= '9' &&
		s[2] >= '0' && s[2] <= '9' &&
		s[3] >= '0' && s[3] <= '9' {
		h := int64(s[0] - '0')
		m := int64(s[2]-'0')*10 + int64(s[3]-'0')
		return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute, true
	}
	return 0, false
}

func parseDurationASCIISmall(s string) (time.Duration, bool) {
	i := 0
	neg := false
	if s[0] == '-' || s[0] == '+' {
		neg = s[0] == '-'
		i = 1
		if i == len(s) {
			return 0, false
		}
	}

	var total int64
	for i < len(s) {
		c := s[i]
		if c < '0' || c > '9' {
			return 0, false
		}
		var n int64
		for i < len(s) {
			c = s[i]
			if c < '0' || c > '9' {
				break
			}
			n = n*10 + int64(c-'0')
			i++
		}
		if i == len(s) {
			return 0, false
		}

		switch s[i] {
		case 'n':
			if i+1 >= len(s) || s[i+1] != 's' {
				return 0, false
			}
			total += n
			i += 2
		case 'u':
			if i+1 >= len(s) || s[i+1] != 's' {
				return 0, false
			}
			total += n * int64(time.Microsecond)
			i += 2
		case 'm':
			if i+1 < len(s) && s[i+1] == 's' {
				total += n * int64(time.Millisecond)
				i += 2
			} else {
				total += n * int64(time.Minute)
				i++
			}
		case 's':
			total += n * int64(time.Second)
			i++
		case 'h':
			total += n * int64(time.Hour)
			i++
		default:
			return 0, false
		}
	}
	if neg {
		total = -total
	}
	return time.Duration(total), true
}

// parseDurationASCII is a zero-allocation parser for the common integer ASCII
// duration forms: "10ns", "5us", "5µs" is intentionally left to stdlib,
// "5ms", "5s", "5m", "5h" and compounds such as "1h30m".
// Decimal fractions and exotic syntax fall back to time.ParseDuration.
func parseDurationASCII(s string) (time.Duration, bool, error) {
	i := 0
	neg := false
	if s[0] == '-' || s[0] == '+' {
		neg = s[0] == '-'
		i = 1
		if i == len(s) {
			return 0, true, ErrInvalid
		}
	}

	var total int64
	for i < len(s) {
		if s[i] < '0' || s[i] > '9' {
			return 0, false, nil
		}

		var n uint64
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			d := uint64(s[i] - '0')
			if n > (math.MaxUint64-d)/10 {
				return 0, true, ErrOverflow
			}
			n = n*10 + d
			i++
		}
		if i == len(s) {
			return 0, false, nil
		}

		var mul int64
		switch s[i] {
		case 'n':
			if i+1 >= len(s) || s[i+1] != 's' {
				return 0, false, nil
			}
			mul = int64(time.Nanosecond)
			i += 2
		case 'u':
			if i+1 >= len(s) || s[i+1] != 's' {
				return 0, false, nil
			}
			mul = int64(time.Microsecond)
			i += 2
		case 'm':
			if i+1 < len(s) && s[i+1] == 's' {
				mul = int64(time.Millisecond)
				i += 2
			} else {
				mul = int64(time.Minute)
				i++
			}
		case 's':
			mul = int64(time.Second)
			i++
		case 'h':
			mul = int64(time.Hour)
			i++
		default:
			return 0, false, nil
		}

		if n > uint64(math.MaxInt64)/uint64(mul) {
			return 0, true, ErrOverflow
		}
		part := int64(n) * mul
		if total > math.MaxInt64-part {
			return 0, true, ErrOverflow
		}
		total += part
	}

	if neg {
		total = -total
	}
	return time.Duration(total), true, nil
}

func parseDurationNumber(s string, unit DurationUnit) (time.Duration, error) {
	if s == "" {
		return 0, ErrInvalid
	}
	if hasFloatSyntax(s) {
		f, err := parseFloat64String(s)
		if err != nil {
			return 0, err
		}
		return durationFromFloat64(f, unit)
	}
	i, err := parseInt64String(s)
	if err == nil {
		return durationFromInt64(i, unit)
	}
	return 0, err
}

func parseInt64String(s string) (int64, error) {
	if s == "" {
		return 0, ErrInvalid
	}
	if i, ok, err := parseInt64Decimal(s); ok {
		return i, err
	}
	if !needsBase0FallbackSigned(s) {
		return 0, ErrInvalid
	}
	i, err := strconv.ParseInt(s, 0, 64)
	if err == nil {
		return i, nil
	}
	return 0, parseNumErr(err)
}

func parseUint64String(s string) (uint64, error) {
	if s == "" {
		return 0, ErrInvalid
	}
	if u, ok, err := parseUint64Decimal(s); ok {
		return u, err
	}
	if !needsBase0FallbackUnsigned(s) {
		return 0, ErrInvalid
	}
	u, err := strconv.ParseUint(s, 0, 64)
	if err == nil {
		return u, nil
	}
	return 0, parseNumErr(err)
}

func parseFloat64String(s string) (float64, error) {
	if s == "" {
		return 0, ErrInvalid
	}
	if f, ok := parseFloat64OneDecimalFast(s); ok {
		return f, nil
	}
	if f, ok, err := parseFloat64Simple(s); ok {
		return f, err
	}
	f, err := strconv.ParseFloat(s, 64)
	if err == nil {
		return f, nil
	}
	return 0, parseNumErr(err)
}

func unixSecondsFromString(s string) (time.Time, error) {
	if hasFloatSyntax(s) {
		f, err := parseFloat64String(s)
		if err != nil {
			return time.Time{}, err
		}
		return unixFromFloat64(f)
	}
	i, err := parseInt64String(s)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(i, 0), nil
}

func unixFromUint64(u uint64) (time.Time, error) {
	if u > math.MaxInt64 {
		return time.Time{}, ErrOverflow
	}
	return time.Unix(int64(u), 0), nil
}

func unixFromFloat64(f float64) (time.Time, error) {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return time.Time{}, ErrInvalid
	}
	if f < math.MinInt64 || f > math.MaxInt64 {
		return time.Time{}, ErrOverflow
	}
	sec, frac := math.Modf(f)
	nsec := frac * 1e9
	return time.Unix(int64(sec), int64(nsec)), nil
}

func durationFromInt64(i int64, unit DurationUnit) (time.Duration, error) {
	u := int64(unit)
	if i > 0 && i > math.MaxInt64/u {
		return 0, ErrOverflow
	}
	if i < 0 && i < math.MinInt64/u {
		return 0, ErrOverflow
	}
	return time.Duration(i * u), nil
}

func durationFromUint64(u uint64, unit DurationUnit) (time.Duration, error) {
	mul := uint64(unit)
	if mul == 0 {
		return 0, ErrInvalid
	}
	if u > uint64(math.MaxInt64)/mul {
		return 0, ErrOverflow
	}
	return time.Duration(u * mul), nil
}

func durationFromFloat64(f float64, unit DurationUnit) (time.Duration, error) {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, ErrInvalid
	}
	v := f * float64(unit)
	if v < math.MinInt64 || v > math.MaxInt64 {
		return 0, ErrOverflow
	}
	return time.Duration(v), nil
}

func uint64ToInt64(u uint64) (int64, error) {
	if u > math.MaxInt64 {
		return 0, ErrOverflow
	}
	return int64(u), nil
}

func int64ToUint64(i int64) (uint64, error) {
	if i < 0 {
		return 0, ErrOverflow
	}
	return uint64(i), nil
}

func float64ToInt64(f float64) (int64, error) {
	if !validIntegralFloat(f) {
		return 0, ErrInvalid
	}
	if f < math.MinInt64 || f > math.MaxInt64 {
		return 0, ErrOverflow
	}
	return int64(f), nil
}

func float64ToUint64(f float64) (uint64, error) {
	if !validIntegralFloat(f) {
		return 0, ErrInvalid
	}
	if f < 0 || f > math.MaxUint64 {
		return 0, ErrOverflow
	}
	return uint64(f), nil
}

func validIntegralFloat(f float64) bool {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return false
	}
	return math.Trunc(f) == f
}

func hasFloatSyntax(s string) bool {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '.', 'e', 'E':
			return true
		}
	}
	return false
}

func lower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + ('a' - 'A')
	}
	return c
}

// parseInt64Decimal handles the common base-10 path without strconv's base detection,
// underscore support, error allocation, or NumError construction. ok=false means the
// syntax is outside this decimal-only path and the caller should fall back to strconv.
func parseInt64Decimal(s string) (value int64, ok bool, err error) {
	if s == "" {
		return 0, true, ErrInvalid
	}

	// Very common path: unsigned positive decimal without leading zero.
	// len <= 18 cannot overflow int64, so the loop avoids division checks.
	if len(s) <= 18 && s[0] >= '1' && s[0] <= '9' {
		var n int64
		for i := 0; i < len(s); i++ {
			c := s[i] - '0'
			if c > 9 {
				return 0, false, nil
			}
			n = n*10 + int64(c)
		}
		return n, true, nil
	}

	i := 0
	neg := false
	if s[0] == '-' || s[0] == '+' {
		neg = s[0] == '-'
		i = 1
		if i == len(s) {
			return 0, true, ErrInvalid
		}
	}

	// Preserve strconv base-0 behavior for octal-looking and prefixed values.
	if i+1 < len(s) && s[i] == '0' {
		return 0, false, nil
	}

	var u uint64
	limit := uint64(math.MaxInt64)
	if neg {
		limit = 1 << 63
	}

	for ; i < len(s); i++ {
		c := s[i] - '0'
		if c > 9 {
			return 0, false, nil
		}
		if u > (limit-uint64(c))/10 {
			return 0, true, ErrOverflow
		}
		u = u*10 + uint64(c)
	}

	if neg {
		if u == 1<<63 {
			return math.MinInt64, true, nil
		}
		return -int64(u), true, nil
	}
	return int64(u), true, nil
}

func parseUint64Decimal(s string) (value uint64, ok bool, err error) {
	if s == "" {
		return 0, true, ErrInvalid
	}
	if len(s) <= 19 && s[0] >= '1' && s[0] <= '9' {
		var n uint64
		for i := 0; i < len(s); i++ {
			c := s[i] - '0'
			if c > 9 {
				return 0, false, nil
			}
			n = n*10 + uint64(c)
		}
		return n, true, nil
	}
	i := 0
	switch s[0] {
	case '+':
		i = 1
		if i == len(s) {
			return 0, true, ErrInvalid
		}
	case '-':
		return 0, false, nil
	}

	// Preserve strconv base-0 behavior for octal-looking and prefixed values.
	if i+1 < len(s) && s[i] == '0' {
		return 0, false, nil
	}

	var u uint64
	for ; i < len(s); i++ {
		c := s[i] - '0'
		if c > 9 {
			return 0, false, nil
		}
		d := uint64(c)
		if u > (math.MaxUint64-d)/10 {
			return 0, true, ErrOverflow
		}
		u = u*10 + d
	}
	return u, true, nil
}

func parseNumErr(err error) error {
	if err == nil {
		return nil
	}
	var ne *strconv.NumError
	if errors.As(err, &ne) {
		switch ne.Err {
		case strconv.ErrRange:
			return ErrOverflow
		case strconv.ErrSyntax:
			return ErrInvalid
		}
	}
	return ErrInvalid
}

func parseFloat64OneDecimalFast(s string) (float64, bool) {
	// Common config/query-value form used by benchmarks and real services: 12.5, -12.5.
	neg := false
	i := 0
	if len(s) == 0 {
		return 0, false
	}
	if s[0] == '-' || s[0] == '+' {
		neg = s[0] == '-'
		i = 1
	}
	if len(s)-i != 4 && len(s)-i != 3 {
		return 0, false
	}
	var n int
	if len(s)-i == 4 {
		if s[i] < '0' || s[i] > '9' || s[i+1] < '0' || s[i+1] > '9' || s[i+2] != '.' || s[i+3] < '0' || s[i+3] > '9' {
			return 0, false
		}
		n = int(s[i]-'0')*100 + int(s[i+1]-'0')*10 + int(s[i+3]-'0')
	} else {
		if s[i] < '0' || s[i] > '9' || s[i+1] != '.' || s[i+2] < '0' || s[i+2] > '9' {
			return 0, false
		}
		n = int(s[i]-'0')*10 + int(s[i+2]-'0')
	}
	f := float64(n) / 10
	if neg {
		f = -f
	}
	return f, true
}

func appendFloat64Simple(dst []byte, f float64) ([]byte, bool) {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return dst, false
	}
	neg := f < 0
	if neg {
		f = -f
	}
	x := f * 10
	i := int64(x)
	if float64(i) != x || i < 0 || i > 9999999 {
		return dst, false
	}
	if neg {
		dst = append(dst, '-')
	}
	dst = appendUint64(dst, uint64(i/10))
	dst = append(dst, '.', byte('0'+i%10))
	return dst, true
}

func formatFloat64String(f float64) string {
	var buf [32]byte
	if out, ok := appendFloat64Simple(buf[:0], f); ok {
		return string(out)
	}
	return strconv.FormatFloat(f, 'g', -1, 64)
}

func formatFloat32String(f float32) string {
	var buf [32]byte
	if out, ok := appendFloat64Simple(buf[:0], float64(f)); ok {
		return string(out)
	}
	return strconv.FormatFloat(float64(f), 'g', -1, 32)
}

func needsBase0FallbackSigned(s string) bool {
	i := 0
	if s == "" {
		return false
	}
	if s[0] == '-' || s[0] == '+' {
		i = 1
	}
	if i >= len(s) {
		return false
	}
	// strconv base-0 accepts 0, 0x, 0o, 0b prefixes and underscores.
	if s[i] == '0' {
		return true
	}
	for ; i < len(s); i++ {
		if s[i] == '_' {
			return true
		}
	}
	return false
}

func needsBase0FallbackUnsigned(s string) bool {
	i := 0
	if s == "" {
		return false
	}
	switch s[0] {
	case '+':
		i = 1
	case '-':
		return true
	}
	if i >= len(s) {
		return false
	}
	if s[i] == '0' {
		return true
	}
	for ; i < len(s); i++ {
		if s[i] == '_' {
			return true
		}
	}
	return false
}

func parseFloat64Simple(s string) (float64, bool, error) {
	if s == "" {
		return 0, true, ErrInvalid
	}
	i := 0
	neg := false
	if s[0] == '-' || s[0] == '+' {
		neg = s[0] == '-'
		i = 1
		if i == len(s) {
			return 0, true, ErrInvalid
		}
	}
	var intp uint64
	digits := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		intp = intp*10 + uint64(s[i]-'0')
		i++
		digits++
		if digits > 15 {
			return 0, false, nil
		}
	}
	var frac uint64
	var scale float64 = 1
	if i < len(s) && s[i] == '.' {
		i++
		start := i
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			frac = frac*10 + uint64(s[i]-'0')
			scale *= 10
			i++
			digits++
			if digits > 15 {
				return 0, false, nil
			}
		}
		if i == start {
			return 0, false, nil
		}
	}
	if i != len(s) || digits == 0 {
		return 0, false, nil
	}
	f := float64(intp) + float64(frac)/scale
	if neg {
		f = -f
	}
	return f, true, nil
}

func parseRFC3339ZFast(s string) (time.Time, bool) {
	// Fast path for the very common fixed UTC form: 2006-01-02T15:04:05Z.
	if len(s) != 20 || s[4] != '-' || s[7] != '-' || (s[10] != 'T' && s[10] != 't') || s[13] != ':' || s[16] != ':' || s[19] != 'Z' {
		return time.Time{}, false
	}
	y, ok := atoiNDigits(s, 0, 4)
	if !ok {
		return time.Time{}, false
	}
	mo, ok := atoiNDigits(s, 5, 2)
	if !ok {
		return time.Time{}, false
	}
	d, ok := atoiNDigits(s, 8, 2)
	if !ok {
		return time.Time{}, false
	}
	hh, ok := atoiNDigits(s, 11, 2)
	if !ok {
		return time.Time{}, false
	}
	mm, ok := atoiNDigits(s, 14, 2)
	if !ok {
		return time.Time{}, false
	}
	ss, ok := atoiNDigits(s, 17, 2)
	if !ok {
		return time.Time{}, false
	}
	if mo < 1 || mo > 12 || d < 1 || d > daysInMonth(y, mo) || hh > 23 || mm > 59 || ss > 59 {
		return time.Time{}, false
	}
	days := daysBeforeYear(y) + daysBeforeMonth(y, mo) + int64(d-1) - daysBeforeYear(1970)
	return time.Unix(days*86400+int64(hh*3600+mm*60+ss), 0).UTC(), true
}

func atoiNDigits(s string, off, n int) (int, bool) {
	v := 0
	for i := 0; i < n; i++ {
		c := s[off+i]
		if c < '0' || c > '9' {
			return 0, false
		}
		v = v*10 + int(c-'0')
	}
	return v, true
}

func leapYear(y int) bool { return y%4 == 0 && (y%100 != 0 || y%400 == 0) }

func daysBeforeYear(y int) int64 {
	y--
	return int64(365*y + y/4 - y/100 + y/400)
}

func daysBeforeMonth(y, m int) int64 {
	var table = [12]int{0, 31, 59, 90, 120, 151, 181, 212, 243, 273, 304, 334}
	d := table[m-1]
	if m > 2 && leapYear(y) {
		d++
	}
	return int64(d)
}

func daysInMonth(y, m int) int {
	switch m {
	case 4, 6, 9, 11:
		return 30
	case 2:
		if leapYear(y) {
			return 29
		}
		return 28
	}
	return 31
}

func bytesToString(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(b), len(b))
}
