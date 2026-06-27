package convert

import (
	"encoding/hex"
	"strconv"
	"testing"
	"time"
)

func FuzzToInt(f *testing.F) {
	for _, s := range []string{"0", "1", "-1", "123456", "abc", "999999999999999999999"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) { _, _ = ToInt(s) })
}
func FuzzToUint(f *testing.F) {
	for _, s := range []string{"0", "1", "123456", "-1", "abc"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) { _, _ = ToUint64(s) })
}
func FuzzToFloat(f *testing.F) {
	for _, s := range []string{"0", "1.2", "-1.2", "NaN", "Inf", "abc"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) { _, _ = ToFloat64(s) })
}
func FuzzToBool(f *testing.F) {
	for _, s := range []string{"true", "false", "1", "0", "yes", "no", "abc"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) { _, _ = ToBool(s) })
}
func FuzzToDuration(f *testing.F) {
	for _, s := range []string{"1s", "2m", "1h30m", "bad", "123"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) { _, _ = ToDuration(s) })
}
func FuzzToTime(f *testing.F) {
	for _, s := range []string{"1710000000", time.Now().UTC().Format(time.RFC3339Nano), "2026-06-27", "bad"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) { _, _ = ToTime(s) })
}
func FuzzSizeParser(f *testing.F) {
	for _, s := range []string{"1KB", "64MiB", "0", "bad", "999999999999999999999PB"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) { _, _ = ToBytesSize(s) })
}
func FuzzPathGetSet(f *testing.F) {
	f.Add("a.b", "123")
	f.Fuzz(func(t *testing.T, path, val string) {
		var dst struct{ A struct{ B string } }
		_ = Set(&dst, "A.B", val)
		_, _ = GetAny(map[string]any{"a": map[string]any{"b": val}}, path)
	})
}
func FuzzNormalizeJSONNumbers(f *testing.F) {
	for _, s := range []string{"1", "1.2", "bad"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_ = NormalizeJSONNumbers(strconv.NumError{})
		_ = NormalizeJSONNumbers(map[string]any{"n": s})
	})
}
func FuzzHex(f *testing.F) {
	f.Add("6869")
	f.Fuzz(func(t *testing.T, s string) { _, _ = hex.DecodeString(s); _, _ = FromHex(s) })
}
