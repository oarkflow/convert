# convert

Production-ready Go conversion package with allocation-conscious scalar hot paths and comprehensive higher-level conversion helpers.

## Core API

```go
n, err := convert.To[int]("42")
b, err := convert.ToBool("true")
d, err := convert.To[time.Duration]("2s")
t, err := convert.To[time.Time]("1710000000")

var out int
err = convert.AsInto("42", &out) // no interface result boxing
x, err := convert.As("42", 0)    // ergonomic target-based conversion
```

## Slice conversion, including []any

```go
ints, err := convert.ToSlice[int]([]any{"1", 2, int64(3)})
parts, err := convert.ToAnySlice("api, edge, prod", convert.WithTrimSpace())
custom, err := convert.ToTypedSlice[MyType]([]any{map[string]any{"id":"1"}})
```

Slice helpers support strings, `[]byte`, `[]any`, typed slices, arrays, trimming, ignore-empty and unique filtering.

## Policies

```go
n, err := convert.ToIntWith(1.0, convert.NoPrecisionLoss())
n, err = convert.ToIntWith(1.8, convert.AllowFloatToIntTruncate())
x, err := convert.ToWith[int](" 42 ", convert.TrimSpace())
```

## Structs, maps, query, form, header and CSV binding

```go
type Config struct {
    Port    int           `convert:"port" query:"port" default:"8080"`
    Debug   bool          `convert:"debug"`
    Tags    []string      `convert:"tags" query:"tag"`
    Timeout time.Duration `convert:"timeout" default:"5s"`
}

cfg, err := convert.ToStruct[Config](map[string]any{"port":"9000"})
err = convert.BindQuery(&cfg, values)
err = convert.BindHeader(&cfg, header)
err = convert.BindCSVRow(&cfg, headers, row)
```

## Code generation

`cmd/convertgen` generates no-reflection struct converters for hot paths.

```go
//go:generate go run github.com/oarkflow/convert/cmd/convertgen -type Config
```

Generated shape:

```go
cfg, err := ConvertConfig(map[string]any{"port":"9000"})
err = PopulateConfig(&cfg, data)
```

## Custom converters and graph conversion

```go
type UserID uint64
convert.Register(func(v any, ctx convert.Context) (UserID, error) {
    u, err := convert.ToUint64(v)
    return UserID(u), err
})

convert.RegisterEdge[string, uint64](convert.ToUint64)
convert.RegisterEdge[uint64, UserID](func(u uint64) (UserID, error) { return UserID(u), nil })
id, err := convert.ConvertGraph[UserID]("42")
```

## Other features

- nullable/option helpers
- SQL `Null*`, `Scanner`, `driver.Valuer` helpers
- enum registry
- validation hooks
- hex/base64/endian helpers
- size parsing: `64MB`, `64MiB`, etc.
- decimal parser adapter point
- URL/IP/net/netip helpers
- UUID validation
- path-aware `Get`/`Set`
- env/config helpers
- rich path-aware errors
- fuzz and differential tests
- conversion matrix documentation

## Examples

Run focused examples:

```bash
go run ./examples/slices
go run ./examples/config
go run ./examples/http_query
go run ./examples/sql_scan
go run ./examples/struct_binding
go run ./examples/custom_converter
go run ./examples/enum
go run ./examples/validation
go run ./examples/bytes_binary
go run ./examples/network
go run ./examples/generated
```

## Validation

```bash
go test ./...
go test ./convert -bench=. -benchmem
```

Scalar hot paths remain separate from reflection-based convenience APIs. Returning a newly formatted Go `string` from numeric values necessarily allocates; use `AppendString` for no-allocation formatting.

## Production hardening added

The package now includes the production features from the remaining roadmap items 3-8:

