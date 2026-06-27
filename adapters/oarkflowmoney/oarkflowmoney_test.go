package oarkflowmoney

import (
	"testing"

	"github.com/oarkflow/convert"
	"github.com/oarkflow/money"
)

func TestToMoneyInputs(t *testing.T) {
	cases := []struct {
		in       any
		currency string
		minor    int64
	}{
		{"USD 12.50", "USD", 1250},
		{"12.50 USD", "USD", 1250},
		{"US$12.50", "USD", 1250},
		{map[string]any{"amount": "12.50", "currency": "USD"}, "USD", 1250},
		{map[string]any{"minor": int64(1250), "currency": "USD"}, "USD", 1250},
		{[]any{"USD", "12.50"}, "USD", 1250},
		{12.50, "USD", 1250},
		{int64(12), "USD", 1200},
	}
	for _, tc := range cases {
		m, err := ToMoney(tc.in, "USD")
		if err != nil {
			t.Fatalf("%#v: %v", tc.in, err)
		}
		if m.Currency().Code != tc.currency || m.Minor() != tc.minor {
			t.Fatalf("%#v -> %s %d", tc.in, m.Currency().Code, m.Minor())
		}
	}
}

func TestRegisterGenericConversion(t *testing.T) {
	Register("NPR")
	defer Unregister()
	m, err := convert.Convert[money.Money]("99.90")
	if err != nil {
		t.Fatal(err)
	}
	if m.Currency().Code != "NPR" || m.Minor() != 9990 {
		t.Fatalf("got %s %d", m.Currency().Code, m.Minor())
	}
	m2, err := convert.ConvertGraph[money.Money]([]byte("USD 10"))
	if err != nil {
		t.Fatal(err)
	}
	if m2.Currency().Code != "USD" || m2.Minor() != 1000 {
		t.Fatalf("got %s %d", m2.Currency().Code, m2.Minor())
	}
}

func TestHelpers(t *testing.T) {
	m, err := NewFromMinor(1250, "USD")
	if err != nil {
		t.Fatal(err)
	}
	if Currency(m) != "USD" || Minor(m) != 1250 || AmountString(m) != "12.50" || ToString(m) != "USD 12.50" {
		t.Fatalf("helpers failed: %s %d %q %q", Currency(m), Minor(m), AmountString(m), ToString(m))
	}
}
