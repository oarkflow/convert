package convert

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

type usabilityUser struct {
	ID    int    `json:"id" query:"id" csv:"id" db:"id" validate:"required"`
	Name  string `json:"name,trim,title" query:"name" csv:"name" db:"name"`
	Email string `json:"email" validate:"email" sensitive:"true"`
	Role  string `json:"role" default:"user"`
}

type usabilityPatch struct {
	Name  Optional[string] `json:"name"`
	Email Optional[string] `json:"email"`
}

func TestUsabilityMapTraceDryRun(t *testing.T) {
	in := map[string]any{"id": "7", "name": " sujit ", "email": "bad"}
	_, err := MapAll[usabilityUser](in)
	if err == nil || !strings.Contains(HumanError(err), "email") {
		t.Fatalf("expected email error, got %v", err)
	}
	tr := MapTrace[usabilityUser](map[string]any{"id": "7", "name": " sujit ", "email": "a@b.com"})
	if tr.Err != nil || len(tr.Steps) == 0 {
		t.Fatalf("trace failed: %+v", tr)
	}
	dr := DryRun[usabilityUser](map[string]any{"name": "x"})
	if len(dr.Errors) == 0 {
		t.Fatal("expected dry-run missing required error")
	}
}

func TestUsabilityOptionalPatchAndSafeLog(t *testing.T) {
	u := usabilityUser{ID: 1, Name: "Old", Email: "old@example.com"}
	changed, err := ApplyOptionalPatch(&u, usabilityPatch{Name: Some("New"), Email: Null[string]()})
	if err != nil || len(changed) != 2 || u.Name != "New" || u.Email != "" {
		t.Fatalf("bad patch changed=%v user=%+v err=%v", changed, u, err)
	}
	if s := SafeJSON(u); strings.Contains(s, "old@example.com") || !strings.Contains(s, "REDACTED") {
		t.Fatalf("unsafe json: %s", s)
	}
}

func TestUsabilityBindCSVBatchCollection(t *testing.T) {
	q := url.Values{"id": {"9"}, "name": {"alice"}, "email": {"alice@example.com"}}
	u, err := Bind[usabilityUser](FromQuerySource(q), FromDefaultSource(map[string]any{"role": "member"}))
	if err != nil || u.ID != 9 || u.Role != "member" {
		t.Fatalf("bind failed: %+v %v", u, err)
	}
	csvData := "id,name,email\n1,A,a@example.com\n2,B,b@example.com\n"
	users, err := ReadCSV[usabilityUser](strings.NewReader(csvData))
	if err != nil || len(users) != 2 || users[1].ID != 2 {
		t.Fatalf("csv failed: %+v %v", users, err)
	}
	byID, err := SliceToMap[usabilityUser, int](users, "id")
	if err != nil || byID[1].Name != "A" {
		t.Fatalf("slice index failed: %+v %v", byID, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	batch, err := BatchContext[usabilityUser](ctx, []any{map[string]any{"id": 1, "email": "a@b.com"}})
	if err != nil || len(batch) != 1 {
		t.Fatalf("batch ctx failed: %+v %v", batch, err)
	}
}

func TestUsabilityHTTPAndConfig(t *testing.T) {
	req, _ := http.NewRequest("GET", "/?id=11&name=bob&email=bob@example.com", nil)
	var u usabilityUser
	if err := BindRequestQuery(req, &u); err != nil || u.ID != 11 {
		t.Fatalf("query bind failed: %+v %v", u, err)
	}
	cfg, err := LoadConfig[usabilityUser](MapSource(map[string]any{"id": 5, "name": "Cfg", "email": "cfg@example.com"}))
	if err != nil || cfg.ID != 5 {
		t.Fatalf("config failed: %+v %v", cfg, err)
	}
	if len(Describe[usabilityUser]()) == 0 || StableJSONSchema[usabilityUser]() == "" {
		t.Fatal("describe/schema missing")
	}
}
