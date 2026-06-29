package convert

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

type dtoAddress struct {
	City string `json:"city"`
	Zip  int    `json:"zip"`
}

type dtoUser struct {
	ID        int            `json:"id" validate:"required"`
	Name      string         `json:"name"`
	Active    bool           `json:"active" default:"true"`
	Tags      []string       `json:"tags"`
	Scores    []int          `json:"scores"`
	Address   dtoAddress     `json:"address"`
	Lookup    map[string]int `json:"lookup"`
	Timeout   time.Duration  `json:"timeout" default:"5s"`
	CreatedAt time.Time      `json:"created_at"`
	Meta      map[string]any `json:"meta"`
	Ptr       *dtoAddress    `json:"ptr"`
}

func TestDTOMapStructSliceDeep(t *testing.T) {
	created := time.Unix(1700000000, 0).UTC()
	got, err := DTOTo[dtoUser](map[string]any{
		"id":         "42",
		"name":       []byte("sujit"),
		"tags":       "api, sms, dto",
		"scores":     []any{"1", 2, int64(3)},
		"address":    map[any]any{"city": "Kathmandu", "zip": "44600"},
		"lookup":     map[any]any{"a": "1", "b": 2},
		"created_at": created.Format(time.RFC3339),
		"meta":       map[string]any{"role": "admin"},
		"ptr":        map[string]any{"city": "Pokhara", "zip": 33700},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != 42 || got.Name != "sujit" || !got.Active || got.Tags[1] != "sms" || got.Scores[2] != 3 || got.Address.Zip != 44600 || got.Lookup["b"] != 2 || got.Timeout != 5*time.Second || got.CreatedAt.Unix() != 1700000000 || got.Ptr.City != "Pokhara" {
		t.Fatalf("unexpected %#v", got)
	}
}

func TestDTOStructSourceToMapAndStruct(t *testing.T) {
	src := dtoUser{ID: 7, Name: "source", Tags: []string{"a"}, Address: dtoAddress{City: "A", Zip: 1}}
	m, err := DTOMap[string, any](src)
	if err != nil {
		t.Fatal(err)
	}
	if m["id"] != 7 || m["name"] != "source" {
		t.Fatalf("bad map %#v", m)
	}
	var dst dtoUser
	if err := DTO(&dst, src); err != nil {
		t.Fatal(err)
	}
	if dst.ID != 7 || dst.Address.City != "A" {
		t.Fatalf("bad struct %#v", dst)
	}
}

func TestDTOErrorPathAndUnused(t *testing.T) {
	_, err := DTOTo[dtoUser](map[string]any{"id": "bad", "name": "x"})
	if err == nil {
		t.Fatal("expected error")
	}
	var detail *ErrorDetail
	if !errors.As(err, &detail) || detail.Path != "id" {
		t.Fatalf("expected id path, got %#v %v", detail, err)
	}
	var out dtoAddress
	err = DTO(&out, map[string]any{"city": "A", "unused": true}, WithDTOErrorUnused())
	if err == nil || !strings.Contains(err.Error(), "unused") {
		t.Fatalf("expected unused error, got %v", err)
	}
}

func TestDTOHook(t *testing.T) {
	type Custom struct{ V string }
	hook := func(ctx DTOContext, dst reflect.Value, src any) (bool, error) {
		if dst.Type() == reflect.TypeOf(Custom{}) {
			dst.Set(reflect.ValueOf(Custom{V: "hook:" + ToDebugString(src)}))
			return true, nil
		}
		return false, nil
	}
	var out struct {
		C Custom `json:"c"`
	}
	if err := DTO(&out, map[string]any{"c": 123}, WithDTODecodeHook(hook)); err != nil {
		t.Fatal(err)
	}
	if out.C.V != "hook:123" {
		t.Fatalf("hook failed %#v", out)
	}
}
