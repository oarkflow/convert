package convert

import (
	"testing"
	"time"
)

func TestToComplex64(t *testing.T) {
	tests := []struct {
		input    any
		expected complex64
		hasError bool
	}{
		{"1+2i", 1 + 2i, false},
		{complex128(3 + 4i), 3 + 4i, false},
		{nil, 0, false},
		{"invalid", 0, true},
	}

	for _, test := range tests {
		result, err := ToComplex64(test.input)
		if (err != nil) != test.hasError {
			t.Errorf("ToComplex64(%v) error = %v, want error = %v", test.input, err, test.hasError)
		}
		if !test.hasError && result != test.expected {
			t.Errorf("ToComplex64(%v) = %v, want %v", test.input, result, test.expected)
		}
	}
}

func TestToDuration(t *testing.T) {
	tests := []struct {
		input    any
		expected time.Duration
		hasError bool
	}{
		{"1h", time.Hour, false},
		{int64(1000000000), time.Second, false},
		{nil, 0, false},
		{"invalid", 0, true},
	}

	for _, test := range tests {
		result, err := ToDuration(test.input)
		if (err != nil) != test.hasError {
			t.Errorf("ToDuration(%v) error = %v, want error = %v", test.input, err, test.hasError)
		}
		if !test.hasError && result != test.expected {
			t.Errorf("ToDuration(%v) = %v, want %v", test.input, result, test.expected)
		}
	}
}

func BenchmarkToInt64(b *testing.B) {
	val := "123"
	for i := 0; i < b.N; i++ {
		_, _ = ToInt64(val)
	}
}

func BenchmarkToFloat64(b *testing.B) {
	val := "123.45"
	for i := 0; i < b.N; i++ {
		_, _ = ToFloat64(val)
	}
}
