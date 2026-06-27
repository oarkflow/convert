package convert

import "testing"

func FuzzToInt(f *testing.F) {
	for _, s := range []string{"0", "1", "-1", "123", "abc", "999999999999999999999"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) { _, _ = ToInt(s) })
}
func FuzzToUint(f *testing.F) {
	for _, s := range []string{"0", "1", "-1", "123", "abc"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) { _, _ = ToUint64(s) })
}
func FuzzToFloat(f *testing.F) {
	for _, s := range []string{"0", "1.2", "-1.5", "NaN", "abc"} {
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
	for _, s := range []string{"1s", "2m", "1h30m", "abc"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) { _, _ = ToDuration(s) })
}
func FuzzToTime(f *testing.F) {
	for _, s := range []string{"1710000000", "2026-06-27T12:00:00Z", "2026-06-27", "abc"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) { _, _ = ToTime(s) })
}
func FuzzSizeParser(f *testing.F) {
	for _, s := range []string{"1B", "2KB", "64MiB", "abc"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) { _, _ = ToBytesSize(s) })
}
