package convert

import (
	"net/http"
	"net/url"
	"testing"
)

type benchNestedItem struct {
	ID   int      `json:"id"`
	Tags []string `json:"tags"`
}
type benchNestedConfig struct {
	Port   int               `query:"port" json:"port" default:"8080"`
	Host   string            `query:"host" json:"host"`
	Items  []benchNestedItem `json:"items"`
	Lookup map[string][]int  `json:"lookup"`
}

var benchAnySlice = []any{"1", 2, uint64(3), "4", int8(5)}
var benchStructMap = map[string]any{"port": "8080", "host": "localhost", "items": []any{map[string]any{"id": "1", "tags": "a,b"}}, "lookup": map[string]any{"ports": []any{"80", 443, 8080}}}

func BenchmarkProductionSliceConversion(b *testing.B) {
	b.ReportAllocs()
	b.Run("ToDeepSlice_int_from_any", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			x, err := ToDeepSlice[int](benchAnySlice)
			if err != nil || len(x) != 5 {
				b.Fatal(x, err)
			}
		}
	})
	b.Run("ToDeepSlice_struct_from_any", func(b *testing.B) {
		in := []any{map[string]any{"id": "1", "tags": "a,b"}, map[string]any{"id": 2, "tags": []any{"x", "y"}}}
		for i := 0; i < b.N; i++ {
			x, err := ToDeepSlice[benchNestedItem](in)
			if err != nil || len(x) != 2 {
				b.Fatal(x, err)
			}
		}
	})
	b.Run("ToTypedMap_string_slice_int", func(b *testing.B) {
		in := map[string]any{"a": []any{"1", 2, 3}, "b": "4,5,6"}
		for i := 0; i < b.N; i++ {
			x, err := ToTypedMap[string, []int](in)
			if err != nil || len(x["a"]) != 3 {
				b.Fatal(x, err)
			}
		}
	})
}

func BenchmarkProductionStructBinding(b *testing.B) {
	b.ReportAllocs()
	b.Run("PopulateWithOptions", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var cfg benchNestedConfig
			if err := PopulateWithOptions(&cfg, benchStructMap); err != nil || cfg.Port != 8080 {
				b.Fatal(cfg, err)
			}
		}
	})
	b.Run("ConvertInto", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var cfg benchNestedConfig
			if err := ConvertInto(&cfg, benchStructMap); err != nil || cfg.Lookup["ports"][1] != 443 {
				b.Fatal(cfg, err)
			}
		}
	})
	b.Run("BindQuery", func(b *testing.B) {
		vals := url.Values{"port": {"8080"}, "host": {"localhost"}}
		for i := 0; i < b.N; i++ {
			var cfg benchNestedConfig
			if err := BindQuery(&cfg, vals); err != nil || cfg.Port != 8080 {
				b.Fatal(cfg, err)
			}
		}
	})
	b.Run("BindHeader", func(b *testing.B) {
		h := http.Header{"Port": {"8080"}, "Host": {"localhost"}}
		for i := 0; i < b.N; i++ {
			var cfg benchNestedConfig
			if err := BindHeader(&cfg, h); err != nil || cfg.Host != "localhost" {
				b.Fatal(cfg, err)
			}
		}
	})
}

func BenchmarkProductionValidationLocaleDecimal(b *testing.B) {
	b.ReportAllocs()
	b.Run("ValidateSlug", func(b *testing.B) {
		v := Slug()
		for i := 0; i < b.N; i++ {
			if err := v("release-2026"); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("ValidateSemVer", func(b *testing.B) {
		v := SemVer()
		for i := 0; i < b.N; i++ {
			if err := v("1.2.3-beta+1"); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("LocaleFloatEU", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			x, err := ParseLocaleFloat64("1.234,56", LocaleEU)
			if err != nil || x != 1234.56 {
				b.Fatal(x, err)
			}
		}
	})
	b.Run("DecimalAdapter", func(b *testing.B) {
		RegisterDecimalAdapter[testDecimal](DecimalAdapter[testDecimal]{Parse: func(s string) (testDecimal, error) { return testDecimal(s), nil }, Format: func(d testDecimal) string { return string(d) }})
		for i := 0; i < b.N; i++ {
			x, err := ToDecimalTyped[testDecimal]("12.34")
			if err != nil || x != "12.34" {
				b.Fatal(x, err)
			}
		}
	})
}
