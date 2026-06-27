package convert

import (
	"strconv"
	"testing"
	"time"
)

func TestDifferentialStdlibNumericAndDuration(t *testing.T) {
	ints := []string{"0", "1", "42", "-42", "123456"}
	for _, s := range ints {
		want, wantErr := strconv.ParseInt(s, 0, 64)
		got, gotErr := ToInt64(s)
		if (wantErr != nil) != (gotErr != nil) || want != got {
			t.Fatalf("int %s got %v/%v want %v/%v", s, got, gotErr, want, wantErr)
		}
	}
	floats := []string{"0", "1.25", "-2.5", "12"}
	for _, s := range floats {
		want, wantErr := strconv.ParseFloat(s, 64)
		got, gotErr := ToFloat64(s)
		if (wantErr != nil) != (gotErr != nil) || want != got {
			t.Fatalf("float %s got %v/%v want %v/%v", s, got, gotErr, want, wantErr)
		}
	}
	durs := []string{"1s", "2m", "1h30m", "100ms"}
	for _, s := range durs {
		want, wantErr := time.ParseDuration(s)
		got, gotErr := ToDuration(s)
		if (wantErr != nil) != (gotErr != nil) || want != got {
			t.Fatalf("duration %s got %v/%v want %v/%v", s, got, gotErr, want, wantErr)
		}
	}
}