- Deep slice conversion with `ToDeepSlice[T]`, `ToSliceAny[T]`, `ToTypedSlice[T]`, and `ConvertInto`.
- Full `[]any` support, including nested values such as `[]any{map[string]any{...}} -> []Struct`.
- Deep map conversion with `ToTypedMap[K,V]`, including `map[string]any -> map[string][]int` and map-to-struct values.
- Path-aware errors for nested conversion failures, for example `[0].id` or `lookup.ports[2]`.
- Expanded struct tag system using `convert`, `json`, `env`, `query`, `form`, `header`, `csv`, `default`, `required`, and `validate`.
- Additional validators: slug, semver, domain, FQDN, MAC address, phone, country code, currency code, timezone, safe file path, charset, and reserved usernames.
- Locale-aware numeric parsing with `ParseLocaleFloat64`, `ParseLocaleInt64`, `ToFloat64Locale`, and `ToInt64Locale`.
- Dependency-free typed decimal adapter registry with `RegisterDecimalAdapter`, `ToDecimalTyped`, and `DecimalToString`.
- Enterprise benchmark groups covering deep slices, deep maps, struct binding, query/header binding, validators, locale parsing, and decimal adapters.

Example:

```go
var cfg Config
err := convert.PopulateWithOptions(&cfg, map[string]any{
    "host": "api.example.com",
    "PORT": "9090",
    "items": []any{
        map[string]any{"id": "1", "tags": "api,edge"},
        map[string]any{"id": 2, "tags": []any{"prod", "blue"}},
    },
    "lookup": map[string]any{"ports": []any{"80", 443, uint64(8080)}},
})
```

For a complete runnable example, see `examples/production_features`.

## github.com/oarkflow/money adapter

The package includes an optional adapter at `adapters/oarkflowmoney` for the real `github.com/oarkflow/money` API. That library stores money as integer minor units with unexported fields and exposes construction/access through `money.New`, `money.NewFromMinor`, `money.NewFromFloat`, `money.Parse`, `money.ParseMoney`, `Money.Minor`, and `Money.Currency`. Register the adapter when you need domain money conversion:

```go
import (
    "github.com/oarkflow/convert"
    "github.com/oarkflow/convert/adapters/oarkflowmoney"
    "github.com/oarkflow/money"
)

func example() {
    oarkflowmoney.Register("USD")
    price, _ := convert.Convert[money.Money]("12.50")
    _ = price
}
```

Supported money inputs include strings (`"USD 12.50"`, `"12.50 USD"`, `"US$12.50"`), numbers with a default currency, maps with `amount`/`currency` or `minor`/`currency`, structs with `Amount`/`Currency` or `Minor`/`Currency`, and `[]any` pairs. Numeric scalar inputs are treated as major units; use `oarkflowmoney.NewFromMinor` or a `minor` field when you have cents/minor units.

## High-performance DTO conversion

The `DTO` API is the production data-to-object layer for deep conversion across structs, maps, slices, arrays, pointers and scalar fields. It caches struct field metadata, honors `convert`, `json`, `env`, `query`, `form`, `header` and `csv` tags, supports defaults, required/validate tags, case-insensitive field matching, string-to-slice splitting, struct-to-map conversion, custom decode hooks and path-aware errors.

```go
type Address struct {
    City string `json:"city"`
    Zip  int    `json:"zip"`
}

type UserDTO struct {
    ID      int            `json:"id" validate:"required"`
    Active  bool           `json:"active" default:"true"`
    Tags    []string       `json:"tags"`
    Address Address        `json:"address"`
    Limits  map[string]int `json:"limits"`
    TTL     time.Duration  `json:"ttl" default:"30s"`
}

user, err := convert.DTOTo[UserDTO](map[string]any{
    "id": "1001",
    "tags": "smpp,edge,production",
    "address": map[string]any{"city":"Kathmandu", "zip":"44600"},
    "limits": map[any]any{"tps":"500", "burst":1000},
})

var dst UserDTO
err = convert.DTO(&dst, input, convert.WithDTOErrorUnused())
items, err := convert.DTOSlice[UserDTO]([]any{input})
m, err := convert.DTOMap[string, any](dst)
```

For hot paths that need domain-specific behavior, add a decode hook:

