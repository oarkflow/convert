package main

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/oarkflow/convert"
)

type User struct {
	ID    int    `json:"id" query:"id" csv:"id" validate:"required"`
	Name  string `json:"name,trim,title" query:"name" csv:"name"`
	Email string `json:"email" validate:"email" sensitive:"true"`
	Role  string `json:"role" default:"user"`
}

type UserPatch struct {
	Name  convert.Optional[string] `json:"name"`
	Email convert.Optional[string] `json:"email"`
}

func main() {
	user := convert.MustMap[User](map[string]any{"id": "42", "name": " sujit ", "email": "sujit@example.com"})
	fmt.Printf("mapped: %+v\n", user)

	query := url.Values{"id": {"7"}, "name": {"alice"}, "email": {"alice@example.com"}}
	bound, _ := convert.Bind[User](convert.FromQuerySource(query), convert.FromDefaultSource(map[string]any{"role": "member"}))
	fmt.Printf("bound: %+v\n", bound)

	changed, _ := convert.ApplyOptionalPatch(&bound, UserPatch{Name: convert.Some("Alice Updated"), Email: convert.Null[string]()})
	fmt.Printf("patch changed=%v user=%+v\n", changed, bound)

	trace := convert.MapTrace[User](map[string]any{"id": "1", "name": "bob", "email": "bob@example.com"})
	fmt.Printf("trace steps=%d err=%v\n", len(trace.Steps), trace.Err)

	fmt.Println("safe:", convert.SafeJSON(user))
	fmt.Println("schema:", convert.StableJSONSchema[User]())

	csvData := "id,name,email\n1,A,a@example.com\n2,B,b@example.com\n"
	users, _ := convert.ReadCSV[User](strings.NewReader(csvData))
	fmt.Printf("csv users=%d\n", len(users))
}
