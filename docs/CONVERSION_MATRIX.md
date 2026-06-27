# Conversion Matrix

| from \\ to     | bool | int  | uint | float | string | duration | time | bytes | slice |
| ------------- | ---- | ---- | ---- | ----- | ------ | -------- | ---- | ----- | ----- |
| string        | yes  | yes  | yes  | yes   | yes    | yes      | yes  | yes   | yes   |
| []byte        | yes  | yes  | yes  | yes   | yes    | yes      | yes  | yes   | yes   |
| int           | yes  | yes  | yes  | yes   | yes    | yes      | yes  | yes   | no    |
| uint          | yes  | yes  | yes  | yes   | yes    | yes      | yes  | yes   | no    |
| float         | yes  | yes* | yes* | yes   | yes    | yes      | yes  | yes   | no    |
| bool          | yes  | yes  | yes  | yes   | yes    | no       | no   | yes   | no    |
| time.Time     | no   | yes  | yes  | no    | yes    | no       | yes  | no    | no    |
| time.Duration | no   | yes  | yes  | no    | yes    | yes      | no   | yes   | no    |
| []any/slice   | no   | no   | no   | no    | no     | no       | no   | no    | yes   |

`yes*` means controlled by precision policy. Use `NoPrecisionLoss`, `AllowFloatToIntExactOnly`, or `AllowFloatToIntTruncate` explicitly for critical paths.

## Deep conversion additions

| Source | Target | Supported |
| --- | --- | --- |
| `[]any` | `[]T` | yes, via `ToDeepSlice`, `ToTypedSlice`, `ConvertInto` |
| `[]map[string]any` | `[]Struct` | yes |
| `map[string]any` | `map[string][]T` | yes, via `ToTypedMap` / `ConvertInto` |
| `map[string]any` | `Struct` | yes, via `PopulateWithOptions` / `ConvertInto` |
| localized number string | `int64`, `float64` | yes, via locale-aware parsing |
| decimal string | custom decimal type | yes, via typed decimal adapter |

## Oarkflow money adapter

Optional package: `github.com/oarkflow/convert/adapters/oarkflowmoney`.

| From | To `money.Money` | Notes |
|---|---:|---|
| `money.Money` | yes | preserves currency, fills default when empty |
| `*money.Money` | yes | nil checked |
| `string` | yes | supports `"USD 12.50"`, `"12.50 USD"`, and amount-only with default currency |
| `[]byte` | yes | parsed as string |
| numeric scalars | yes | interpreted as amount with default currency |
| `map[string]any` | yes | reads `amount`/`currency`, also `value`, `total`, `code` |
| structs | yes | reads `Amount`/`Currency`, also `Value`, `Total`, `Code` |
| `[]any` | yes | supports `["USD", "12.50"]` and `["12.50", "USD"]` |
