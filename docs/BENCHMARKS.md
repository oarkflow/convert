# Benchmarks

Run:

```bash
go test . -bench=. -benchmem
```

The benchmark suite covers:

- bool/string/bytes/int/uint/float typed conversions
- all signed/unsigned widths
- generic `To[T]`
- target-based `As`, `AsTo`, `AsInto`
- `time.Time` and `time.Duration`
- defaults/must helpers
- named scalar aliases
- append formatting
- slice conversion, including `[]any`

Notes:

- Scalar parse paths are allocation-conscious and typically 0 B/op.
- `ToString(int)` and `ToString(float64)` return a newly allocated Go string. Use `AppendString` for zero-allocation formatting.
- `As(src, target) any` may allocate because it returns a boxed interface. Use `AsInto(src, &target)` for hot paths.
- Reflection-based struct/slice convenience APIs are production helpers. Use `convertgen` for zero-reflection struct hot paths.

## Enterprise feature benchmarks

The benchmark suite now includes production feature groups:

```bash
go test . -bench='BenchmarkProduction' -benchmem
```

Covered groups:

- `ToDeepSlice[int]` from `[]any`
- `ToDeepSlice[Struct]` from `[]any` containing maps
- `ToTypedMap[string, []int]`
- `PopulateWithOptions`
- `ConvertInto`
- `BindQuery`
- `BindHeader`
- extra validators such as slug and semver
- locale-aware float parsing
- typed decimal adapter conversion

Reflection-heavy deep struct/map conversion intentionally trades speed for ergonomics. Use `convertgen` for zero-reflection generated conversion in hot paths.

## Oarkflow money adapter benchmarks

Run:

```bash
go test ./adapters/oarkflowmoney -bench=. -benchmem
```

The adapter is intentionally outside the scalar hot path. It parses flexible money inputs such as strings, maps, structs and `[]any` pairs into `github.com/oarkflow/money.Money` through the custom converter registry and conversion graph.
