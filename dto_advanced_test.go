package convert

import (
	"bytes"
	"context"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

type dtoAdvAddress struct { City string `json:"city"` }
type dtoAdvUser struct {
	ID int `json:"id,readonly"`
	Name string `json:"name,trim,title" validate:"required"`
	Email string `json:"email,lower,trim"`
	City string `json:"address.city"`
	Password string `json:"password,writeonly,sensitive"`
	Token string `json:"token,sensitive"`
}

func TestDTOAdvancedFlattenTransformRedact(t *testing.T) {
	in := map[string]any{"id": 99, "name":"  sujit kumar ", "email":" ADMIN@EXAMPLE.COM ", "address": map[string]any{"city":"Kathmandu"}, "password":"secret", "token":"abc"}
	u, err := DTOTo[dtoAdvUser](in, WithDTOFlatten())
	if err != nil { t.Fatal(err) }
	if u.ID != 0 { t.Fatalf("readonly ID was set: %d", u.ID) }
	if u.Name != "Sujit Kumar" || u.Email != "admin@example.com" || u.City != "Kathmandu" || u.Password != "secret" { t.Fatalf("bad conversion: %#v", u) }
	red, err := Redact(u)
	if err != nil { t.Fatal(err) }
	if _, ok := red["password"]; ok { t.Fatalf("writeonly field leaked: %#v", red) }
	if red["token"] != "[REDACTED]" { t.Fatalf("token not redacted: %#v", red) }
}

func TestDTOReportBatchPatchDiffSchemaAndQuery(t *testing.T) {
	type User struct { Name string `query:"name"`; Age int `query:"age" default:"18"`; City string `query:"address.city"` }
	q := url.Values{"name": {"Sujit"}, "address.city": {"Kathmandu"}, "extra": {"x"}}
	u, err := FromQuery[User](q, WithDTOFlatten())
	if err != nil { t.Fatal(err) }
	if u.Age != 18 || u.City != "Kathmandu" { t.Fatalf("bad query conversion: %#v", u) }
	report := DTOToReport[User](map[string]any{"name":"A", "extra":1})
	if len(report.Warnings) == 0 { t.Fatal("expected report warnings") }
	batch := DTOBatchConvert[User]([]any{map[string]any{"name":"A","age":"20"}, map[string]any{"age":"bad"}}, DTOBatchCollectErrors())
	if len(batch.Errors) != 1 { t.Fatalf("expected one batch error: %#v", batch.Errors) }
	base := User{Name:"A", Age:20}
	changed, err := ApplyPatch(&base, map[string]any{"city":"Pokhara"})
	if err != nil { t.Fatal(err) }
	if !reflect.DeepEqual(changed, []string{"address.city"}) || base.City != "Pokhara" { t.Fatalf("bad patch: %#v %#v", changed, base) }
	diff := SortedDiff(User{Name:"A"}, User{Name:"B"})
	if len(diff)!=1 || diff[0].Path!="name" { t.Fatalf("bad diff: %#v", diff) }
	schema := SchemaOf[User]()
	if schema.Type != "object" || len(schema.Fields) != 3 { t.Fatalf("bad schema: %#v", schema) }
}

func TestDTOStreamAndPair(t *testing.T) {
	type A struct{ Name string `json:"name"` }
	type B struct{ Label string `json:"label"` }
	RegisterPair[A,B](func(a A)(B,error){ return B{Label:a.Name}, nil })
	b, err := DTOConvertPair[A,B](A{Name:"x"})
	if err != nil || b.Label != "x" { t.Fatalf("bad pair: %#v %v", b, err) }
	var out bytes.Buffer
	err = DTOStreamJSONL[A](context.Background(), strings.NewReader("{\"name\":\"a\"}\n"), &out)
	if err != nil { t.Fatal(err) }
	if !strings.Contains(out.String(), "\"name\":\"a\"") { t.Fatalf("bad stream: %s", out.String()) }
}
