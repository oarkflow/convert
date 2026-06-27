package main

import (
	"fmt"

	"example.com/goconvert/convert"
)

func main() {
	port, _ := convert.To[int]("8080")
	debug, _ := convert.ToBool("on")
	ratio, _ := convert.To[float64]([]byte("1.25"))

	buf := make([]byte, 0, 32)
	buf, _ = convert.AppendString(buf, port)

	fmt.Println("port:", port)
	fmt.Println("debug:", debug)
	fmt.Println("ratio:", ratio)
	fmt.Println("appended:", string(buf))
}
