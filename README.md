# goconvert

Production-grade Go conversion package with fast scalar hot paths and higher-level conversion helpers for config, HTTP, SQL, JSON-like data, slices, maps, structs, validation, enums, network values, size units, custom converters, and code generation.

## Core APIs

```go
v, err := convert.To[int]("42")
b, err := convert.ToBool("yes")
d, err := convert.To[time.Duration]("1s")
t, err := convert.To[time.Time]("1710000000")
```

Target-based conversion:

```go
var target int
err := convert.AsInto("42", &target) // no interface result boxing

x, err := convert.As("42", target)   // ergonomic any return
```

## Slice conversion, including []any

```go
ints, err := convert.ToSlice[int]([]any{"1", 2, int64(3)})
vals, err := convert.ToAnySlice("a,b,c")
durs, err := convert.ToDurationSlice([]any{"1s", "2s"})
unique, err := convert.ToSlice[int]("1,1,2", convert.WithUnique())
```

Supported slice inputs:

- `string`, split by separator
- `[]byte`
- `[]any`
- any slice or array via reflection
- query/form/header multi-values
- CSV rows
- struct fields

Options:

```go
convert.WithSeparator("|")
convert.WithTrimSpace()
convert.WithIgnoreEmpty()
convert.WithUnique()
```

## Policies

```go
x, err := convert.ToWith[int](" 42 ", convert.TrimSpace(), convert.EmptyAsZero())
y, err := convert.ConvertWith[int](1.0, convert.NoPrecisionLoss())
```

Policy helpers include strict/loose mode, whitespace trimming, empty-as-zero/nil, bool policy, duration default units, custom time layouts, safe/unsafe byte-string policy, and precision-loss controls.

## Structs, maps, and binding

```go
type Config struct {
    Port    int           `convert:"port" default:"8080"`
    Debug   bool          `convert:"debug"`
    Timeout time.Duration `convert:"timeout" default:"5s"`
    Tags    []string      `convert:"tags"`
}

cfg, err := convert.ToStruct[Config](map[string]any{
    "debug": "true",
    "tags": []any{"api", "edge"},
})
```

HTTP/query/header/form/CSV binding:

```go
convert.BindQuery(&req, values)
convert.BindHeader(&req, headers)
convert.BindForm(&req, values)
convert.BindCSVRow(&row, headers, values)
```

## Custom converters and graph conversion

```go
type UserID uint64

convert.Register(func(v any, ctx convert.Context) (UserID, error) {
    u, err := convert.ToUint64(v)
    return UserID(u), err
})

id, err := convert.Convert[UserID]("42")
```

Conversion graph supports chained conversions:

```go
convert.RegisterEdge[string, uint64](func(s string) (uint64, error) { return convert.ToUint64(s) })
convert.RegisterEdge[uint64, UserID](func(u uint64) (UserID, error) { return UserID(u), nil })
id, err := convert.ConvertGraph[UserID]("42")
```

## Validation and enums

```go
port, err := convert.ToValidated[int]("8080", convert.Range(1, 65535))
email, err := convert.ToValidated[string]("hello@example.com", convert.Email())
```

Enums:

```go
convert.RegisterEnum(map[string]Level{"debug": Debug, "info": Info})
level, err := convert.EnumValue[Level]("info")
```

## Other production features

- SQL helpers: `ScanTo`, `ToDriverValue`, `ToSQLNull*`
- JSON helpers: `NormalizeJSONNumbers`, `JSONNumberToInt64`, `JSONMapToStruct`
- bytes/binary: hex, base64, endian uint64 helpers
- size units: `10MB`, `64MiB`, `1GB`
- network types: URL, IP, CIDR, `netip.Addr`, `netip.Prefix`, `netip.AddrPort`
- UUID validation
- env/config helpers: `Env`, `EnvRequired`, `EnvSlice`, `EnvStruct`
- path helpers: `Get`, `GetAny`, `Set`
- conversion matrix: `CONVERSION_MATRIX.md`
- fuzz and differential tests
- code generator: `cmd/convertgen`

## Code generation

For zero-reflection struct conversion, generate a binder:

```bash
go run github.com/oarkflow/convert/cmd/convertgen -type Config -input config.go -out convert_config_gen.go
```

The generated code emits direct field assignments through `convert.AsInto`.

## Performance guidance

The scalar hot paths avoid allocations. Returning a new Go `string` from numeric formatting can allocate by language design; use `AppendString` for zero-allocation formatting:

```go
buf := make([]byte, 0, 32)
buf, _ = convert.AppendString(buf, 123456)
```

For target-based conversion in hot paths, prefer `AsInto` over `As` because `As` returns `any` and may box.

## Validation

```bash
go test ./...
go test ./... -run=Fuzz -fuzz=FuzzToInt -fuzztime=5s
go test . -bench=. -benchmem
```

See the focused examples under `examples/`.
