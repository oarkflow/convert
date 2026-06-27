package main

import (
	"fmt"
	"github.com/oarkflow/convert"
	"os"
	"time"
)

type Config struct {
	Port    int           `convert:"port" default:"8080"`
	Debug   bool          `convert:"debug"`
	Timeout time.Duration `convert:"timeout" default:"5s"`
}

func main() {
	os.Setenv("APP_PORT", "9000")
	os.Setenv("APP_DEBUG", "true")
	cfg, _ := convert.EnvStruct[Config](convert.WithPrefix("APP_"))
	fmt.Println(cfg.Port, cfg.Debug, cfg.Timeout)
}
