package main

import (
	"fmt"
	convert "github.com/oarkflow/convert"
)

type Level int

const (
	Debug Level = iota
	Info
	Warn
	Error
)

func main() {
	convert.RegisterEnum(map[string]Level{"debug": Debug, "info": Info, "warn": Warn, "error": Error})
	v, _ := convert.EnumValue[Level]("warn")
	name, _ := convert.EnumName(v)
	fmt.Println(v, name, convert.EnumValid(v))
}
