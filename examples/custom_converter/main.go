package main

import (
	"fmt"

	convert "github.com/oarkflow/convert"
)

type UserID uint64

func main() {
	convert.Register(func(v any, _ convert.Context) (UserID, error) {
		u, err := convert.ToUint64(v)
		return UserID(u), err
	})
	id, _ := convert.Convert[UserID]("42")
	fmt.Println(id)
}
