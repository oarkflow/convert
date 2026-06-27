package convert

import (
	"strconv"
	"testing"
	"time"
)

var (
	benchBool     bool
	benchString   string
	benchBytes    []byte
	benchInt      int
	benchInt8     int8
	benchInt16    int16
	benchInt32    int32
	benchInt64    int64
	benchUint     uint
	benchUint8    uint8
	benchUint16   uint16
	benchUint32   uint32
	benchUint64   uint64
	benchUintptr  uintptr
	benchFloat32  float32
	benchFloat64  float64
	benchDuration time.Duration
	benchTime     time.Time
	benchAny      any
)

func BenchmarkStdlibParseInt(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		x, err := strconv.ParseInt("123456", 10, 64)
		if err != nil || x != 123456 {
			b.Fatal(x, err)
		}
		benchInt64 = x
	}
}

func BenchmarkTypedScalarConversions(b *testing.B) {
	b.Run("ToBool/bool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToBool(true)
			if err != nil || !x {
				b.Fatal(x, err)
			}
			benchBool = x
		}
	})
	b.Run("ToBool/string_true", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToBool("true")
			if err != nil || !x {
				b.Fatal(x, err)
			}
			benchBool = x
		}
	})
	b.Run("ToBool/int", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToBool(1)
			if err != nil || !x {
				b.Fatal(x, err)
			}
			benchBool = x
		}
	})
	b.Run("ToBool/float64", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToBool(1.25)
			if err != nil || !x {
				b.Fatal(x, err)
			}
			benchBool = x
		}
	})

	b.Run("ToString/string", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToString("value")
			if err != nil || x != "value" {
				b.Fatal(x, err)
			}
			benchString = x
		}
	})
	b.Run("ToString/bytes", func(b *testing.B) {
		in := []byte("value")
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToString(in)
			if err != nil || x != "value" {
				b.Fatal(x, err)
			}
			benchString = x
		}
	})
	b.Run("ToString/int", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToString(123456)
			if err != nil || x != "123456" {
				b.Fatal(x, err)
			}
			benchString = x
		}
	})
	b.Run("ToString/bool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToString(true)
			if err != nil || x != "true" {
				b.Fatal(x, err)
			}
			benchString = x
		}
	})
	b.Run("ToString/float64", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToString(12.5)
			if err != nil || x != "12.5" {
				b.Fatal(x, err)
			}
			benchString = x
		}
	})
}

func BenchmarkTypedIntegerConversions(b *testing.B) {
	b.Run("ToInt/int", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToInt(123456)
			if err != nil || x != 123456 {
				b.Fatal(x, err)
			}
			benchInt = x
		}
	})
	b.Run("ToInt/string", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToInt("123456")
			if err != nil || x != 123456 {
				b.Fatal(x, err)
			}
			benchInt = x
		}
	})
	b.Run("ToInt/bytes", func(b *testing.B) {
		in := []byte("123456")
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToInt(in)
			if err != nil || x != 123456 {
				b.Fatal(x, err)
			}
			benchInt = x
		}
	})
	b.Run("ToInt/int64", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToInt(int64(123456))
			if err != nil || x != 123456 {
				b.Fatal(x, err)
			}
			benchInt = x
		}
	})
	b.Run("ToInt/uint64", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToInt(uint64(123456))
			if err != nil || x != 123456 {
				b.Fatal(x, err)
			}
			benchInt = x
		}
	})
	b.Run("ToInt/float64_integral", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToInt(float64(123456))
			if err != nil || x != 123456 {
				b.Fatal(x, err)
			}
			benchInt = x
		}
	})
	b.Run("ToInt/bool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToInt(true)
			if err != nil || x != 1 {
				b.Fatal(x, err)
			}
			benchInt = x
		}
	})

	b.Run("ToInt8/string", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToInt8("120")
			if err != nil || x != 120 {
				b.Fatal(x, err)
			}
			benchInt8 = x
		}
	})
	b.Run("ToInt16/string", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToInt16("12345")
			if err != nil || x != 12345 {
				b.Fatal(x, err)
			}
			benchInt16 = x
		}
	})
	b.Run("ToInt32/string", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToInt32("123456")
			if err != nil || x != 123456 {
				b.Fatal(x, err)
			}
			benchInt32 = x
		}
	})
	b.Run("ToInt64/int64", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToInt64(int64(123456))
			if err != nil || x != 123456 {
				b.Fatal(x, err)
			}
			benchInt64 = x
		}
	})
	b.Run("ToInt64/string", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToInt64("123456")
			if err != nil || x != 123456 {
				b.Fatal(x, err)
			}
			benchInt64 = x
		}
	})
}

