// Package oarkflowmoney integrates github.com/oarkflow/money with convert.
//
// This adapter targets the real github.com/oarkflow/money API. The money
// package stores values as unexported minor units and exposes construction and
// access through Currency, Money, Parse, ParseMoney, ParseAny, New, NewFromMinor,
// NewFromFloat, Minor, Amount, Float64, Currency, Format and String methods.
package oarkflowmoney

import (
	"errors"
	"math"
	"reflect"
	"strconv"
	"strings"

	"github.com/oarkflow/convert"
	"github.com/oarkflow/money"
)

var ErrMissingCurrency = errors.New("oarkflowmoney: missing currency")

// Register installs money.Money support into the generic convert registry and
// conversion graph. After calling Register("USD"), these work:
//
//	convert.Convert[money.Money]("12.50")
//	convert.ConvertGraph[money.Money]([]byte("NPR 100"))
func Register(defaultCurrency string) {
	defaultCurrency = strings.ToUpper(strings.TrimSpace(defaultCurrency))
	convert.Register[money.Money](func(v any, ctx convert.Context) (money.Money, error) {
		return ToMoney(v, defaultCurrency)
	})
	convert.RegisterEdge[string, money.Money](func(s string) (money.Money, error) { return ToMoney(s, defaultCurrency) })
	convert.RegisterEdge[[]byte, money.Money](func(b []byte) (money.Money, error) { return ToMoney(b, defaultCurrency) })
}

// Unregister removes the generic money.Money converter from convert's registry.
func Unregister() { convert.Unregister[money.Money]() }

// ToMoney converts common runtime values to money.Money.
//
// Supported inputs:
//   - money.Money and *money.Money
//   - string/[]byte: "USD 12.50", "12.50 USD", "US$12.50", or "12.50" with default currency
//   - numeric scalar: treated as major units in the default currency
//   - map with amount/currency keys; map with minor/currency keys for minor units
//   - struct with Amount/Value/Total or Minor and Currency/Code/CurrencyCode fields
//   - []any pairs like []any{"USD", "12.50"} or []any{"12.50", "USD"}
func ToMoney(v any, defaultCurrency string) (money.Money, error) {
	switch x := v.(type) {
	case money.Money:
		if x.Currency().IsValid() {
			return x, nil
		}
		if defaultCurrency == "" {
			return money.Money{}, ErrMissingCurrency
		}
		c, err := currency(defaultCurrency)
		if err != nil {
			return money.Money{}, err
		}
		return money.NewFromMinor(x.Minor(), c), nil
	case *money.Money:
		if x == nil {
			return money.Money{}, convert.ErrNil
		}
		return ToMoney(*x, defaultCurrency)
	case string:
		return parseMoneyString(x, defaultCurrency)
	case []byte:
		return parseMoneyString(string(x), defaultCurrency)
	case []any:
		return moneyFromAnySlice(x, defaultCurrency)
	case map[string]any:
		return moneyFromMap(x, defaultCurrency)
	case map[string]string:
		m := make(map[string]any, len(x))
		for k, v := range x {
			m[k] = v
		}
		return moneyFromMap(m, defaultCurrency)
	}

	if isNumeric(v) {
		return NewFromMajor(v, defaultCurrency)
	}

	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return money.Money{}, convert.ErrNil
	}
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return money.Money{}, convert.ErrNil
		}
		rv = rv.Elem()
	}
	switch rv.Kind() {
	case reflect.Map:
		m := make(map[string]any, rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			k, err := convert.ToString(iter.Key().Interface())
			if err != nil {
				return money.Money{}, err
			}
			m[k] = iter.Value().Interface()
		}
		return moneyFromMap(m, defaultCurrency)
	case reflect.Struct:
		return moneyFromStruct(rv, defaultCurrency)
	}
	return money.Money{}, convert.ErrUnsupported
}

