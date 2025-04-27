package v2

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/oarkflow/date"
)

func To[T any](src T, dst any) (T, bool) {
	switch any(src).(type) {
	case string:
		val, ok := ToString(dst)
		if !ok {
			return *new(T), false
		}
		return any(val).(T), true
	case bool:
		val, ok := ToBool(dst)
		if !ok {
			return *new(T), false
		}
		return any(val).(T), true
	case time.Time:
		val, ok := ToTime(dst)
		if !ok {
			return *new(T), false
		}
		return any(val).(T), true
	case float32:
		val, ok := ToFloat32(dst)
		if !ok {
			return *new(T), false
		}
		return any(val).(T), true
	case float64:
		val, ok := ToFloat64(dst)
		if !ok {
			return *new(T), false
		}
		return any(val).(T), true
	case uint:
		val, ok := ToUint(dst)
		if !ok {
			return *new(T), false
		}
		return any(val).(T), true
	case uint8:
		val, ok := ToUint8(dst)
		if !ok {
			return *new(T), false
		}
		return any(val).(T), true
	case uint16:
		val, ok := ToUint16(dst)
		if !ok {
			return *new(T), false
		}
		return any(val).(T), true
	case uint32:
		val, ok := ToUint32(dst)
		if !ok {
			return *new(T), false
		}
		return any(val).(T), true
	case uint64:
		val, ok := ToUint64(dst)
		if !ok {
			return *new(T), false
		}
		return any(val).(T), true
	case int:
		val, ok := ToInt(dst)
		if !ok {
			return *new(T), false
		}
		return any(val).(T), true
	case int8:
		val, ok := ToInt8(dst)
		if !ok {
			return *new(T), false
		}
		return any(val).(T), true
	case int16:
		val, ok := ToInt16(dst)
		if !ok {
			return *new(T), false
		}
		return any(val).(T), true
	case int32:
		val, ok := ToInt32(dst)
		if !ok {
			return *new(T), false
		}
		return any(val).(T), true
	case int64:
		val, ok := ToInt64(dst)
		if !ok {
			return *new(T), false
		}
		return any(val).(T), true
	case []string:
		val, ok := ToSlice[string](dst)
		if !ok {
			return *new(T), false
		}
		return any(val).(T), true
	case []bool:
		val, ok := ToSlice[bool](dst)
		if !ok {
			return *new(T), false
		}
		return any(val).(T), true
	case []time.Time:
		val, ok := ToSlice[time.Time](dst)
		if !ok {
			return *new(T), false
		}
		return any(val).(T), true
	case []float32:
		val, ok := ToSlice[float32](dst)
		if !ok {
			return *new(T), false
		}
		return any(val).(T), true
	case []float64:
		val, ok := ToSlice[float64](dst)
		if !ok {
			return *new(T), false
		}
		return any(val).(T), true
	case []uint:
		val, ok := ToSlice[uint](dst)
		if !ok {
			return *new(T), false
		}
		return any(val).(T), true
	case []uint8:
		val, ok := ToSlice[uint8](dst)
		if !ok {
			return *new(T), false
		}
		return any(val).(T), true
	case []uint16:
		val, ok := ToSlice[uint16](dst)
		if !ok {
			return *new(T), false
		}
		return any(val).(T), true
	case []uint32:
		val, ok := ToSlice[uint32](dst)
		if !ok {
			return *new(T), false
		}
		return any(val).(T), true
	case []uint64:
		val, ok := ToSlice[uint64](dst)
		if !ok {
			return *new(T), false
		}
		return any(val).(T), true
	case []int:
		val, ok := ToSlice[int](dst)
		if !ok {
			return *new(T), false
		}
		return any(val).(T), true
	case []int8:
		val, ok := ToSlice[int8](dst)
		if !ok {
			return *new(T), false
		}
		return any(val).(T), true
	case []int16:
		val, ok := ToSlice[int16](dst)
		if !ok {
			return *new(T), false
		}
		return any(val).(T), true
	case []int32:
		val, ok := ToSlice[int32](dst)
		if !ok {
			return *new(T), false
		}
		return any(val).(T), true
	case []int64:
		val, ok := ToSlice[int64](dst)
		if !ok {
			return *new(T), false
		}
		return any(val).(T), true
	default:
		return *new(T), false
	}
}

// ToString - Basic type conversion functions
func ToString(val any) (string, bool) {
	switch v := val.(type) {
	case string:
		return v, true
	case []byte:
		return string(v), true
	case fmt.Stringer:
		return v.String(), true
	case json.Number:
		return v.String(), true
	default:
		return fmt.Sprintf("%v", val), true
	}
}

func ToBool(val any) (bool, bool) {
	switch v := val.(type) {
	case bool:
		return v, true
	case string:
		b, err := strconv.ParseBool(v)
		return b, err == nil
	default:
		return false, false
	}
}

func ToTime(val any) (time.Time, bool) {
	switch v := val.(type) {
	case time.Time:
		return v, true
	case string:
		t, err := date.Parse(v)
		return t, err == nil
	default:
		return time.Time{}, false
	}
}

