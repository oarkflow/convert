package main

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/oarkflow/convert"
)

type UserDTO struct {
	ID       int    `json:"id,readonly"`
	Name     string `json:"name,trim,title" validate:"required"`
	Email    string `json:"email,trim,lower"`
	City     string `json:"address.city"`
	APIKey   string `json:"api_key,sensitive"`
	Password string `json:"password,writeonly,sensitive"`
}

func main() {
	input := map[string]any{
		"id":       42,
		"name":     "  sujit kumar ",
		"email":    " ADMIN@EXAMPLE.COM ",
		"address":  map[string]any{"city": "Kathmandu"},
		"api_key":  "secret-key",
		"password": "secret-password",
	}

	user, err := convert.DTOTo[UserDTO](input, convert.WithDTOFlatten())
	if err != nil {
		panic(err)
	}
	fmt.Printf("user=%+v\n", user)

	safe, _ := convert.Redact(user)
	fmt.Printf("safe=%+v\n", safe)

	q := url.Values{"name": {"sujit"}, "address.city": {"Kathmandu"}}
	fromQuery, _ := convert.FromQuery[UserDTO](q, convert.WithDTOFlatten())
	fmt.Printf("query=%+v\n", fromQuery)

	changed, _ := convert.ApplyPatch(&user, map[string]any{"name": "updated user"})
	fmt.Printf("changed=%v patched=%+v\n", changed, user)

	batch := convert.DTOBatchConvert[UserDTO]([]any{input, map[string]any{"email": 10}}, convert.DTOBatchCollectErrors(), convert.DTOBatchWithOptions(convert.WithDTOFlatten()))
	fmt.Printf("batch values=%d errors=%d\n", len(batch.Values), len(batch.Errors))

	var out bytes.Buffer
	_ = convert.DTOStreamJSONL[UserDTO](context.Background(), strings.NewReader(`{"name":"json line","email":"A@B.COM"}`+"\n"), &out)
	fmt.Print(out.String())
}