```go
err := convert.DTO(&dst, input, convert.WithDTODecodeHook(func(ctx convert.DTOContext, dst reflect.Value, src any) (bool, error) {
    // set dst and return true to bypass the built-in converter
    return false, nil
}))
```

## Advanced DTO conversion features

The DTO layer now includes a production feature set for struct/map/slice/object mapping:

- Cached field metadata and reusable `DTOPlan[S,T]` conversion facades.
- Type-pair fast converter registry: `RegisterPair[S,T]`, `DTOConvertPair[S,T]`.
- Strict DTO mode: `WithDTOStrict()`, `WithDTOErrorUnused()`, `WithDTOLosslessNumbers()`.
- Resource limits: `WithDTOMaxDepth`, `WithDTOMaxSliceLen`, `WithDTOMaxMapSize`.
- Flatten/unflatten support for dotted paths: `WithDTOFlatten()`, `FlattenMap`, `UnflattenMap`, `StructToFlatMap`.
- Field aliases from tag options: `source=a|b|c` and `alias=a|b|c`.
- Field transforms: `trim`, `lower`, `upper`, `title`, `snake`, `camel`, `kebab`.
- Field security flags: `readonly`, `writeonly`, `sensitive`, `secret`.
- Redaction for logging: `Redact`.
- Warning/report mode: `DTOToReport`, `DTOReport`.
- Batch conversion: `DTOBatch`, `DTOBatchConvert`, error collection, skip invalid, and parallel mode.
- Patch/update support: `ApplyPatch` returns changed paths and ignores absent fields.
- Struct diff and merge: `Diff`, `SortedDiff`, `Merge`.
- Schema generation: `SchemaOf[T]`, `SchemaFor`, `StableJSONSchema[T]`.
- Source-specific decoders: `FromQuery`, `FromForm`, `FromHeaders`, `FromEnv`, `FromCSVRow`.
- Object encoders: `ToQuery`, `ToHeaders`, `ToEnv`.
- JSONL streaming conversion: `DTOStreamJSONL[T]`.
- Plugin installation hook: `Use(plugin...)`.
- Profiles: `DTOProfileAPI`, `DTOProfileDB`, `DTOProfileConfig`, `DTOProfileForm`, `DTOProfileCSV`, `DTOProfileStrict`.

Example:

```go
package main

import (
    "fmt"
    "net/url"

    "github.com/oarkflow/convert"
)

type UserDTO struct {
    ID       int    `json:"id,readonly"`
    Name     string `json:"name,trim,title" validate:"required"`
    Email    string `json:"email,trim,lower"`
    City     string `json:"address.city"`
    APIKey   string `json:"api_key,sensitive"`
    Password string `json:"password,writeonly,sensitive"`
}

func main() {
    input := map[string]any{
        "id":       42,
        "name":     "  sujit kumar ",
        "email":    " ADMIN@EXAMPLE.COM ",
        "address":  map[string]any{"city": "Kathmandu"},
        "api_key":  "secret-key",
        "password": "secret-password",
    }

    user, err := convert.DTOTo[UserDTO](input, convert.WithDTOFlatten())
    if err != nil {
        panic(err)
    }
    fmt.Printf("%+v\n", user)

    safe, _ := convert.Redact(user)
    fmt.Printf("%+v\n", safe)

    q := url.Values{"name": {"sujit"}, "address.city": {"Kathmandu"}}
    fromQuery, _ := convert.FromQuery[UserDTO](q, convert.WithDTOFlatten())
    fmt.Printf("%+v\n", fromQuery)
}
```

Patch/update example:

```go
changed, err := convert.ApplyPatch(&user, map[string]any{
    "name": "Updated Name",
})
// changed == []string{"name"}
```

Batch/report example:

```go
report := convert.DTOBatchConvert[UserDTO](rows,
    convert.DTOBatchCollectErrors(),
    convert.DTOBatchParallel(8),
    convert.DTOBatchWithOptions(convert.WithDTOFlatten()),
)
```

Schema example:

```go
schema := convert.SchemaOf[UserDTO]()
fmt.Println(convert.StableJSONSchema[UserDTO]())
```

