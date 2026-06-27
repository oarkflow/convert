package main

import (
	"fmt"

	"github.com/oarkflow/convert"
	"github.com/oarkflow/convert/adapters/oarkflowmoney"
	"github.com/oarkflow/money"
)

func main() {
	oarkflowmoney.Register("USD")

	price, _ := oarkflowmoney.ToMoney("USD 12.50", "")
	fee, _ := convert.Convert[money.Money](map[string]any{"amount": "2.25", "currency": "USD"})
	fallbackCurrency, _ := convert.Convert[money.Money]("5.00")
	graphValue, _ := convert.ConvertGraph[money.Money]([]byte("NPR 100"))
	minor, _ := oarkflowmoney.NewFromMinor(999, "USD")

	fmt.Println(oarkflowmoney.ToString(price))
	fmt.Println(oarkflowmoney.ToString(fee))
	fmt.Println(oarkflowmoney.ToString(fallbackCurrency))
	fmt.Println(oarkflowmoney.ToString(graphValue))
	fmt.Println(oarkflowmoney.ToString(minor))
}
