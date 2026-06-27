package main

import (
	"fmt"
	"os"
	"time"

	convert "github.com/oarkflow/convert"
)

type Config struct {
	Port    int           `env:"PORT" convert:"port" default:"8080"`
	Debug   bool          `env:"DEBUG" convert:"debug"`
	Timeout time.Duration `env:"TIMEOUT" convert:"timeout" default:"5s"`
	Tags    []string      `env:"TAGS" convert:"tags"`
}

func main() {
	os.Setenv("APP_PORT", "9000")
	os.Setenv("APP_DEBUG", "true")
	os.Setenv("APP_TIMEOUT", "2s")
	os.Setenv("APP_TAGS", "api,edge,prod")
	cfg, err := convert.EnvStruct[Config](convert.WithPrefix("APP_"))
	if err != nil {
		panic(err)
	}
	cfg.Tags = convert.EnvSlice[string]("TAGS", nil, convert.WithPrefix("APP_"), convert.WithEnvTrimSpace())
	fmt.Println(cfg.Port, cfg.Debug, cfg.Timeout, cfg.Tags)
}
