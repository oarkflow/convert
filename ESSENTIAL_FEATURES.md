# Essential Conversion Features Status

This package now includes the production features identified for a comprehensive conversion platform:

- scalar conversion with fast hot paths
- `time.Time` and `time.Duration`
- target-based `As`, `AsTo`, `AsInto`
- strict/loose policy system
- precision-loss policy helpers
- nullable, pointer, and option helpers
- SQL `Null*`, `ScanTo`, and `driver.Value` helpers
- production slice conversion including `[]any`, string splitting, arrays, typed slices, uniqueness, trimming, empty filtering, and struct-field binding
- map conversion
- struct population with tags and defaults
- custom converter registry
- conversion graph for chained domain conversion
- validation hooks
- enum registry
- bytes, hex, base64, binary endian helpers
- size unit parser and formatter
- decimal parser adapter point
- URL/IP/CIDR/netip helpers
- UUID validation
- advanced time helpers and Unix unit helpers
- path-aware get/set and path errors for binding
- query, form, header, and CSV binding
- JSON number normalization helpers
- kind and can-convert helpers
- user-defined converter interfaces
- env/config helpers
- unsafe byte-string audit helpers
- conversion matrix document
- fuzz tests and differential stdlib tests
- `cmd/convertgen` code generator
- focused examples for all major feature groups

Remaining optional future work:

- external adapter submodules for specific decimal packages
- deeper multi-hop shortest-path graph search beyond one-hop chaining
- generated code for slices/maps with no reflection at all
- CI workflow files for fuzz/nightly compatibility testing
