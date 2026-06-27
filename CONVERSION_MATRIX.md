# Conversion Matrix

| from/to | bool | int | uint | float | string | duration | time | bytes | slice | map | struct |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| string | yes | yes | yes | yes | yes | yes | yes | yes | yes | no | yes |
| []byte | yes | yes | yes | yes | yes | yes | yes | yes | yes | no | no |
| []any | no | no | no | no | no | no | no | no | yes | no | no |
| int | yes | yes | yes | yes | yes | yes | yes | yes | no | no | no |
| uint | yes | yes | yes | yes | yes | yes | yes | yes | no | no | no |
| float | yes | exact | exact | yes | yes | yes | yes | yes | no | no | no |
| bool | yes | yes | yes | yes | yes | yes | no | yes | no | no | no |
| time.Time | yes | unix | unix | unix | yes | no | yes | yes | no | no | no |
| duration | yes | ns | ns | ns | yes | yes | no | yes | no | no | no |
| map | no | no | no | no | yes | no | no | no | no | yes | yes |
| struct | no | no | no | no | no | no | no | no | no | no | yes |

Notes:

- `[]any` is supported through `ToAnySlice`, `ToSlice[any]`, `ToSlice[T]`, struct fields, query/form/header binding, CSV row binding, and `Convert[[]T]`.
- `exact` means float-to-int conversion rejects fractional values and overflow.
- `As(...) any` is ergonomic but may box values; use `AsInto` for allocation-free target conversion.
- `ToString(number)` returns a new Go string and can allocate; use `AppendString` for zero-allocation formatting.
