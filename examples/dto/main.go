package main

import (
	"fmt"
	"time"

	"github.com/oarkflow/convert"
)

type Address struct {
	City string `json:"city"`
	Zip  int    `json:"zip"`
}

type UserDTO struct {
	ID      int               `json:"id" validate:"required"`
	Name    string            `json:"name"`
	Active  bool              `json:"active" default:"true"`
	Tags    []string          `json:"tags"`
	Address Address           `json:"address"`
	Limits  map[string]int    `json:"limits"`
	TTL     time.Duration     `json:"ttl" default:"30s"`
	Meta    map[string]string `json:"meta"`
}

func main() {
	input := map[string]any{
		"id":      "1001",
		"name":    []byte("gateway"),
		"tags":    "smpp,edge,production",
		"address": map[string]any{"city": "Kathmandu", "zip": "44600"},
		"limits":  map[any]any{"tps": "500", "burst": 1000},
		"meta":    map[string]any{"owner": "sms", "tier": "gold"},
	}

	user, err := convert.DTOTo[UserDTO](input)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", user)

	asMap, err := convert.DTOMap[string, any](user)
	if err != nil {
		panic(err)
	}
	fmt.Printf("id=%v ttl=%v city=%v\n", asMap["id"], asMap["ttl"], asMap["address"].(Address).City)
}