func ToFloat32(val any) (float32, bool) {
	switch v := val.(type) {
	case float32:
		return v, true
	case float64:
		return float32(v), true
	case string:
		f, err := strconv.ParseFloat(v, 32)
		return float32(f), err == nil
	case int:
		return float32(v), true
	case int64:
		return float32(v), true
	case int8:
		return float32(v), true
	case int16:
		return float32(v), true
	case uint:
		return float32(v), true
	case uint64:
		return float32(v), true
	case json.Number:
		f, err := v.Float64()
		return float32(f), err == nil
	default:
		return 0, false
	}
}

func ToFloat64(val any) (float64, bool) {
	switch v := val.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case string:
		f, err := strconv.ParseFloat(v, 64)
		return f, err == nil
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint64:
		return float64(v), true
	case json.Number:
		f, err := v.Float64()
		return float64(f), err == nil
	default:
		return 0, false
	}
}

func ToUint(val any) (uint, bool) {
	switch v := val.(type) {
	case uint:
		return v, true
	case uint64:
		return uint(v), true
	case int:
		return uint(v), v >= 0
	case int64:
		return uint(v), v >= 0
	case string:
		u, err := strconv.ParseUint(v, 10, 64)
		return uint(u), err == nil
	case json.Number:
		i, err := v.Int64()
		if err != nil || i < 0 {
			return 0, false
		}
		return uint(i), true
	default:
		return 0, false
	}
}

func ToUint8(val any) (uint8, bool) {
	switch v := val.(type) {
	case uint8:
		return v, true
	case uint:
		return uint8(v), v <= 255
	case int:
		return uint8(v), v >= 0 && v <= 255
	case float32:
		return uint8(v), true
	case float64:
		return uint8(v), true
	case string:
		u, err := strconv.ParseUint(v, 10, 8)
		return uint8(u), err == nil
	case json.Number:
		i, err := v.Int64()
		if err != nil || i < 0 || i > 255 {
			return 0, false
		}
		return uint8(i), true
	default:
		return 0, false
	}
}

func ToUint16(val any) (uint16, bool) {
	switch v := val.(type) {
	case uint16:
		return v, true
	case uint:
		return uint16(v), v <= 65535
	case int:
		return uint16(v), v >= 0 && v <= 65535
	case float32:
		return uint16(v), true
	case float64:
		return uint16(v), true
	case string:
		u, err := strconv.ParseUint(v, 10, 16)
		return uint16(u), err == nil
	case json.Number:
		i, err := v.Int64()
		if err != nil || i < 0 || i > 65535 {
			return 0, false
		}
		return uint16(i), true
	default:
		return 0, false
	}
}

func ToUint32(val any) (uint32, bool) {
	maxUint32Bit := uint32((1 << 32) - 1)
	switch v := val.(type) {
	case uint32:
		return v, true
	case uint:
		vt := uint32(v)
		return vt, vt <= maxUint32Bit
	case int:
		vt := uint32(v)
		return vt, v >= 0 && vt <= maxUint32Bit
	case string:
		u, err := strconv.ParseUint(v, 10, 32)
		return uint32(u), err == nil
	case json.Number:
		i, err := v.Int64()
		maxUint32Bit := int64((1 << 32) - 1)
		if err != nil || i < 0 || i > maxUint32Bit {
			return 0, false
		}
		return uint32(i), true
	default:
		return 0, false
	}
}

func ToUint64(val any) (uint64, bool) {
	switch v := val.(type) {
	case uint64:
		return v, true
	case uint:
		return uint64(v), true
	case int:
		return uint64(v), v >= 0
	case int64:
		return uint64(v), v >= 0
	case float32:
		return uint64(v), true
	case float64:
		return uint64(v), true
	case string:
		u, err := strconv.ParseUint(v, 10, 64)
		return u, err == nil
	case json.Number:
		i, err := v.Int64()
		if err != nil || i < 0 {
			return 0, false
		}
		return uint64(i), true
	default:
		return 0, false
	}
}

func ToInt(val any) (int, bool) {
	switch v := val.(type) {
	case int:
		return v, true
	case int64:
		return int(v), v <= int64(^uint(0)>>1) && v >= -int64(^uint(0)>>1)-1
	case uint:
		return int(v), v <= ^uint(0)>>1
	case float32:
		return int(v), true
	case float64:
		return int(v), true
	case string:
		i, err := strconv.Atoi(v)
		return i, err == nil
	case json.Number:
		i, err := v.Int64()
		return int(i), err == nil
	default:
		return 0, false
	}
}

func ToInt8(val any) (int8, bool) {
	switch v := val.(type) {
	case int8:
		return v, true
	case int:
		return int8(v), v >= -128 && v <= 127
	case float32:
		return int8(v), true
	case float64:
		return int8(v), true
	case string:
		i, err := strconv.ParseInt(v, 10, 8)
		return int8(i), err == nil
	case json.Number:
		i, err := v.Int64()
		if err != nil || i < -128 || i > 127 {
			return 0, false
		}
		return int8(i), true
	default:
		return 0, false
	}
}

