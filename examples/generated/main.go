package main

import "fmt"

type Config struct {
	Port  int  `convert:"port" default:"8080"`
	Debug bool `convert:"debug"`
}

func main() {
	cfg, _ := ConvertConfig(map[string]any{"port": "9000", "debug": "true"})
	fmt.Println(cfg.Port, cfg.Debug)
}
