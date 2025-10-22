package main

import (
	"fmt"

	"github.com/oarkflow/convert"
)

func main() {
	input := []any{"1", "2", "3", 4}
	output, err := convert.ToSlice[int](input)
	if err != nil {
		panic(err)
	}
	fmt.Println(output)
}