## Usability APIs

The package now includes a high-level usability layer on top of the fast DTO converter.

### One-call mapping

```go
user, err := convert.Map[User](input)
err := convert.MapInto(input, &user)
user := convert.MustMap[User](input)
```

Presets are available for common modes:

```go
convert.Map[User](input, convert.API())
convert.Map[User](input, convert.StrictAPI())
convert.Map[Config](input, convert.Config())
convert.Map[Search](input, convert.QueryDTO())
convert.Map[HeaderDTO](input, convert.Header())
```

`DTOLoose` and `DTOStrictPreset` are named to avoid conflicts with the existing scalar `Loose()` and `Strict()` policy helpers.

### Multi-source binding

```go
req, err := convert.Bind[SearchRequest](
    convert.FromMapSource("body", body, "json", "convert"),
    convert.FromQuerySource(query),
    convert.FromHeaderSource(headers),
    convert.FromDefaultSource(map[string]any{"limit": 25}),
)
```

Later sources overwrite earlier keys. Use `BindWithOptions` for strict, flattening, or source-priority options.

### HTTP helpers

```go
err := convert.BindJSON(r, &dto)
err := convert.BindRequestQuery(r, &dto)
err := convert.BindRequestForm(r, &dto)
err := convert.BindHeaders(r, &dto)
err := convert.BindRequest(r, &dto)
```

The HTTP helpers are optional convenience functions built on `net/http`; the core scalar converter remains dependency-light.

### Error reporting

```go
out, err := convert.MapAll[User](input)
fmt.Println(convert.HumanError(err))
fmt.Println(convert.DebugError(err))
```

`MapAll` collects field errors where possible and returns `convert.MultiError`.

### Trace and dry-run

```go
trace := convert.MapTrace[User](input)
report := convert.DryRun[User](input)
```

Trace steps show whether fields were mapped, defaulted, skipped, readonly-skipped, or missing.

### Three-state patch values

```go
type UserPatch struct {
    Name  convert.Optional[string] `json:"name"`
    Email convert.Optional[string] `json:"email"`
}

changed, err := convert.ApplyOptionalPatch(&user, UserPatch{
    Name:  convert.Some("New Name"),
    Email: convert.Null[string](),
})
```

`Optional[T]` supports unset, explicit null/clear, and value states.

### Safe logging and redaction

```go
safeJSON := convert.SafeJSON(user)
safeMap := convert.SafeMap(user)
safeText := convert.SafeString(user)
```

Fields tagged `sensitive`, `secret`, or `sensitive:"true"` are redacted.

### Config loader

```go
cfg, err := convert.LoadConfig[Config](
    convert.FileSource("config.json"),
    convert.EnvSource(),
    convert.MapSource(map[string]any{"debug": true}),
)
```

Sources are merged in order. Later sources override earlier sources.

### SQL helpers

```go
user, err := convert.FromSQLRow[User](row, []string{"id", "name", "email"})
users, err := convert.FromSQLRows[User](rows)
args, err := convert.ToSQLArgs(user, "id", "name", "email")
```

The helpers use small interfaces compatible with `database/sql` rows.

### CSV helpers

```go
users, err := convert.ReadCSV[User](reader)
err := convert.WriteCSV(writer, users)
```

CSV headers are matched using `csv`, `json`, and `convert` tags.

### Collections

```go
byID, err := convert.SliceToMap[User, int](users, "id")
groups, err := convert.GroupBy[User, string](users, "role")
emails, err := convert.Pluck[User, string](users, "email")
```

### Runtime diagnostics and warmup

```go
stats := convert.Stats()
err := convert.Warmup[User](map[string]any{"id": 1})
err := convert.WarmupPair[Source, Target]()
```

### CLI

A lightweight CLI entry point is available:

```bash
convert schema
convert inspect
convert validate -file payload.json
```

Typed schemas are generated in Go with `convert.StableJSONSchema[T]()` and field docs with `convert.Describe[T]()`.

### Runnable example

```bash
go run ./examples/usability
```
