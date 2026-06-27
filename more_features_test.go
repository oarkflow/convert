package convert

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestSliceConversionAnyAndTyped(t *testing.T) {
	ints, err := ToSlice[int]([]any{"1", 2, int64(3)})
	if err != nil || !reflect.DeepEqual(ints, []int{1, 2, 3}) {
		t.Fatalf("ints=%v err=%v", ints, err)
	}
	anys, err := ToAnySlice("a,b,c", WithSeparator(","))
	if err != nil || !reflect.DeepEqual(anys, []any{"a", "b", "c"}) {
		t.Fatalf("anys=%#v err=%v", anys, err)
	}
	durs, err := ToDurationSlice([]any{"1s", int64(time.Second)})
	if err != nil || durs[0] != time.Second || durs[1] != time.Second {
		t.Fatalf("durs=%v err=%v", durs, err)
	}
	uniq, err := ToSlice[int]("1,1,2", WithUnique())
	if err != nil || !reflect.DeepEqual(uniq, []int{1, 2}) {
		t.Fatalf("uniq=%v err=%v", uniq, err)
	}
}

func TestEnvAndBinding(t *testing.T) {
	t.Setenv("APP_PORT", "9090")
	t.Setenv("APP_TAGS", "api, edge")
	if got := Env("PORT", 80, WithPrefix("APP_")); got != 9090 {
		t.Fatal(got)
	}
	tags := EnvSlice[string]("TAGS", nil, WithPrefix("APP_"), WithEnvTrimSpace())
	if !reflect.DeepEqual(tags, []string{"api", "edge"}) {
		t.Fatal(tags)
	}
	type Req struct {
		Page  int    `query:"page"`
		Tags  []int  `query:"tag"`
		Token string `header:"X-Token"`
	}
	var q Req
	if err := BindQuery(&q, url.Values{"page": {"2"}, "tag": {"1", "2"}}); err != nil {
		t.Fatal(err)
	}
	if q.Page != 2 || !reflect.DeepEqual(q.Tags, []int{1, 2}) {
		t.Fatalf("%+v", q)
	}
	if err := BindHeader(&q, http.Header{"X-Token": {"abc"}}); err != nil {
		t.Fatal(err)
	}
	if q.Token != "abc" {
		t.Fatal(q.Token)
	}
}

func TestConversionGraph(t *testing.T) {
	type UserID uint64
	RegisterEdge(func(s string) (uint64, error) { return ToUint64(s) })
	RegisterEdge(func(u uint64) (UserID, error) { return UserID(u), nil })
	got, err := ConvertGraph[UserID]("42")
	if err != nil || got != 42 {
		t.Fatalf("got=%v err=%v", got, err)
	}
}

func TestCSVPathMatrixAndJSON(t *testing.T) {
	type Row struct {
		ID      int  `csv:"id"`
		Enabled bool `csv:"enabled"`
	}
	var r Row
	if err := BindCSVRow(&r, []string{"id", "enabled"}, []string{"7", "true"}); err != nil {
		t.Fatal(err)
	}
	if r.ID != 7 || !r.Enabled {
		t.Fatalf("%+v", r)
	}
	if MatrixMarkdown() == "" {
		t.Fatal("empty matrix")
	}
	m := map[string]any{"n": json.Number("42"), "xs": []any{json.Number("1")}}
	NormalizeJSONNumbers(m)
	if m["n"] != int64(42) {
		t.Fatalf("%#v", m)
	}
}

func TestNetworkSizeBytesAndValidation(t *testing.T) {
	if _, err := ToValidated("hello@example.com", Email()); err != nil {
		t.Fatal(err)
	}
	if s, _ := ToBytesSize("64MiB"); s != 64<<20 {
		t.Fatal(s)
	}
	h, _ := ToHex("go")
	b, err := FromHex(h)
	if err != nil || string(b) != "go" {
		t.Fatal(h, b, err)
	}
	if _, err := ToAddr("127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	if _, err := ToURL("https://example.com"); err != nil {
		t.Fatal(err)
	}
}

func TestEnvStruct(t *testing.T) {
	type Config struct {
		Port  int  `env:"PORT" convert:"port"`
		Debug bool `env:"DEBUG" convert:"debug"`
	}
	os.Setenv("X_PORT", "8081")
	os.Setenv("X_DEBUG", "true")
	defer os.Unsetenv("X_PORT")
	defer os.Unsetenv("X_DEBUG")
	cfg, err := EnvStruct[Config](WithPrefix("X_"))
	if err != nil || cfg.Port != 8081 || !cfg.Debug {
		t.Fatalf("%+v err=%v", cfg, err)
	}
}
