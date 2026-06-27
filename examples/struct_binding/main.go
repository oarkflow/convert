package main

import (
	"fmt"
	"time"

	convert "github.com/oarkflow/convert"
)

type Server struct {
	Port    int           `convert:"port" default:"8080"`
	TLS     bool          `convert:"tls"`
	Timeout time.Duration `convert:"timeout" default:"3s"`
	Ports   []int         `convert:"ports"`
}

func main() {
	var s Server
	err := convert.Populate(&s, map[string]any{"tls": "true", "ports": []any{"80", 443, 8080}})
	if err != nil {
		panic(err)
	}
	fmt.Println(s.Port, s.TLS, s.Timeout, s.Ports)
}
