package main

import (
	"fmt"
	"github.com/oarkflow/convert"
)

type Level int

const (
	Debug Level = iota
	Info
	Warn
)

func main() {
	convert.RegisterEnum(map[string]Level{"debug": Debug, "info": Info, "warn": Warn})
	v, _ := convert.EnumValue[Level]("warn")
	name, _ := convert.EnumName(v)
	fmt.Println(v, name)
}
