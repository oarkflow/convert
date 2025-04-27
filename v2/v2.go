package v2

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/oarkflow/date"
)

var (
	reDate             = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+\-]\d{2}:\d{2})?)|(\d{2} \w{3} \d{4} \d{2}:\d{2}:\d{2} [A-Z]{3})|(\w{3} \d{1,2},? \d{4} \d{2}:\d{2}(:\d{2})? [AP]M)|(\d{4}-\d{2}-\d{2})$`)
	customConverters   = map[reflect.Type]func(any) (any, error){}
	ErrUnsupportedType = errors.New("unsupported type")
)

func RegisterConverter[T any](fn func(any) (T, error)) {
	var t T
	customConverters[reflect.TypeOf(t)] = func(v any) (any, error) {
		return fn(v)
	}
}

// To preserves original signature but now returns error
func To[T any](src T, dst any) (T, error) {
	var zero T
	// custom hook
	if fn, ok := customConverters[reflect.TypeOf(zero)]; ok {
		out, err := fn(dst)
		if err != nil {
			return zero, err
		}
		return out.(T), nil
	}
	switch any(src).(type) {
	case string:
		s, err := ToString(dst)
		return any(s).(T), err
	case bool:
		b, err := ToBool(dst)
		return any(b).(T), err
	case time.Time:
		tm, err := ToTime(dst)
		return any(tm).(T), err
	case float32:
		f, err := ToFloat32(dst)
		return any(f).(T), err
	case float64:
		f, err := ToFloat64(dst)
		return any(f).(T), err
	case uint:
		u, err := ToUint(dst)
		return any(u).(T), err
	case uint8:
		u, err := ToUint8(dst)
		return any(u).(T), err
	case uint16:
		u, err := ToUint16(dst)
		return any(u).(T), err
	case uint32:
		u, err := ToUint32(dst)
		return any(u).(T), err
	case uint64:
		u, err := ToUint64(dst)
		return any(u).(T), err
	case int:
		i, err := ToInt(dst)
		return any(i).(T), err
	case int8:
		i, err := ToInt8(dst)
		return any(i).(T), err
	case int16:
		i, err := ToInt16(dst)
		return any(i).(T), err
	case int32:
		i, err := ToInt32(dst)
		return any(i).(T), err
	case int64:
		i, err := ToInt64(dst)
		return any(i).(T), err
	case []string:
		sl, err := ToSlice[string](dst)
		return any(sl).(T), err
	case []bool:
		sl, err := ToSlice[bool](dst)
		return any(sl).(T), err
	case []time.Time:
		sl, err := ToSlice[time.Time](dst)
		return any(sl).(T), err
	case []float32:
		sl, err := ToSlice[float32](dst)
		return any(sl).(T), err
	case []float64:
		sl, err := ToSlice[float64](dst)
		return any(sl).(T), err
	case []uint:
		sl, err := ToSlice[uint](dst)
		return any(sl).(T), err
	case []uint8:
		sl, err := ToSlice[uint8](dst)
		return any(sl).(T), err
	case []uint16:
		sl, err := ToSlice[uint16](dst)
		return any(sl).(T), err
	case []uint32:
		sl, err := ToSlice[uint32](dst)
		return any(sl).(T), err
	case []uint64:
		sl, err := ToSlice[uint64](dst)
		return any(sl).(T), err
	case []int:
		sl, err := ToSlice[int](dst)
		return any(sl).(T), err
	case []int8:
		sl, err := ToSlice[int8](dst)
		return any(sl).(T), err
	case []int16:
		sl, err := ToSlice[int16](dst)
		return any(sl).(T), err
	case []int32:
		sl, err := ToSlice[int32](dst)
		return any(sl).(T), err
	case []int64:
		sl, err := ToSlice[int64](dst)
		return any(sl).(T), err
	case json.Number:
		var result json.Number
		switch dst.(type) {
		case string:
			result = json.Number(dst.(string))
		case int, int8, int16, int32, int64:
			i, err := ToInt64(dst)
			if err != nil {
				return zero, err
			}
			result = json.Number(strconv.FormatInt(i, 10))
		case uint, uint8, uint16, uint32, uint64:
			u, err := ToUint64(dst)
			if err != nil {
				return zero, err
			}
			result = json.Number(strconv.FormatUint(u, 10))
		case float32, float64:
			f, err := ToFloat64(dst)
			if err != nil {
				return zero, err
			}
			result = json.Number(strconv.FormatFloat(f, 'f', -1, 64))
		default:
			s, err := ToString(dst)
			if err != nil {
				return zero, err
			}
			result = json.Number(s)
		}
		return any(result).(T), nil
	default:
		return zero, ErrUnsupportedType
	}
}

func ToString(val any) (string, error) {
	switch v := val.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	case fmt.Stringer:
		return v.String(), nil
	case json.Number:
		return v.String(), nil
	default:
		return fmt.Sprintf("%v", val), nil
	}
}

func ToBool(val any) (bool, error) {
	switch v := val.(type) {
	case bool:
		return v, nil
	case string:
		b, err := strconv.ParseBool(strings.ToLower(v))
		return b, err
	default:
		return false, fmt.Errorf("cannot convert %T to bool", val)
	}
}

func ToTime(val any) (time.Time, error) {
	switch v := val.(type) {
	case time.Time:
		return v, nil
	case string:
		if !reDate.MatchString(v) {
			return time.Time{}, fmt.Errorf("invalid datetime: %s", v)
		}
		if t, err := date.Parse(v); err == nil {
			return t, nil
		}
		return date.Parse(v)
	default:
		return time.Time{}, fmt.Errorf("cannot convert %T to time", val)
	}
}

func ToFloat32(val any) (float32, error) {
	f, err := ToFloat64(val)
	if err != nil {
		return 0, err
	}
	if f > math.MaxFloat32 || f < -math.MaxFloat32 {
		return 0, errors.New("float32 overflow")
	}
	return float32(f), nil
}

func ToFloat64(val any) (float64, error) {
	switch v := val.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int, int8, int16, int32, int64:
		return float64(reflect.ValueOf(v).Int()), nil
	case uint, uint8, uint16, uint32, uint64:
		return float64(reflect.ValueOf(v).Uint()), nil
	case string:
		return strconv.ParseFloat(v, 64)
	case json.Number:
		return v.Float64()
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", val)
	}
}

func ToUint(val any) (uint, error) {
	u, err := ToUint64(val)
	if err != nil {
		return 0, err
	}
	if u > math.MaxUint {
		return 0, errors.New("uint overflow")
	}
	return uint(u), nil
}

func ToUint8(val any) (uint8, error) {
	u, err := ToUint64(val)
	if err != nil {
		return 0, err
	}
	if u > math.MaxUint8 {
		return 0, errors.New("uint8 overflow")
	}
	return uint8(u), nil
}

func ToUint16(val any) (uint16, error) {
	u, err := ToUint64(val)
	if err != nil {
		return 0, err
	}
	if u > math.MaxUint16 {
		return 0, errors.New("uint16 overflow")
	}
	return uint16(u), nil
}

func ToUint32(val any) (uint32, error) {
	u, err := ToUint64(val)
	if err != nil {
		return 0, err
	}
	if u > math.MaxUint32 {
		return 0, errors.New("uint32 overflow")
	}
	return uint32(u), nil
}

func ToUint64(val any) (uint64, error) {
	switch v := val.(type) {
	case uint64:
		return v, nil
	case uint, uint32, uint16, uint8:
		return reflect.ValueOf(v).Uint(), nil
	case int, int8, int16, int32, int64:
		i := reflect.ValueOf(v).Int()
		if i < 0 {
			return 0, errors.New("negative to unsigned")
		}
		return uint64(i), nil
	case float32, float64:
		f := reflect.ValueOf(v).Float()
		if f < 0 {
			return 0, errors.New("negative to unsigned")
		}
		return uint64(f), nil
	case string:
		return strconv.ParseUint(v, 10, 64)
	case json.Number:
		if i, err := v.Int64(); err == nil && i >= 0 {
			return uint64(i), nil
		} else {
			return 0, err
		}
	default:
		return 0, fmt.Errorf("cannot convert %T to uint64", val)
	}
}

func ToInt(val any) (int, error) {
	i, err := ToInt64(val)
	if err != nil {
		return 0, err
	}
	if i < math.MinInt || i > math.MaxInt {
		return 0, errors.New("int overflow")
	}
	return int(i), nil
}

func ToInt8(val any) (int8, error) {
	i, err := ToInt64(val)
	if err != nil {
		return 0, err
	}
	if i < math.MinInt8 || i > math.MaxInt8 {
		return 0, errors.New("int8 overflow")
	}
	return int8(i), nil
}

func ToInt16(val any) (int16, error) {
	i, err := ToInt64(val)
	if err != nil {
		return 0, err
	}
	if i < math.MinInt16 || i > math.MaxInt16 {
		return 0, errors.New("int16 overflow")
	}
	return int16(i), nil
}

func ToInt32(val any) (int32, error) {
	i, err := ToInt64(val)
	if err != nil {
		return 0, err
	}
	if i < math.MinInt32 || i > math.MaxInt32 {
		return 0, errors.New("int32 overflow")
	}
	return int32(i), nil
}

func ToInt64(val any) (int64, error) {
	switch v := val.(type) {
	case int64:
		return v, nil
	case int, int32, int16, int8:
		return reflect.ValueOf(v).Int(), nil
	case uint, uint8, uint16, uint32, uint64:
		return int64(reflect.ValueOf(v).Uint()), nil
	case float32, float64:
		return int64(reflect.ValueOf(v).Float()), nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	case json.Number:
		return v.Int64()
	case bool:
		if v {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("cannot convert %T to int64", val)
	}
}

func ToAny(val any) (any, error) {
	return val, nil
}

type Callback[T any] func(val any) (T, error)

func convertSlice[T any, U any](slice []T) ([]U, error) {
	var t U
	result := make([]U, len(slice))
	for i, elem := range slice {
		if any(t) == nil {
			result[i] = any(elem).(U)
		} else {
			str, ok := To(t, elem)
			if ok != nil {
				return nil, ok
			}
			result[i] = str
		}
	}
	return result, nil
}

func ToSlice[U any](val any) ([]U, error) {
	switch v := val.(type) {
	case []U:
		return v, nil
	case []any:
		return convertSlice[any, U](v)
	case []string:
		return convertSlice[string, U](v)
	case []int:
		return convertSlice[int, U](v)
	case []int8:
		return convertSlice[int8, U](v)
	case []int16:
		return convertSlice[int16, U](v)
	case []int32:
		return convertSlice[int32, U](v)
	case []int64:
		return convertSlice[int64, U](v)
	case []uint:
		return convertSlice[uint, U](v)
	case []uint8:
		return convertSlice[uint8, U](v)
	case []uint16:
		return convertSlice[uint16, U](v)
	case []uint32:
		return convertSlice[uint32, U](v)
	case []uint64:
		return convertSlice[uint64, U](v)
	case []float32:
		return convertSlice[float32, U](v)
	case []float64:
		return convertSlice[float64, U](v)
	default:
		return nil, fmt.Errorf("cannot convert %T to slice[%T]", val, *new(U))
	}
}

func Compare[T any](a T, b any) (int, error) {
	av := any(a)
	// try date first
	if s, ok := av.(string); ok && IsValidDateTime(s) {
		at, err := date.Parse(s)
		if err != nil {
			return 0, err
		}
		bt, err := ToTime(b)
		if err != nil {
			return 0, err
		}
		switch {
		case at.Before(bt):
			return -1, nil
		case at.After(bt):
			return 1, nil
		default:
			return 0, nil
		}
	}
	// fallback to To()
	dst, err := To(a, b)
	if err != nil {
		return 0, err
	}
	switch x := av.(type) {
	case string:
		y := any(dst).(string)
		return strings.Compare(x, y), nil
	case bool:
		xb, _ := ToBool(x)
		yb, _ := ToBool(dst)
		if xb == yb {
			return 0, nil
		}
		if !xb && yb {
			return -1, nil
		}
		return 1, nil
	case time.Time:
		y := any(dst).(time.Time)
		switch {
		case x.Before(y):
			return -1, nil
		case x.After(y):
			return 1, nil
		default:
			return 0, nil
		}
	case float32:
		y := any(dst).(float32)
		if x < y {
			return -1, nil
		}
		if x > y {
			return 1, nil
		}
		return 0, nil
	case float64:
		y := any(dst).(float64)
		if x < y {
			return -1, nil
		}
		if x > y {
			return 1, nil
		}
		return 0, nil
	case uint:
		y := any(dst).(uint)
		if x < y {
			return -1, nil
		}
		if x > y {
			return 1, nil
		}
		return 0, nil
	case uint8:
		y := any(dst).(uint8)
		if x < y {
			return -1, nil
		}
		if x > y {
			return 1, nil
		}
		return 0, nil
	case uint16:
		y := any(dst).(uint16)
		if x < y {
			return -1, nil
		}
		if x > y {
			return 1, nil
		}
		return 0, nil
	case uint32:
		y := any(dst).(uint32)
		if x < y {
			return -1, nil
		}
		if x > y {
			return 1, nil
		}
		return 0, nil
	case uint64:
		y := any(dst).(uint64)
		if x < y {
			return -1, nil
		}
		if x > y {
			return 1, nil
		}
		return 0, nil
	case int:
		y, _ := any(dst).(int)
		if x < y {
			return -1, nil
		}
		if x > y {
			return 1, nil
		}
		return 0, nil
	case int8:
		y := any(dst).(int8)
		if x < y {
			return -1, nil
		}
		if x > y {
			return 1, nil
		}
		return 0, nil
	case int16:
		y := any(dst).(int16)
		if x < y {
			return -1, nil
		}
		if x > y {
			return 1, nil
		}
		return 0, nil
	case int32:
		y := any(dst).(int32)
		if x < y {
			return -1, nil
		}
		if x > y {
			return 1, nil
		}
		return 0, nil
	case int64:
		y := any(dst).(int64)
		if x < y {
			return -1, nil
		}
		if x > y {
			return 1, nil
		}
		return 0, nil
	case json.Number:
		xFloat, err := ToFloat64(x)
		if err != nil {
			return 0, err
		}
		yFloat, err := ToFloat64(dst)
		if err != nil {
			return 0, err
		}
		if xFloat < yFloat {
			return -1, nil
		}
		if xFloat > yFloat {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, ErrUnsupportedType
	}
}

func IsValidDateTime(str string) bool {
	return reDate.MatchString(str)
}
