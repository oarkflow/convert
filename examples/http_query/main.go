package main

import (
	"fmt"
	"net/url"
	"time"

	convert "github.com/oarkflow/convert"
)

type Request struct {
	Page    int           `query:"page" default:"1"`
	Debug   bool          `query:"debug"`
	Timeout time.Duration `query:"timeout" default:"5s"`
	Tags    []string      `query:"tag"`
}

func main() {
	values := url.Values{"page": {"2"}, "debug": {"true"}, "timeout": {"1s"}, "tag": {"api", "edge"}}
	var req Request
	if err := convert.BindQuery(&req, values); err != nil {
		panic(err)
	}
	fmt.Println(req.Page, req.Debug, req.Timeout, req.Tags)
}
