package main

import (
	"fmt"

	convert "github.com/oarkflow/convert/v2"
)

func main() {
	input := []any{"1", "2", "3", 4}
	output, err := convert.ToSlice[int](input)
	if err != nil {
		panic(err)
	}
	fmt.Println(output)
}
