# goconvert

Reflection-free, allocation-conscious conversion helpers for Go.

This module keeps the public API clean and optimized through the existing conversion functions. There are no separate public parser APIs. Hot-path improvements are built directly into `ToInt`, `ToInt64`, `ToUint64`, `ToBool`, `To[T]`, and `AsTo`.

## Features

- `convert.To[T](val any) (T, error)`
- `convert.As(src, target) (any, error)`
- `convert.AsTo(src, target)`
- `convert.AsInto(src, &target)` for no-boxing target-based hot paths
- `convert.ToBool`, `ToInt`, `ToUint64`, `ToFloat64`, `ToString`, etc.
- optimized decimal string parsing inside the existing integer helpers
- optimized ASCII bool parsing inside `ToBool`
- `time.Time` support
- `time.Duration` support
- target-based conversion with value, pointer and no-boxing pointer-write targets
- named scalar alias helpers like `ToIntLike[Port]`
- overflow checks
- invalid-value checks
- no `reflect`
- no `fmt` in the package
- append-based formatting via `AppendString`
- zero allocations on successful primitive parse paths

## Install

```bash
go mod init your/module
cp -R convert your/project/
```

Or use the module directly after replacing the module path.

## Basic usage

```go
port, err := convert.To[int]("8080")
debug, err := convert.ToBool("on")
ratio, err := convert.To[float64]([]byte("1.25"))
```

## Optimized existing functions

Use the normal APIs. The optimized decimal and bool parsers are internal implementation details:

```go
n, err := convert.ToInt64("123456")
ok, err := convert.ToBool("true")
```

The integer helpers use a hand-written decimal path for common base-10 values and fall back to `strconv` for base-0 syntax such as `0x10` and `010`.

Representative benchmark from this environment:

```text
BenchmarkStdlibParseInt-56                         13.44 ns/op   0 B/op   0 allocs/op
BenchmarkTypedIntegerConversions/ToInt/string-56     7.19 ns/op   0 B/op   0 allocs/op
BenchmarkGenericToConversions/To[int]/string-56      7.76 ns/op   0 B/op   0 allocs/op
BenchmarkTargetBasedConversions/AsTo/int_from_string 7.59 ns/op   0 B/op   0 allocs/op
BenchmarkTypedFloatConversions/ToFloat64/string-56   6.62 ns/op   0 B/op   0 allocs/op
BenchmarkToBool/string_true-56                       4.11 ns/op   0 B/op   0 allocs/op
BenchmarkToDuration/string-56                        4.44 ns/op   0 B/op   0 allocs/op
BenchmarkAppendString/float64-56                     7.94 ns/op   0 B/op   0 allocs/op
BenchmarkDefault/int_fallback-56                     7.41 ns/op   0 B/op   0 allocs/op
```

The existing APIs now use internal hot paths directly. `ToInt`, `ToInt64`, `ToBool`, `To[int]`, `AsTo`, `ToDuration`, and `AppendString` can all run with zero allocations and single-digit ns/op on common scalar inputs.

## Time and duration

```go
d, err := convert.To[time.Duration]("1h30m")
d2, err := convert.ToDurationSeconds(2.5) // 2.5s

t, err := convert.To[time.Time]("2026-06-27T12:30:00Z")
t2, err := convert.ToTime(1700000000) // Unix seconds
```

`ToDuration` treats numeric values as nanoseconds because that is the native unit of `time.Duration`.
Use explicit helpers when the numeric value means another unit:

```go
convert.ToDurationSeconds(5)
convert.ToDurationMilliseconds(500)
convert.ToDurationUnit(2, convert.Hour)
```

## Convert based on another value

```go
var a = 1.2
var b = "5"

x, err := convert.As(a, b)
// x is string("1.2")
```

Pointer targets are supported too:

```go
var n int
x, err := convert.As("42", &n)
// x is int(42)
```

For hot paths, avoid the `any` result box and write into the target directly:

```go
var n int
err := convert.AsInto("42", &n)
// n == 42
```

Generic form:

```go
x, err := convert.AsTo("5s", time.Duration(0))
// x is time.Duration(5 * time.Second)
```

## Named scalar aliases

Go cannot generically convert into all named scalar aliases with one reflection-free `To[T]` function. This package provides explicit helpers that preserve performance and avoid reflection:

```go
type Port int
p, err := convert.ToIntLike[Port]("8080")
```

Available alias helpers:

- `ToBoolLike[T ~bool]`
- `ToStringLike[T ~string]`
- `ToIntLike[T ~int]`
- `ToInt8Like[T ~int8]`
- `ToInt16Like[T ~int16]`
- `ToInt32Like[T ~int32]`
- `ToInt64Like[T ~int64]`
- `ToUintLike[T ~uint]`
- `ToUint8Like[T ~uint8]`
- `ToUint16Like[T ~uint16]`
- `ToUint32Like[T ~uint32]`
- `ToUint64Like[T ~uint64]`
- `ToUintptrLike[T ~uintptr]`
- `ToFloat32Like[T ~float32]`
- `ToFloat64Like[T ~float64]`

## Allocation-conscious formatting

`ToString(123)` and `ToString(12.5)` return new strings, so Go must allocate for numeric/float formatting. For no intermediate string, use `AppendString`:

```go
buf := make([]byte, 0, 32)
buf, err := convert.AppendString(buf, 123456)
```

## Essential next features for a complete conversion platform

The core package now covers the main scalar, time and duration cases. Essential next features are listed in `ESSENTIAL_FEATURES.md` and include configurable conversion policy, loose/strict modes, locale-aware number parsing, slice/map conversion, struct population, enum registration, nullable types, binary units, decimal-safe conversion, custom converter registry, and path-based conversion helpers.

## Run tests

```bash
go test ./...
```

## Run benchmarks

```bash
go test ./convert -bench=. -benchmem
```

The benchmark suite now covers the full conversion matrix: direct typed conversions, string and []byte parsing, numeric-to-numeric, bool, time, duration, generic `To[T]`, `AsTo`, `As`, named aliases, default/must helpers, and append formatting. See `BENCHMARKS.md` for the full list.

## Examples

```bash
go run ./examples/basic
go run ./examples/time_duration
go run ./examples/as_target
```

## Hot-path guidance

For parsing and numeric conversion, use `ToInt`, `ToFloat64`, `ToBool`, `ToDuration`, `ToTime`, `To[T]`, or `AsTo[T]`.

For formatting to text in hot paths, prefer:

```go
buf := make([]byte, 0, 32)
buf, err := convert.AppendString(buf, value)
```

Returning a new `string` from a numeric value cannot be zero-allocation in Go because strings are immutable and need backing storage. `AppendString` is the zero-allocation API for that path.

For target-based conversion in hot paths, prefer:

```go
var out int
err := convert.AsInto("123", &out)
```

`As(src, target) any` is convenient but can box the result into an interface.