// NewFromMajor creates Money from a major-unit amount such as "12.50" dollars.
// Integer inputs are major units, not minor units. Use NewFromMinor for cents.
func NewFromMajor(amount any, code string) (money.Money, error) {
	c, err := currency(code)
	if err != nil {
		return money.Money{}, err
	}
	switch x := amount.(type) {
	case string:
		return money.Parse(cleanAmount(x), c)
	case []byte:
		return money.Parse(cleanAmount(string(x)), c)
	case float64:
		if math.IsNaN(x) || math.IsInf(x, 0) {
			return money.Money{}, convert.ErrInvalid
		}
		return money.NewFromFloat(x, c), nil
	case float32:
		f := float64(x)
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return money.Money{}, convert.ErrInvalid
		}
		return money.NewFromFloat(f, c), nil
	case int:
		return money.New(int64(x), c), nil
	case int8:
		return money.New(int64(x), c), nil
	case int16:
		return money.New(int64(x), c), nil
	case int32:
		return money.New(int64(x), c), nil
	case int64:
		return money.New(x, c), nil
	case uint:
		if uint64(x) > math.MaxInt64 {
			return money.Money{}, convert.ErrOverflow
		}
		return money.New(int64(x), c), nil
	case uint8:
		return money.New(int64(x), c), nil
	case uint16:
		return money.New(int64(x), c), nil
	case uint32:
		return money.New(int64(x), c), nil
	case uint64:
		if x > math.MaxInt64 {
			return money.Money{}, convert.ErrOverflow
		}
		return money.New(int64(x), c), nil
	case uintptr:
		if uint64(x) > math.MaxInt64 {
			return money.Money{}, convert.ErrOverflow
		}
		return money.New(int64(x), c), nil
	default:
		s, err := convert.ToString(amount)
		if err != nil {
			return money.Money{}, err
		}
		return money.Parse(cleanAmount(s), c)
	}
}

// NewFromMinor creates Money from minor units such as cents.
func NewFromMinor(minor any, code string) (money.Money, error) {
	c, err := currency(code)
	if err != nil {
		return money.Money{}, err
	}
	n, err := convert.ToInt64(minor)
	if err != nil {
		return money.Money{}, err
	}
	return money.NewFromMinor(n, c), nil
}

func AmountString(m money.Money) string { return appendMajorString(nil, m) }
func Currency(m money.Money) string     { return m.Currency().Code }
func Minor(m money.Money) int64         { return m.Minor() }
func Float64(m money.Money) float64     { return m.Float64() }
func ToString(m money.Money) string {
	c := m.Currency()
	if !c.IsValid() {
		return AmountString(m)
	}
	return c.Code + " " + AmountString(m)
}

func parseMoneyString(s, defaultCurrency string) (money.Money, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return money.Money{}, convert.ErrEmpty
	}
	if m, err := money.ParseMoney(s); err == nil {
		return m, nil
	}
	if m, err := money.Parse(s); err == nil {
		return m, nil
	}
	if defaultCurrency == "" {
		return money.Money{}, ErrMissingCurrency
	}
	return NewFromMajor(stripCurrencySymbol(s), defaultCurrency)
}

func moneyFromAnySlice(x []any, defaultCurrency string) (money.Money, error) {
	if len(x) == 0 {
		return money.Money{}, convert.ErrEmpty
	}
	if len(x) == 1 {
		return ToMoney(x[0], defaultCurrency)
	}
	as, aerr := convert.ToString(x[0])
	bs, berr := convert.ToString(x[1])
	if aerr == nil && looksCurrency(as) {
		return NewFromMajor(x[1], as)
	}
	if berr == nil && looksCurrency(bs) {
		return NewFromMajor(x[0], bs)
	}
	return NewFromMajor(x[0], defaultCurrency)
}

func moneyFromMap(m map[string]any, defaultCurrency string) (money.Money, error) {
	code := defaultCurrency
	if c, ok := lookup(m, "currency", "Currency", "code", "Code", "currency_code", "CurrencyCode"); ok {
		cs, err := currencyCode(c)
		if err != nil {
			return money.Money{}, err
		}
		code = cs
	}
	if minor, ok := lookup(m, "minor", "Minor", "minor_units", "MinorUnits", "cents", "Cents"); ok {
		return NewFromMinor(minor, code)
	}
	amount, ok := lookup(m, "amount", "Amount", "value", "Value", "total", "Total")
	if !ok {
		return money.Money{}, convert.ErrInvalid
	}
	return NewFromMajor(amount, code)
}

