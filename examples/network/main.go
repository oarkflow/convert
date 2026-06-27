package main

import (
	"fmt"
	convert "github.com/oarkflow/convert"
)

func main() {
	u, _ := convert.ToURL("https://example.com/a")
	ip, _ := convert.ToAddr("127.0.0.1")
	prefix, _ := convert.ToPrefix("10.0.0.0/8")
	fmt.Println(u.Host, ip.String(), prefix.String())
}
