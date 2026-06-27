package main

import (
	"fmt"
	"net/url"
	"time"

	"github.com/oarkflow/convert"
)

type UserID uint64
type Config struct {
	Port    int           `convert:"port" default:"8080"`
	Debug   bool          `convert:"debug"`
	Timeout time.Duration `convert:"timeout" default:"5s"`
	Tags    []string      `convert:"tags"`
}

func main() {
	convert.Register[UserID](func(v any, _ convert.Context) (UserID, error) {
		u, err := convert.ToUint64(v)
		return UserID(u), err
	})

	uid, _ := convert.Convert[UserID]("42")
	cfg, _ := convert.ToStruct[Config](map[string]any{
		"debug":   "true",
		"timeout": "2s",
		"tags":    "api,edge,prod",
	})
	ports, _ := convert.ToSlice[int]("80,443,8080")
	size, _ := convert.ToBytesSize("64MiB")
	q := url.Values{"page": []string{"2"}}

	fmt.Println(uid)
	fmt.Println(cfg.Port, cfg.Debug, cfg.Timeout, cfg.Tags)
	fmt.Println(ports, size, convert.QueryInt(q, "page", 1))
}
