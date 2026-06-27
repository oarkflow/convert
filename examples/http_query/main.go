package main

import (
	"fmt"
	"github.com/oarkflow/convert"
	"net/url"
	"time"
)

type Request struct {
	Page    int           `query:"page" default:"1"`
	Tags    []string      `query:"tag"`
	Timeout time.Duration `query:"timeout" default:"1s"`
}

func main() {
	q := url.Values{"page": {"2"}, "tag": {"api", "edge"}, "timeout": {"3s"}}
	var r Request
	_ = convert.BindQuery(&r, q)
	fmt.Println(r.Page, r.Tags, r.Timeout)
}
