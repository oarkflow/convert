package main

import (
	"fmt"
	"github.com/oarkflow/convert"
)

func main() {
	ints, _ := convert.ToSlice[int]([]any{"1", 2, int64(3)})
	anyv, _ := convert.ToAnySlice("api, edge, prod", convert.WithTrimSpace())
	fmt.Println(ints, anyv)
}
