package oarkflowmoney

import (
	"testing"

	"github.com/oarkflow/convert"
	"github.com/oarkflow/money"
)

func BenchmarkOarkflowMoneyConversion(b *testing.B) {
	Register("USD")
	b.Run("ToMoney/string_code_amount", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			m, err := ToMoney("USD 12.50", "")
			if err != nil || m.Currency().Code != "USD" {
				b.Fatal(m, err)
			}
		}
	})
	b.Run("ToMoney/map", func(b *testing.B) {
		in := map[string]any{"amount": "12.50", "currency": "USD"}
		for i := 0; i < b.N; i++ {
			m, err := ToMoney(in, "")
			if err != nil || m.Currency().Code != "USD" {
				b.Fatal(m, err)
			}
		}
	})
	b.Run("ConvertRegistry/string", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			m, err := convert.Convert[money.Money]("12.50")
			if err != nil || m.Currency().Code != "USD" {
				b.Fatal(m, err)
			}
		}
	})
}
