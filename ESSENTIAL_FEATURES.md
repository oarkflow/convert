# Essential Features for a Complete Conversion Library

## Implemented in this package

- Generic conversion: `To[T](any) (T, error)`
- Target-based conversion: `As(src, target)` and `AsTo(src, target)`
- Built-in scalar conversions: bool, string, signed ints, unsigned ints, floats
- Optimized parsing inside the existing functions, not as separate public APIs
- `time.Time` conversion
- `time.Duration` conversion
- Explicit duration unit helpers
- Overflow and invalid-value errors
- Named scalar alias helpers
- Append-based string formatting with `AppendString`
- No reflection and no `fmt` in the package
- Zero allocations for successful primitive parse paths

## Essential next features

1. **Configurable conversion policy**
   - strict mode
   - loose mode
   - trim-space option
   - empty-string-as-zero option
   - empty-string-as-nil option
   - bool vocabulary configuration
   - numeric base policy

2. **Nullable and optional values**
   - `sql.NullString`, `sql.NullInt64`, `sql.NullBool`, `sql.NullFloat64`, `sql.NullTime`
   - pointer targets such as `*int`, `*time.Time`
   - custom `Option[T]` style types

3. **Slices and arrays**
   - `ToSlice[T](any) ([]T, error)`
   - comma-separated strings to slices
   - `[]any` to typed slices
   - fixed array conversion with exact length validation

4. **Maps**
   - `ToMap[K,V](any) (map[K]V, error)`
   - `map[string]any` conversion
   - key conversion with strict duplicate detection after conversion

5. **Struct population**
   - map to struct
   - env-style map to struct
   - tag support: `convert`, `json`, `env`
   - required/default tags
   - nested field paths

6. **Enum support**
   - string enum registration
   - numeric enum registration
   - case-insensitive option
   - aliases and deprecation warnings

7. **Custom converter registry**
   - register converter by source/target type
   - scoped registry
   - package-level default registry
   - no-reflection generated registration mode for hot paths

8. **Binary and size units**
   - `1KB`, `1KiB`, `5MB`, `2GiB`
   - configurable decimal/binary unit behavior
   - overflow-safe conversion to int64/uint64

9. **Decimal-safe conversion**
   - avoid float precision loss for money/decimal strings
   - adapters for decimal packages without hard dependencies
   - exact integer conversion from decimal strings when valid

10. **Time policy support**
    - configurable layouts
    - default location
    - Unix seconds/millis/micros/nanos policy
    - date-only policy
    - timezone abbreviation handling

11. **Path-based conversion helpers**
    - get and convert from nested maps
    - `PathTo[T](data, "user.profile.age")`
    - safe default values

12. **Code generation for zero-dispatch conversion**
    - generated typed converters for project structs
    - no `any` dispatch in generated hot paths
    - ideal for config loading and request binding

13. **Validation integration**
    - min/max
    - length
    - regex
    - enum validation
    - required/non-zero

14. **Error context**
    - source path
    - source value type
    - target type
    - wrapped cause
    - batch conversion errors