func BenchmarkTypedUnsignedConversions(b *testing.B) {
	b.Run("ToUint/uint", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToUint(uint(123456))
			if err != nil || x != 123456 {
				b.Fatal(x, err)
			}
			benchUint = x
		}
	})
	b.Run("ToUint/string", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToUint("123456")
			if err != nil || x != 123456 {
				b.Fatal(x, err)
			}
			benchUint = x
		}
	})
	b.Run("ToUint/int", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToUint(123456)
			if err != nil || x != 123456 {
				b.Fatal(x, err)
			}
			benchUint = x
		}
	})
	b.Run("ToUint8/string", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToUint8("250")
			if err != nil || x != 250 {
				b.Fatal(x, err)
			}
			benchUint8 = x
		}
	})
	b.Run("ToUint16/string", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToUint16("65000")
			if err != nil || x != 65000 {
				b.Fatal(x, err)
			}
			benchUint16 = x
		}
	})
	b.Run("ToUint32/string", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToUint32("123456")
			if err != nil || x != 123456 {
				b.Fatal(x, err)
			}
			benchUint32 = x
		}
	})
	b.Run("ToUint64/uint64", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToUint64(uint64(123456))
			if err != nil || x != 123456 {
				b.Fatal(x, err)
			}
			benchUint64 = x
		}
	})
	b.Run("ToUint64/string", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToUint64("123456")
			if err != nil || x != 123456 {
				b.Fatal(x, err)
			}
			benchUint64 = x
		}
	})
	b.Run("ToUintptr/string", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToUintptr("123456")
			if err != nil || x != 123456 {
				b.Fatal(x, err)
			}
			benchUintptr = x
		}
	})
}

func BenchmarkTypedFloatConversions(b *testing.B) {
	b.Run("ToFloat32/float32", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToFloat32(float32(12.5))
			if err != nil || x != 12.5 {
				b.Fatal(x, err)
			}
			benchFloat32 = x
		}
	})
	b.Run("ToFloat32/string", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToFloat32("12.5")
			if err != nil || x != 12.5 {
				b.Fatal(x, err)
			}
			benchFloat32 = x
		}
	})
	b.Run("ToFloat64/float64", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToFloat64(float64(12.5))
			if err != nil || x != 12.5 {
				b.Fatal(x, err)
			}
			benchFloat64 = x
		}
	})
	b.Run("ToFloat64/string", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToFloat64("12.5")
			if err != nil || x != 12.5 {
				b.Fatal(x, err)
			}
			benchFloat64 = x
		}
	})
	b.Run("ToFloat64/int", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToFloat64(123456)
			if err != nil || x != 123456 {
				b.Fatal(x, err)
			}
			benchFloat64 = x
		}
	})
	b.Run("ToFloat64/bool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToFloat64(true)
			if err != nil || x != 1 {
				b.Fatal(x, err)
			}
			benchFloat64 = x
		}
	})
}

func BenchmarkGenericToConversions(b *testing.B) {
	b.Run("To[int]/string", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := To[int]("123456")
			if err != nil || x != 123456 {
				b.Fatal(x, err)
			}
			benchInt = x
		}
	})
	b.Run("To[int]/int", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := To[int](123456)
			if err != nil || x != 123456 {
				b.Fatal(x, err)
			}
			benchInt = x
		}
	})
	b.Run("To[int64]/string", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := To[int64]("123456")
			if err != nil || x != 123456 {
				b.Fatal(x, err)
			}
			benchInt64 = x
		}
	})
	b.Run("To[uint64]/string", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := To[uint64]("123456")
			if err != nil || x != 123456 {
				b.Fatal(x, err)
			}
			benchUint64 = x
		}
	})
	b.Run("To[float64]/string", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := To[float64]("12.5")
			if err != nil || x != 12.5 {
				b.Fatal(x, err)
			}
			benchFloat64 = x
		}
	})
	b.Run("To[bool]/string", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := To[bool]("true")
			if err != nil || !x {
				b.Fatal(x, err)
			}
			benchBool = x
		}
	})
	b.Run("To[string]/int", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := To[string](123456)
			if err != nil || x != "123456" {
				b.Fatal(x, err)
			}
			benchString = x
		}
	})
	b.Run("To[Duration]/string", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := To[time.Duration]("1h30m")
			if err != nil || x != 90*time.Minute {
				b.Fatal(x, err)
			}
			benchDuration = x
		}
	})
	b.Run("To[Time]/unix_string", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := To[time.Time]("1710000000")
			if err != nil || x.Unix() != 1710000000 {
				b.Fatal(x, err)
			}
			benchTime = x
		}
	})
}

