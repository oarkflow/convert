package main

import (
	"fmt"
	"time"

	"github.com/oarkflow/convert"
)

type Item struct {
	ID   int      `json:"id" validate:"required"`
	Tags []string `json:"tags"`
}

type Config struct {
	Port    int              `env:"PORT" default:"8080"`
	Host    string           `json:"host" validate:"required,hostname"`
	Items   []Item           `json:"items"`
	Lookup  map[string][]int `json:"lookup"`
	Timeout time.Duration    `query:"timeout" default:"2s"`
}

type Decimal string

func (d Decimal) String() string { return string(d) }

func main() {
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
	if err != nil {
		panic(err)
	}

	nums, err := convert.ToDeepSlice[int]([]any{"1", 2, uint64(3)})
	if err != nil {
		panic(err)
	}

	amount, err := convert.ParseLocaleFloat64("1.234,56", convert.LocaleEU)
	if err != nil {
		panic(err)
	}

	convert.RegisterDecimalAdapter[Decimal](convert.DecimalAdapter[Decimal]{
		Parse:  func(s string) (Decimal, error) { return Decimal(s), nil },
		Format: func(d Decimal) string { return string(d) },
	})
	dec, err := convert.ToDecimalTyped[Decimal]("12.3400")
	if err != nil {
		panic(err)
	}

	_, err = convert.ToValidated[string]("release-2026", convert.Slug())
	if err != nil {
		panic(err)
	}

	fmt.Println(cfg.Port, cfg.Host, cfg.Items[0].ID, cfg.Lookup["ports"][2], cfg.Timeout)
	fmt.Println(nums, amount, dec)
}
