package main

import (
	"fmt"
	"github.com/oarkflow/convert"
)

func main() {
	n, _ := convert.ScanTo[int64]([]byte("42"))
	v, _ := convert.ToDriverValue(42)
	fmt.Println(n, v)
}