func BenchmarkTargetBasedConversions(b *testing.B) {
	b.Run("AsTo/int_from_string", func(b *testing.B) {
		var target int
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := AsTo("123456", target)
			if err != nil || x != 123456 {
				b.Fatal(x, err)
			}
			benchInt = x
		}
	})
	b.Run("AsTo/string_from_float64", func(b *testing.B) {
		var target string
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := AsTo(12.5, target)
			if err != nil || x != "12.5" {
				b.Fatal(x, err)
			}
			benchString = x
		}
	})
	b.Run("AsTo/duration_from_string", func(b *testing.B) {
		var target time.Duration
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := AsTo("1h30m", target)
			if err != nil || x != 90*time.Minute {
				b.Fatal(x, err)
			}
			benchDuration = x
		}
	})
	b.Run("As/int_from_string", func(b *testing.B) {
		var target int
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := As("123456", target)
			if err != nil || x.(int) != 123456 {
				b.Fatal(x, err)
			}
			benchAny = x
		}
	})
	b.Run("As/string_from_float64", func(b *testing.B) {
		var target string
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := As(12.5, target)
			if err != nil || x.(string) != "12.5" {
				b.Fatal(x, err)
			}
			benchAny = x
		}
	})
	b.Run("As/duration_from_string", func(b *testing.B) {
		var target time.Duration
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := As("1h30m", target)
			if err != nil || x.(time.Duration) != 90*time.Minute {
				b.Fatal(x, err)
			}
			benchAny = x
		}
	})
}

func BenchmarkTimeAndDurationConversions(b *testing.B) {
	b.Run("ToDuration/duration", func(b *testing.B) {
		in := 90 * time.Minute
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToDuration(in)
			if err != nil || x != in {
				b.Fatal(x, err)
			}
			benchDuration = x
		}
	})
	b.Run("ToDuration/string", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToDuration("1h30m")
			if err != nil || x != 90*time.Minute {
				b.Fatal(x, err)
			}
			benchDuration = x
		}
	})
	b.Run("ToDuration/int64_ns", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToDuration(int64(time.Second))
			if err != nil || x != time.Second {
				b.Fatal(x, err)
			}
			benchDuration = x
		}
	})
	b.Run("ToDurationSeconds/int", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToDurationSeconds(90)
			if err != nil || x != 90*time.Second {
				b.Fatal(x, err)
			}
			benchDuration = x
		}
	})
	b.Run("ToDurationMilliseconds/string", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToDurationMilliseconds("1500ms")
			if err != nil || x != 1500*time.Millisecond {
				b.Fatal(x, err)
			}
			benchDuration = x
		}
	})
	b.Run("ToTime/time", func(b *testing.B) {
		in := time.Unix(1710000000, 0).UTC()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToTime(in)
			if err != nil || !x.Equal(in) {
				b.Fatal(x, err)
			}
			benchTime = x
		}
	})
	b.Run("ToTime/unix_int64", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToTime(int64(1710000000))
			if err != nil || x.Unix() != 1710000000 {
				b.Fatal(x, err)
			}
			benchTime = x
		}
	})
	b.Run("ToTime/unix_string", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToTime("1710000000")
			if err != nil || x.Unix() != 1710000000 {
				b.Fatal(x, err)
			}
			benchTime = x
		}
	})
	b.Run("ToTime/rfc3339", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToTime("2024-03-09T16:00:00Z")
			if err != nil || x.Unix() != 1710000000 {
				b.Fatal(x, err)
			}
			benchTime = x
		}
	})
	b.Run("ToUnix/time", func(b *testing.B) {
		in := time.Unix(1710000000, 0).UTC()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToUnix(in)
			if err != nil || x != 1710000000 {
				b.Fatal(x, err)
			}
			benchInt64 = x
		}
	})
	b.Run("ToUnixMilli/time", func(b *testing.B) {
		in := time.UnixMilli(1710000000123).UTC()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToUnixMilli(in)
			if err != nil || x != 1710000000123 {
				b.Fatal(x, err)
			}
			benchInt64 = x
		}
	})
}

