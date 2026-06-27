package main

import (
	"fmt"
	"github.com/oarkflow/convert"
)

func main() {
	hx, _ := convert.ToHex("hi")
	b64, _ := convert.ToBase64("hi")
	raw := convert.Uint64ToBytes(42, convert.BigEndian())
	n, _ := convert.BytesToUint64(raw, convert.BigEndian())
	fmt.Println(hx, b64, n)
}