func ToInt16(val any) (int16, bool) {
	switch v := val.(type) {
	case int16:
		return v, true
	case int:
		return int16(v), v >= -32768 && v <= 32767
	case float32:
		return int16(v), true
	case float64:
		return int16(v), true
	case string:
		i, err := strconv.ParseInt(v, 10, 16)
		return int16(i), err == nil
	case json.Number:
		i, err := v.Int64()
		if err != nil || i < -32768 || i > 32767 {
			return 0, false
		}
		return int16(i), true
	default:
		return 0, false
	}
}

func ToInt32(val any) (int32, bool) {
	maxInt32Bit := int32((1 << 31) - 1)
	switch v := val.(type) {
	case int32:
		return v, true
	case int:
		vt := int32(v)
		return vt, vt >= -maxInt32Bit && vt <= maxInt32Bit
	case string:
		i, err := strconv.ParseInt(v, 10, 32)
		return int32(i), err == nil
	case json.Number:
		i, err := v.Int64()
		if err != nil {
			return 0, false
		}
		return int32(i), true
	default:
		return 0, false
	}
}

func ToInt64(val any) (int64, bool) {
	switch v := val.(type) {
	case int64:
		return v, true
	case int:
		return int64(v), true
	case float32:
		return int64(v), true
	case float64:
		return int64(v), true
	case string:
		i, err := strconv.ParseInt(v, 10, 64)
		return i, err == nil
	case json.Number:
		f, err := v.Int64()
		return int64(f), err == nil
	default:
		return 0, false
	}
}

func ToAny(val any) (any, bool) {
	return val, true
}

type Callback[T any] func(val any) (T, bool)

func convertSlice[T any, U any](slice []T) ([]U, bool) {
	var t U
	result := make([]U, len(slice))
	for i, elem := range slice {
		if any(t) == nil {
			result[i] = any(elem).(U)
		} else {
			str, ok := To(t, elem)
			if !ok {
				return nil, false
			}
			result[i] = str
		}
	}
	return result, true
}

// ToSlice - Slice conversion functions
func ToSlice[U any](val any) ([]U, bool) {
	switch v := val.(type) {
	case []string:
		return convertSlice[string, U](v)
	case []any:
		return convertSlice[any, U](v)
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
		return nil, false
	}
}

var re = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})?)|` +
	`(\d{2} \w{3} \d{4} \d{2}:\d{2}:\d{2} [A-Z]{3})|` +
	`(\w{3} \d{1,2},? \d{4} \d{2}:\d{2}(:\d{2})? [AP]M)|` +
	`(\d{4}-\d{2}-\d{2})$`)

func Compare[T any](a T, b any) int {
	var as, bs any
	var ok bool
	var err error
	switch a := any(a).(type) {
	case string:
		if IsValidDateTime(a) {
			as, err = date.Parse(a)
			if err != nil {
				return 0
			}
			bs, ok = To(as, b)
		}
	default:
		as = a
		bs, ok = To(a, b)
		if !ok {
			return 0
		}
	}
	switch a := as.(type) {
	case string:
		b := bs.(string)
		return strings.Compare(a, b)
	case bool:
		return 0
	case time.Time:
		b := bs.(time.Time)
		switch {
		case a.Before(b):
			return -1
		case a.After(b):
			return 1
		default:
			return 0
		}
	case float32:
		b := bs.(float32)
		switch {
		case a < b:
			return -1
		case a > b:
			return 1
		default:
			return 0
		}
	case float64:
		b := bs.(float64)
		switch {
		case a < b:
			return -1
		case a > b:
			return 1
		default:
			return 0
		}
	case uint:
		b := bs.(uint)
		switch {
		case a < b:
			return -1
		case a > b:
			return 1
		default:
			return 0
		}
	case uint8:
		b := bs.(uint8)
		switch {
		case a < b:
			return -1
		case a > b:
			return 1
		default:
			return 0
		}
	case uint16:
		b := bs.(uint16)
		switch {
		case a < b:
			return -1
		case a > b:
			return 1
		default:
			return 0
		}
	case uint32:
		b := bs.(uint32)
		switch {
		case a < b:
			return -1
		case a > b:
			return 1
		default:
			return 0
		}
	case uint64:
		b := bs.(uint64)
		switch {
		case a < b:
			return -1
		case a > b:
			return 1
		default:
			return 0
		}
	case int:
		b := bs.(int)
		switch {
		case a < b:
			return -1
		case a > b:
			return 1
		default:
			return 0
		}
	case int8:
		b := bs.(int8)
		switch {
		case a < b:
			return -1
		case a > b:
			return 1
		default:
			return 0
		}
	case int16:
		b := bs.(int16)
		switch {
		case a < b:
			return -1
		case a > b:
			return 1
		default:
			return 0
		}
	case int32:
		b := bs.(int32)
		switch {
		case a < b:
			return -1
		case a > b:
			return 1
		default:
			return 0
		}
	case int64:
		b := bs.(int64)
		switch {
		case a < b:
			return -1
		case a > b:
			return 1
		default:
			return 0
		}
	default:
		return 0
	}
}

func IsValidDateTime(str string) bool {
	return re.MatchString(str)
}
