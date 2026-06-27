package convert

import (
	"encoding/base64"
	"encoding/hex"
	"strconv"
	"testing"
	"time"
)

func TestDifferentialStdlibParseInt(t *testing.T) {
	inputs := []string{"0", "1", "-1", "42", "123456", "0x10", "010"}
	for _, s := range inputs {
		got, err := ToInt64(s)
		want, werr := strconv.ParseInt(s, 0, 64)
		if (err == nil) != (werr == nil) || (err == nil && got != want) {
			t.Fatalf("%s got %d/%v want %d/%v", s, got, err, want, werr)
		}
	}
}
func TestDifferentialStdlibParseUint(t *testing.T) {
	inputs := []string{"0", "1", "42", "123456", "0x10", "010"}
	for _, s := range inputs {
		got, err := ToUint64(s)
		want, werr := strconv.ParseUint(s, 0, 64)
		if (err == nil) != (werr == nil) || (err == nil && got != want) {
			t.Fatalf("%s got %d/%v want %d/%v", s, got, err, want, werr)
		}
	}
}
func TestDifferentialStdlibParseFloat(t *testing.T) {
	inputs := []string{"0", "1.25", "-1.25", "1e3"}
	for _, s := range inputs {
		got, err := ToFloat64(s)
		want, werr := strconv.ParseFloat(s, 64)
		if (err == nil) != (werr == nil) || (err == nil && got != want) {
			t.Fatalf("%s got %f/%v want %f/%v", s, got, err, want, werr)
		}
	}
}
func TestDifferentialStdlibDurationTimeEncoding(t *testing.T) {
	ds := []string{"1s", "2m", "1h30m", "250ms"}
	for _, s := range ds {
		got, err := ToDuration(s)
		want, werr := time.ParseDuration(s)
		if (err == nil) != (werr == nil) || got != want {
			t.Fatalf("duration %s", s)
		}
	}
	now := time.Now().UTC().Truncate(0).Format(time.RFC3339Nano)
	if got, err := ToTime(now); err != nil || got.Format(time.RFC3339Nano) != now {
		t.Fatalf("time got %v err %v", got, err)
	}
	raw := []byte("hello")
	hx, _ := ToHex(raw)
	if dec, _ := hex.DecodeString(hx); string(dec) != "hello" {
		t.Fatal("hex")
	}
	b64, _ := ToBase64(raw)
	if dec, _ := base64.StdEncoding.DecodeString(b64); string(dec) != "hello" {
		t.Fatal("b64")
	}
}
