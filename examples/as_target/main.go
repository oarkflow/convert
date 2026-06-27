package main

import (
	"fmt"
	"time"

	"github.com/oarkflow/convert"
)

func main() {
	var a = 1.2
	var b = "5"

	asString, _ := convert.As(a, b)
	fmt.Printf("As(a,b): %T %v\n", asString, asString)

	var n int
	asInt, _ := convert.As("42", &n)
	fmt.Printf("As(string,*int): %T %v\n", asInt, asInt)

	asDuration, _ := convert.AsTo("5s", time.Duration(0))
	fmt.Printf("AsTo(duration): %T %v\n", asDuration, asDuration)
}