func moneyFromStruct(rv reflect.Value, defaultCurrency string) (money.Money, error) {
	code := defaultCurrency
	for _, name := range []string{"Currency", "Code", "CurrencyCode"} {
		f := rv.FieldByName(name)
		if f.IsValid() && f.CanInterface() {
			cs, err := currencyCode(f.Interface())
			if err != nil {
				return money.Money{}, err
			}
			code = cs
			break
		}
	}
	for _, name := range []string{"Minor", "MinorUnits", "Cents"} {
		f := rv.FieldByName(name)
		if f.IsValid() && f.CanInterface() {
			return NewFromMinor(f.Interface(), code)
		}
	}
	for _, name := range []string{"Amount", "Value", "Total"} {
		f := rv.FieldByName(name)
		if f.IsValid() && f.CanInterface() {
			return NewFromMajor(f.Interface(), code)
		}
	}
	return money.Money{}, convert.ErrInvalid
}

func currencyCode(v any) (string, error) {
	switch x := v.(type) {
	case money.Currency:
		if !x.IsValid() {
			return "", money.ErrInvalidCurrency
		}
		return x.Code, nil
	case *money.Currency:
		if x == nil || !x.IsValid() {
			return "", money.ErrInvalidCurrency
		}
		return x.Code, nil
	default:
		s, err := convert.ToString(v)
		if err != nil {
			return "", err
		}
		return strings.ToUpper(strings.TrimSpace(s)), nil
	}
}

func currency(code string) (money.Currency, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return money.Currency{}, ErrMissingCurrency
	}
	c, ok := money.GetCurrency(code)
	if !ok || !c.IsValid() {
		return money.Currency{}, money.ErrInvalidCurrency
	}
	return c, nil
}

func lookup(m map[string]any, names ...string) (any, bool) {
	for _, n := range names {
		if v, ok := m[n]; ok {
			return v, true
		}
	}
	for k, v := range m {
		for _, n := range names {
			if strings.EqualFold(k, n) {
				return v, true
			}
		}
	}
	return nil, false
}

func looksCurrency(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) != 3 {
		return false
	}
	for i := 0; i < 3; i++ {
		c := s[i]
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')) {
			return false
		}
	}
	_, ok := money.GetCurrency(s)
	return ok
}

func stripCurrencySymbol(s string) string { return cleanAmount(s) }
func cleanAmount(s string) string {
	s = strings.TrimSpace(s)
	repl := strings.NewReplacer(",", "", "_", "", "$", "", "€", "", "£", "", "₨", "", "₹", "", "¥", "", "रु", "")
	return strings.TrimSpace(repl.Replace(s))
}
func isNumeric(v any) bool {
	switch v.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, uintptr, float32, float64:
		return true
	}
	return false
}

func appendMajorString(dst []byte, m money.Money) string {
	c := m.Currency()
	decimals := int(c.Decimals)
	minor := m.Minor()
	if decimals <= 0 {
		return string(strconv.AppendInt(dst, minor, 10))
	}
	neg := minor < 0
	if neg {
		minor = -minor
	}
	pow := int64(1)
	for i := 0; i < decimals; i++ {
		pow *= 10
	}
	whole := minor / pow
	frac := minor % pow
	if neg {
		dst = append(dst, '-')
	}
	dst = strconv.AppendInt(dst, whole, 10)
	dst = append(dst, '.')
	var scratch [20]byte
	i := len(scratch)
	if frac == 0 {
		i--
		scratch[i] = '0'
	} else {
		for frac > 0 {
			i--
			scratch[i] = byte('0' + frac%10)
			frac /= 10
		}
	}
	digits := len(scratch) - i
	for ; digits < decimals; digits++ {
		dst = append(dst, '0')
	}
	dst = append(dst, scratch[i:]...)
	return string(dst)
}
