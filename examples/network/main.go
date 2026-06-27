package main

import (
	"fmt"
	"github.com/oarkflow/convert"
)

func main() {
	u, _ := convert.ToURL("https://example.com/a")
	ip, _ := convert.ToAddr("127.0.0.1")
	p, _ := convert.ToPrefix("10.0.0.0/24")
	fmt.Println(u.Host, ip, p)
}
