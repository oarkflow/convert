package main

import (
	"fmt"
	convert "github.com/oarkflow/convert"
)

func main() {
	h, _ := convert.ToHex("go")
	b, _ := convert.FromHex(h)
	raw := convert.Uint64ToBytes(42, convert.BigEndian())
	n, _ := convert.BytesToUint64(raw, convert.BigEndian())
	fmt.Println(h, string(b), n)
}
