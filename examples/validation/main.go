package main

import (
	"fmt"
	"github.com/oarkflow/convert"
)

func main() {
	port, _ := convert.ToValidated[int]("8080", convert.Min(1), convert.Max(65535))
	email, _ := convert.ToValidated[string]("dev@example.com", convert.Email())
	fmt.Println(port, email)
}
