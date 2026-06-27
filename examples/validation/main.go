package main

import (
	"fmt"
	convert "github.com/oarkflow/convert"
)

func main() {
	port, err := convert.ToValidated[int]("8080", convert.Range(1, 65535))
	email, err2 := convert.ToValidated[string]("hello@example.com", convert.Email())
	fmt.Println(port, err == nil, email, err2 == nil)
}
