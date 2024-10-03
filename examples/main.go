package main

import (
	"fmt"
	convert "github.com/oarkflow/convert/v2"
)

func main() {
	input := []string{"1", "2", "3", "4"}
	output, ok := convert.ToSlice[any](input)
	if !ok {
		panic("not able to convert")
	}
	fmt.Println(output)
}