func BenchmarkDefaultAndMustConversions(b *testing.B) {
	b.Run("Default/int_success", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x := Default[int]("123456", 0)
			if x != 123456 {
				b.Fatal(x)
			}
			benchInt = x
		}
	})
	b.Run("Default/int_fallback", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x := Default[int]("bad", 7)
			if x != 7 {
				b.Fatal(x)
			}
			benchInt = x
		}
	})
	b.Run("Must/int", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x := Must[int]("123456")
			if x != 123456 {
				b.Fatal(x)
			}
			benchInt = x
		}
	})
}

func BenchmarkNamedScalarConversions(b *testing.B) {
	type Port int
	type UserID uint64
	type Enabled bool
	b.Run("ToIntLike/Port", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToIntLike[Port]("8080")
			if err != nil || x != 8080 {
				b.Fatal(x, err)
			}
			benchInt = int(x)
		}
	})
	b.Run("ToUint64Like/UserID", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToUint64Like[UserID]("123456")
			if err != nil || x != 123456 {
				b.Fatal(x, err)
			}
			benchUint64 = uint64(x)
		}
	})
	b.Run("ToBoolLike/Enabled", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x, err := ToBoolLike[Enabled]("true")
			if err != nil || !x {
				b.Fatal(x, err)
			}
			benchBool = bool(x)
		}
	})
}

func BenchmarkAppendConversions(b *testing.B) {
	b.Run("AppendString/int", func(b *testing.B) {
		buf := make([]byte, 0, 32)
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf = buf[:0]
			var err error
			buf, err = AppendString(buf, 123456)
			if err != nil || len(buf) == 0 {
				b.Fatal(err)
			}
			benchBytes = buf
		}
	})
	b.Run("AppendString/int64_negative", func(b *testing.B) {
		buf := make([]byte, 0, 32)
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf = buf[:0]
			var err error
			buf, err = AppendString(buf, int64(-123456))
			if err != nil || len(buf) == 0 {
				b.Fatal(err)
			}
			benchBytes = buf
		}
	})
	b.Run("AppendString/uint64", func(b *testing.B) {
		buf := make([]byte, 0, 32)
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf = buf[:0]
			var err error
			buf, err = AppendString(buf, uint64(123456))
			if err != nil || len(buf) == 0 {
				b.Fatal(err)
			}
			benchBytes = buf
		}
	})
	b.Run("AppendString/bool", func(b *testing.B) {
		buf := make([]byte, 0, 32)
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf = buf[:0]
			var err error
			buf, err = AppendString(buf, true)
			if err != nil || len(buf) == 0 {
				b.Fatal(err)
			}
			benchBytes = buf
		}
	})
	b.Run("AppendString/string", func(b *testing.B) {
		buf := make([]byte, 0, 32)
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf = buf[:0]
			var err error
			buf, err = AppendString(buf, "value")
			if err != nil || len(buf) == 0 {
				b.Fatal(err)
			}
			benchBytes = buf
		}
	})
	b.Run("AppendString/float64", func(b *testing.B) {
		buf := make([]byte, 0, 32)
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf = buf[:0]
			var err error
			buf, err = AppendString(buf, 12.5)
			if err != nil || len(buf) == 0 {
				b.Fatal(err)
			}
			benchBytes = buf
		}
	})
}

func BenchmarkTargetBasedConversionsNoBoxing(b *testing.B) {
	b.Run("AsInto/int_from_string", func(b *testing.B) {
		var out int
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if err := AsInto("123456", &out); err != nil || out != 123456 {
				b.Fatal(out, err)
			}
			benchInt = out
		}
	})
	b.Run("AsInto/duration_from_string", func(b *testing.B) {
		var out time.Duration
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if err := AsInto("1h30m", &out); err != nil || out != 90*time.Minute {
				b.Fatal(out, err)
			}
			benchDuration = out
		}
	})
	b.Run("AsInto/float64_from_string", func(b *testing.B) {
		var out float64
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if err := AsInto("12.5", &out); err != nil || out != 12.5 {
				b.Fatal(out, err)
			}
			benchFloat64 = out
		}
	})
}
