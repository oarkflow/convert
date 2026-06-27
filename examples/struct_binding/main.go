package main

import (
	"fmt"
	"github.com/oarkflow/convert"
)

type Server struct {
	Port int      `convert:"port" default:"8080"`
	Tags []string `convert:"tags"`
}

func main() {
	s, _ := convert.ToStruct[Server](map[string]any{"port": "9000", "tags": []any{"api", "edge"}})
	fmt.Println(s.Port, s.Tags)
}
