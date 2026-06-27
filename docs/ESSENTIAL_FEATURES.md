# Essential Features Status

Implemented:

- scalar conversion with generic `To[T]`
- target-based `As`, `AsTo`, `AsInto`
- time and duration conversion
- production slice conversion, including `[]any`, strings, bytes, typed slices and arrays
- `ToTypedSlice[T any]` for custom slice element types
- policy system including strict/loose, trimming and precision options
- nullable/option helpers
- SQL helpers
- map and struct conversion
- custom converter registry
- conversion graph with chained one-hop edge conversion
- validators and enum registry
- hex/base64/endian/bytes helpers
- size unit parsing
- decimal adapter point
- URL/IP/netip/UUID helpers
- enhanced time helpers
- path-aware `Get`/`Set` and rich errors
- query/form/header/CSV binders
- env/config helpers
- `convertgen` code generator
- fuzz tests
- stdlib differential tests
- conversion matrix
- focused examples

Performance note: scalar conversions are optimized and allocation-conscious. Complex helpers such as struct binding and `ToTypedSlice[T any]` intentionally use reflection/registry logic unless generated converters are used.

## Implemented from remaining roadmap items 3-8

### 3. Deep slice/map conversion hardening

Implemented `ToDeepSlice[T]`, `ToSliceAny[T]`, `ToTypedMap[K,V]`, and `ConvertInto`. These support nested slices, arrays, `[]any`, maps, structs, pointers, registered converters, and path-aware errors.

### 4. Struct binding tag system

`PopulateWithOptions` now uses a tag stack: `convert`, `json`, `env`, `query`, `form`, `header`, and `csv`, plus `default`, `required`, and `validate` tags.

### 5. Additional validation rules

Added validators for slug, semver, domain, FQDN, MAC address, phone number, country code, currency code, timezone, safe file path, charset, and reserved usernames.

### 6. Locale-aware parsing

Added allocation-free locale-aware parsers for common localized numeric input: `ParseLocaleFloat64`, `ParseLocaleInt64`, `ToFloat64Locale`, and `ToInt64Locale`.

### 7. Decimal adapters

Added dependency-free typed decimal adapter registry so external decimal libraries can be integrated without adding dependencies to the core package.

### 8. Enterprise benchmarks

Added benchmark groups for deep slice conversion, deep map conversion, struct population, HTTP/query/header binding, validators, locale parsing, and decimal adapter conversion.
