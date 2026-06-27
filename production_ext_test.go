package convert

import (
	"errors"
	"testing"
	"time"
)

type deepItem struct {
	ID   int      `json:"id" validate:"required"`
	Tags []string `json:"tags"`
}

type deepConfig struct {
	Port    int                 `env:"PORT" default:"8080"`
	Host    string              `json:"host" required:"true"`
	Items   []deepItem          `json:"items"`
	Lookup  map[string]deepItem `json:"lookup"`
	Timeout time.Duration       `query:"timeout" default:"2s"`
}

func TestDeepSliceFromAny(t *testing.T) {
	in := []any{
		map[string]any{"id": "1", "tags": []any{"api", "edge"}},
		map[string]any{"id": 2, "tags": "prod,blue"},
	}
	got, err := ToDeepSlice[deepItem](in, WithTrimSpace())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != 1 || got[1].Tags[1] != "blue" {
		t.Fatalf("unexpected %#v", got)
	}
}

func TestDeepSlicePathError(t *testing.T) {
	_, err := ToDeepSlice[deepItem]([]any{map[string]any{"id": "bad"}})
	if err == nil {
		t.Fatal("expected error")
	}
	var detail *ErrorDetail
	if !errors.As(err, &detail) {
		t.Fatalf("expected ErrorDetail: %T %v", err, err)
	}
	if detail.Path != "[0].id" {
		t.Fatalf("expected path [0].id, got %q", detail.Path)
	}
}

func TestTypedMapDeep(t *testing.T) {
	got, err := ToTypedMap[string, deepItem](map[any]any{"one": map[string]any{"id": "7", "tags": "a,b"}})
	if err != nil {
		t.Fatal(err)
	}
	if got["one"].ID != 7 || got["one"].Tags[1] != "b" {
		t.Fatalf("unexpected %#v", got)
	}
}

func TestPopulateTagsAndValidation(t *testing.T) {
	var cfg deepConfig
	err := PopulateWithOptions(&cfg, map[string]any{
		"host":   "localhost",
		"PORT":   "9000",
		"items":  []any{map[string]any{"id": "10", "tags": "x,y"}},
		"lookup": map[string]any{"a": map[string]any{"id": "11", "tags": []any{"z"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 9000 || cfg.Host != "localhost" || cfg.Timeout != 2*time.Second || cfg.Items[0].ID != 10 || cfg.Lookup["a"].ID != 11 {
		t.Fatalf("unexpected %#v", cfg)
	}
}

func TestAdditionalValidators(t *testing.T) {
	cases := []struct {
		name, value string
		v           Validator[string]
	}{
		{"slug", "hello-world", Slug()},
		{"semver", "1.2.3-beta+1", SemVer()},
		{"domain", "example.com", Domain()},
		{"fqdn", "api.example.com", FQDN()},
		{"mac", "aa:bb:cc:dd:ee:ff", MACAddress()},
		{"phone", "+1 (555) 123-4567", Phone()},
		{"country", "NP", CountryCode()},
		{"currency", "USD", CurrencyCode()},
		{"timezone", "Asia/Kathmandu", TimeZoneName()},
		{"path", "data/file.txt", SafeFilePath()},
		{"charset", "abc123", Charset("abc123")},
		{"reserved", "sujit", ReservedUsername()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.v(tc.value); err != nil {
				t.Fatal(err)
			}
		})
	}
	if err := Slug()("Hello World"); err == nil {
		t.Fatal("expected invalid slug")
	}
	if err := ReservedUsername()("admin"); err == nil {
		t.Fatal("expected reserved username")
	}
}

func TestLocaleParsing(t *testing.T) {
	f, err := ParseLocaleFloat64("1.234,56", LocaleEU)
	if err != nil {
		t.Fatal(err)
	}
	if f != 1234.56 {
		t.Fatalf("got %v", f)
	}
	i, err := ParseLocaleInt64("1,234", LocaleEN)
	if err != nil {
		t.Fatal(err)
	}
	if i != 1234 {
		t.Fatalf("got %v", i)
	}
}

type testDecimal string

func (d testDecimal) String() string { return string(d) }

func TestTypedDecimalAdapter(t *testing.T) {
	RegisterDecimalAdapter[testDecimal](DecimalAdapter[testDecimal]{Parse: func(s string) (testDecimal, error) { return testDecimal(s), nil }, Format: func(d testDecimal) string { return string(d) }})
	defer UnregisterDecimalAdapter[testDecimal]()
	d, err := ToDecimalTyped[testDecimal]("12.3400")
	if err != nil {
		t.Fatal(err)
	}
	if d != "12.3400" {
		t.Fatalf("got %q", d)
	}
	s, err := DecimalToString(d)
	if err != nil || s != "12.3400" {
		t.Fatalf("%q %v", s, err)
	}
}

func TestConvertIntoDeep(t *testing.T) {
	var dst struct {
		Items  []deepItem       `json:"items"`
		Lookup map[string][]int `json:"lookup"`
	}
	err := ConvertInto(&dst, map[string]any{"items": []any{map[string]any{"id": "1", "tags": "a,b"}}, "lookup": map[string]any{"x": []any{"1", 2, 3}}})
	if err != nil {
		t.Fatal(err)
	}
	if dst.Items[0].ID != 1 || dst.Lookup["x"][2] != 3 {
		t.Fatalf("unexpected %#v", dst)
	}
}
