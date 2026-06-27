package main

import (
	"fmt"
	"time"

	convert "github.com/oarkflow/convert"
)

type GeneratedConfig struct {
	Port    int           `convert:"port"`
	Timeout time.Duration `convert:"timeout"`
}

// This example uses the runtime binder. The equivalent generated binder is created with:
// go run github.com/oarkflow/convert/cmd/convertgen -type GeneratedConfig -input main.go
func main() {
	cfg, err := convert.ToStruct[GeneratedConfig](map[string]any{"port": "8080", "timeout": "2s"})
	if err != nil {
		panic(err)
	}
	fmt.Println(cfg.Port, cfg.Timeout)
}
