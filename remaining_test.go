package convert

import (
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

type fullBindCfg struct {
	Port    int           `query:"port" default:"8080"`
	Debug   bool          `query:"debug"`
	Tags    []string      `query:"tag"`
	Timeout time.Duration `query:"timeout" default:"1s"`
}

func TestProductionSliceConversionAny(t *testing.T) {
	got, err := ToSlice[int]([]any{"1", 2, int64(3)}, WithUnique())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []int{1, 2, 3}) {
		t.Fatalf("got %#v", got)
	}
	anyv, err := ToAnySlice("a, b,,a", WithTrimSpace(), WithIgnoreEmpty(), WithUnique())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(anyv, []any{"a", "b"}) {
		t.Fatalf("got %#v", anyv)
	}
	type custom struct {
		Name string `convert:"name"`
	}
	cs, err := ToTypedSlice[custom]([]any{map[string]any{"name": "a"}, map[string]any{"name": "b"}})
	if err != nil {
		t.Fatal(err)
	}
	if cs[1].Name != "b" {
		t.Fatalf("bad custom slice %#v", cs)
	}
}

func TestBindQueryHeaderCSVEnv(t *testing.T) {
	q := url.Values{"port": {"9000"}, "debug": {"true"}, "tag": {"api", "edge"}, "timeout": {"2s"}}
	var cfg fullBindCfg
	if err := BindQuery(&cfg, q); err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 9000 || !cfg.Debug || cfg.Timeout != 2*time.Second || len(cfg.Tags) != 2 {
		t.Fatalf("bad cfg %#v", cfg)
	}
	h := http.Header{"X-Retry": {"3"}}
	var hv struct {
		Retry int `header:"X-Retry"`
	}
	if err := BindHeader(&hv, h); err != nil {
		t.Fatal(err)
	}
	if hv.Retry != 3 {
		t.Fatal(hv.Retry)
	}
	var csvCfg fullBindCfg
	if err := BindCSVRow(&csvCfg, []string{"port", "debug", "tag"}, []string{"7000", "true", "a,b"}); err != nil {
		t.Fatal(err)
	}
	if csvCfg.Port != 7000 || len(csvCfg.Tags) != 2 {
		t.Fatalf("bad csv cfg %#v", csvCfg)
	}
	t.Setenv("APP_PORT", "6060")
	t.Setenv("APP_TAGS", "one,two")
	if Env[int]("PORT", 1, WithPrefix("APP_")) != 6060 {
		t.Fatal("env int")
	}
	tags, err := EnvSlice[string]("TAGS", WithPrefix("APP_"), WithEnvTrimSpace())
	if err != nil || len(tags) != 2 {
		t.Fatalf("env slice %#v %v", tags, err)
	}
	os.Unsetenv("APP_PORT")
}

func TestConversionGraphAndPrecision(t *testing.T) {
	type UserID uint64
	RegisterEdge[string, uint64](func(s string) (uint64, error) { return ToUint64(s) })
	RegisterEdge[uint64, UserID](func(u uint64) (UserID, error) { return UserID(u), nil })
	id, err := ConvertGraph[UserID]("42")
	if err != nil || id != 42 {
		t.Fatalf("id %v err %v", id, err)
	}
	if _, err := ToIntWith(1.2, NoPrecisionLoss()); err == nil {
		t.Fatal("expected precision loss")
	}
	i, err := ToIntWith(1.2, AllowFloatToIntTruncate())
	if err != nil || i != 1 {
		t.Fatalf("truncate %d %v", i, err)
	}
}

func TestConversionMatrixAndPathError(t *testing.T) {
	m := ConversionMatrix()
	if !strings.Contains(m, "[]any/slice") || !strings.Contains(m, "duration") {
		t.Fatal(m)
	}
	err := PathError("server.port", KindString, KindInt, "bad", ErrInvalid)
	if !strings.Contains(err.Error(), "server.port") {
		t.Fatal(err)
	}
}
