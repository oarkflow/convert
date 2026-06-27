package convert

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestPolicyNullableAndSQL(t *testing.T) {
	v, err := ToWith[int](" 42 ", TrimSpace())
	if err != nil || v != 42 {
		t.Fatalf("ToWith=%v err=%v", v, err)
	}
	p, err := ToPtr[bool]("yes")
	if err != nil || p == nil || !*p {
		t.Fatalf("ToPtr failed")
	}
	n := ToNullable[int]("7")
	if !n.Valid || n.Value != 7 {
		t.Fatalf("nullable failed")
	}
	if ns := ToSQLNullString("abc"); !ns.Valid || ns.String != "abc" {
		t.Fatalf("sql null string failed")
	}
	if nt := ToSQLNullTime("1700000000"); !nt.Valid || nt.Time.Unix() != 1700000000 {
		t.Fatalf("sql null time failed")
	}
	if _, ok := any(sql.NullString{}).(any); !ok {
		t.Fatal("keep import")
	}
}

func TestSliceMapStruct(t *testing.T) {
	ints, err := ToSlice[int]("1, 2,3", WithTrimSpace())
	if err != nil || len(ints) != 3 || ints[1] != 2 {
		t.Fatalf("slice %#v err=%v", ints, err)
	}
	m, err := ToMap[string, int](map[string]any{"a": "1", "b": 2})
	if err != nil || m["a"] != 1 || m["b"] != 2 {
		t.Fatalf("map %#v err=%v", m, err)
	}
	type Config struct {
		Port    int           `convert:"port" default:"8080"`
		Debug   bool          `convert:"debug"`
		Timeout time.Duration `convert:"timeout"`
		Names   []string      `convert:"names"`
	}
	cfg, err := ToStruct[Config](map[string]any{"debug": "true", "timeout": "5s", "names": "a,b"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 8080 || !cfg.Debug || cfg.Timeout != 5*time.Second || len(cfg.Names) != 2 {
		t.Fatalf("bad cfg %#v", cfg)
	}
}

func TestRegistryEnumValidation(t *testing.T) {
	type UserID uint64
	Register[UserID](func(v any, ctx Context) (UserID, error) { u, err := ToUint64(v); return UserID(u), err })
	defer Unregister[UserID]()
	uid, err := Convert[UserID]("99")
	if err != nil || uid != 99 {
		t.Fatalf("registry %v err=%v", uid, err)
	}
	type Level int
	const (
		Debug Level = iota
		Info
		Warn
	)
	RegisterEnum(map[string]Level{"debug": Debug, "info": Info, "warn": Warn})
	lv, err := EnumValue[Level]("warn")
	if err != nil || lv != Warn || !EnumValid(Warn) {
		t.Fatalf("enum failed")
	}
	port, err := ToValidated[int]("8080", Min(1), Max(65535))
	if err != nil || port != 8080 {
		t.Fatalf("validation failed")
	}
}

func TestBytesSizeNetPathHTTPJSON(t *testing.T) {
	hx, err := ToHex("go")
	if err != nil || hx != "676f" {
		t.Fatalf("hex %q err=%v", hx, err)
	}
	b, err := FromBase64("Z28=")
	if err != nil || string(b) != "go" {
		t.Fatalf("base64 %q err=%v", b, err)
	}
	sz, err := ToBytesSize("2MiB")
	if err != nil || sz != 2<<20 {
		t.Fatalf("size %d err=%v", sz, err)
	}
	if _, err := ToURL("https://example.com/a"); err != nil {
		t.Fatal(err)
	}
	if _, err := ToAddr("127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	if !ValidUUID("550e8400-e29b-41d4-a716-446655440000") {
		t.Fatal("uuid")
	}
	data := map[string]any{"server": map[string]any{"port": "8080"}}
	port, err := Get[int](data, "server.port")
	if err != nil || port != 8080 {
		t.Fatalf("get %v err=%v", port, err)
	}
	q := url.Values{"page": []string{"5"}}
	if QueryInt(q, "page", 1) != 5 {
		t.Fatal("query")
	}
	h := http.Header{"X-Retry": []string{"3"}}
	if HeaderInt(h, "X-Retry", 0) != 3 {
		t.Fatal("header")
	}
	j := map[string]any{"n": json.Number("12")}
	NormalizeJSONNumbers(j)
	if j["n"].(int64) != 12 {
		t.Fatalf("json normalize %#v", j)
	}
}
