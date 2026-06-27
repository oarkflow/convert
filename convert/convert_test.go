package convert

import (
	"encoding/json"
	"errors"
	"math"
	"testing"
	"time"
)

func TestScalarConversions(t *testing.T) {
	i, err := To[int]("42")
	if err != nil || i != 42 {
		t.Fatalf("To[int]: got=%v err=%v", i, err)
	}

	b, err := ToBool("yes")
	if err != nil || !b {
		t.Fatalf("ToBool: got=%v err=%v", b, err)
	}

	f, err := To[float64]([]byte("12.25"))
	if err != nil || f != 12.25 {
		t.Fatalf("To[float64]: got=%v err=%v", f, err)
	}

	s, err := ToString(123)
	if err != nil || s != "123" {
		t.Fatalf("ToString: got=%q err=%v", s, err)
	}

	j, err := To[int64](json.Number("99"))
	if err != nil || j != 99 {
		t.Fatalf("json.Number: got=%v err=%v", j, err)
	}
}

func TestAs(t *testing.T) {
	var a = 1.2
	var b = "5"

	x, err := As(a, b)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := x.(string); !ok || got != "1.2" {
		t.Fatalf("As(float,string): got=%T(%v)", x, x)
	}

	var n int
	y, err := As("123", &n)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := y.(int); !ok || got != 123 {
		t.Fatalf("As(string,*int): got=%T(%v)", y, y)
	}

	z, err := AsTo("3.5", float64(0))
	if err != nil || z != 3.5 {
		t.Fatalf("AsTo: got=%v err=%v", z, err)
	}
}

func TestTime(t *testing.T) {
	tm, err := ToTime("2026-06-27T12:30:45Z")
	if err != nil {
		t.Fatal(err)
	}
	if tm.UTC().Format(time.RFC3339) != "2026-06-27T12:30:45Z" {
		t.Fatalf("unexpected time %s", tm.UTC().Format(time.RFC3339))
	}

	tm, err = To[time.Time](int64(1_700_000_000))
	if err != nil {
		t.Fatal(err)
	}
	if tm.Unix() != 1_700_000_000 {
		t.Fatalf("unexpected unix %d", tm.Unix())
	}

	s, err := ToString(tm.UTC())
	if err != nil {
		t.Fatal(err)
	}
	if s == "" {
		t.Fatal("empty formatted time")
	}
}

func TestDuration(t *testing.T) {
	d, err := ToDuration("1h30m")
	if err != nil || d != 90*time.Minute {
		t.Fatalf("duration string: got=%v err=%v", d, err)
	}

	d, err = ToDurationSeconds(1.5)
	if err != nil || d != 1500*time.Millisecond {
		t.Fatalf("duration seconds: got=%v err=%v", d, err)
	}

	d, err = To[time.Duration](int64(time.Second))
	if err != nil || d != time.Second {
		t.Fatalf("duration generic: got=%v err=%v", d, err)
	}
}

func TestNamedAliases(t *testing.T) {
	type Port int
	type Name string

	p, err := ToIntLike[Port]("8080")
	if err != nil || p != 8080 {
		t.Fatalf("Port: got=%v err=%v", p, err)
	}

	n, err := ToStringLike[Name](123)
	if err != nil || n != "123" {
		t.Fatalf("Name: got=%v err=%v", n, err)
	}
}

func TestOverflowAndInvalid(t *testing.T) {
	_, err := ToInt8("128")
	if !errors.Is(err, ErrOverflow) {
		t.Fatalf("expected overflow, got %v", err)
	}

	_, err = ToInt64(12.5)
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected invalid, got %v", err)
	}

	_, err = ToUint64(-1)
	if !errors.Is(err, ErrOverflow) {
		t.Fatalf("expected overflow, got %v", err)
	}

	_, err = ToBool(math.NaN())
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected invalid, got %v", err)
	}
}

func TestAppendString(t *testing.T) {
	buf := make([]byte, 0, 64)
	var err error

	buf, err = AppendString(buf, 123)
	if err != nil {
		t.Fatal(err)
	}
	buf, err = AppendString(buf, "|")
	if err != nil {
		t.Fatal(err)
	}
	buf, err = AppendString(buf, true)
	if err != nil {
		t.Fatal(err)
	}

	if string(buf) != "123|true" {
		t.Fatalf("unexpected buffer %q", string(buf))
	}
}

func TestOptimizedExistingFunctions(t *testing.T) {
	i, err := ToInt64("123456")
	if err != nil || i != 123456 {
		t.Fatalf("ToInt64 got %d err %v", i, err)
	}
	u, err := ToUint64("123456")
	if err != nil || u != 123456 {
		t.Fatalf("ToUint64 got %d err %v", u, err)
	}
	b, err := ToBool("ON")
	if err != nil || !b {
		t.Fatalf("ToBool got %v err %v", b, err)
	}
}

func TestExistingFunctionsFallBackToStdlibBaseZero(t *testing.T) {
	i, err := ToInt64("0x10")
	if err != nil || i != 16 {
		t.Fatalf("hex ToInt64 got %d err %v", i, err)
	}
	u, err := ToUint64("010")
	if err != nil || u != 8 {
		t.Fatalf("octal ToUint64 got %d err %v", u, err)
	}
}

func TestAsInto(t *testing.T) {
	var n int
	if err := AsInto("42", &n); err != nil || n != 42 {
		t.Fatalf("AsInto int got %d err %v", n, err)
	}
	var d time.Duration
	if err := AsInto("1h30m", &d); err != nil || d != 90*time.Minute {
		t.Fatalf("AsInto duration got %v err %v", d, err)
	}
}
