package claudecfg

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// 测试里把 MiniHome 指到临时目录,绝不碰真实的 ~/.minicode。
func useTempHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("MINI_HOME", dir)
	return dir
}

func TestModelsEndpoints(t *testing.T) {
	cases := []struct {
		name string
		p    Provider
		want []string
	}{
		{"填了地址只用它", Provider{BaseURL: "https://a.com/x", ModelsURL: "https://a.com/list"}, []string{"https://a.com/list"}},
		{"带路径的 Base URL 还要试站点根", Provider{BaseURL: "https://api.deepseek.com/anthropic/"}, []string{
			"https://api.deepseek.com/anthropic/v1/models", "https://api.deepseek.com/anthropic/models",
			"https://api.deepseek.com/v1/models", "https://api.deepseek.com/models"}},
		{"以 /v1 结尾", Provider{BaseURL: "https://a.com/v1"}, []string{"https://a.com/v1/models", "https://a.com/models"}},
		{"没有 Base URL", Provider{}, nil},
	}
	for _, c := range cases {
		if got := c.p.ModelsEndpoints(); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}

func TestParseModelIDs(t *testing.T) {
	cases := map[string][]string{
		`{"data":[{"id":"a"},{"id":"b"},{"id":"a"}]}`:        {"a", "b"},
		`{"models":[{"name":"models/gemini-x"},{"id":"y"}]}`: {"gemini-x", "y"},
		`["m1","m2"]`: {"m1", "m2"},
		`{"data":[]}`: nil,
		`not json`:    nil,
	}
	for in, want := range cases {
		if got := parseModelIDs([]byte(in)); len(got) != len(want) || (len(want) > 0 && !reflect.DeepEqual(got, want)) {
			t.Errorf("%s: got %v want %v", in, got, want)
		}
	}
}

func TestFetchModels(t *testing.T) {
	var gotAuth, gotKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth, gotKey = r.Header.Get("Authorization"), r.Header.Get("x-api-key")
		switch r.URL.Path {
		case "/models": // 只有站点根的 /models 存在,/anthropic 下的候选全是 404
			w.Write([]byte(`{"data":[{"id":"chat"},{"id":"reasoner"}]}`))
		case "/denied/v1/models":
			http.Error(w, `{"error":"bad key"}`, http.StatusUnauthorized)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	// 候选依次回退,最后在根上找到
	ids, err := FetchModels(Provider{ID: "t", BaseURL: srv.URL + "/anthropic"}, "k1")
	if err != nil || !reflect.DeepEqual(ids, []string{"chat", "reasoner"}) {
		t.Fatalf("ids=%v err=%v", ids, err)
	}
	if gotAuth != "Bearer k1" || gotKey != "" {
		t.Errorf("Bearer 鉴权头不对:%q %q", gotAuth, gotKey)
	}
	// x-api-key 鉴权
	if _, err := FetchModels(Provider{ID: "t", BaseURL: srv.URL + "/anthropic", Auth: AuthXAPIKey}, "k2"); err != nil {
		t.Fatal(err)
	}
	if gotKey != "k2" || gotAuth != "" {
		t.Errorf("x-api-key 鉴权头不对:%q %q", gotAuth, gotKey)
	}
	// 全部 404 = 不支持
	empty := httptest.NewServer(http.NotFoundHandler())
	defer empty.Close()
	if _, err := FetchModels(Provider{ID: "t", BaseURL: empty.URL + "/none"}, "k"); !errors.Is(err, ErrModelsUnsupported) {
		t.Errorf("想要 ErrModelsUnsupported,得到 %v", err)
	}
	// 401 带原因且不含密钥
	_, err = FetchModels(Provider{ID: "t", BaseURL: srv.URL + "/denied"}, "secret-key-123")
	if err == nil || !strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "secret-key-123") {
		t.Errorf("401 错误不对:%v", err)
	}
	// 缺 Key / 缺地址
	if _, err := FetchModels(Provider{BaseURL: srv.URL}, ""); err == nil {
		t.Error("没有 Key 应该报错")
	}
	if _, err := FetchModels(Provider{}, "k"); err == nil {
		t.Error("没有 Base URL 应该报错")
	}
}

func TestMergeAndToggle(t *testing.T) {
	p := Provider{Models: []string{"old"}}
	p.MergeFetched([]string{"a", "b"}, time.Unix(100, 0))
	if !reflect.DeepEqual(p.Available, []string{"a", "b"}) || p.FetchedAt != 100 {
		t.Fatalf("Available=%v FetchedAt=%d", p.Available, p.FetchedAt)
	}
	if !reflect.DeepEqual(p.Models, []string{"old"}) {
		t.Errorf("已勾选的模型不应被冲掉:%v", p.Models)
	}
	p.ToggleModel("a")
	if !p.HasModel("a") {
		t.Error("勾选后应该有 a")
	}
	p.ToggleModel("a")
	if p.HasModel("a") {
		t.Error("再点一次应取消勾选")
	}
	if !p.AddManual("m1") || p.AddManual("m1") {
		t.Error("手动添加第一次成功、重复失败")
	}
}

func demoStore() Store {
	return Store{
		Providers: []Provider{
			{ID: "ds", Name: "DeepSeek", BaseURL: "https://api.deepseek.com/anthropic/", Models: []string{"deepseek-chat", "deepseek-reasoner"}},
			{ID: "x", Name: "X", BaseURL: "https://x.example", Auth: AuthXAPIKey, Models: []string{"x-1"}},
		},
	}
}

func TestActivateAndEnv(t *testing.T) {
	s := demoStore()
	keys := map[string]string{"ds": "KEY-DS", "x": "KEY-X"}

	if s.MappingStatus(keys) == "" || s.Env(keys) != nil {
		t.Fatal("未启用时不应有环境变量")
	}
	s.Activate("ds", "deepseek-reasoner")
	if s.ActiveID() != "ds" || s.Mapping.Default != "deepseek-reasoner" {
		t.Fatalf("激活失败:%+v", s.Mapping)
	}
	env := map[string]string{}
	for _, e := range s.Env(keys) {
		env[e.Key] = e.Value
	}
	if env["ANTHROPIC_BASE_URL"] != "https://api.deepseek.com/anthropic" || env["ANTHROPIC_AUTH_TOKEN"] != "KEY-DS" ||
		env["ANTHROPIC_MODEL"] != "deepseek-reasoner" || env["ENABLE_TOOL_SEARCH"] != "false" {
		t.Errorf("环境变量不对:%v", env)
	}
	if _, ok := env["ANTHROPIC_API_KEY"]; ok {
		t.Error("Bearer 鉴权不该写 ANTHROPIC_API_KEY")
	}
	// 换到 x-api-key 服务商:其余槽位里旧服务商的模型被换成新模型
	s.Mapping.Sonnet = "deepseek-chat"
	s.Activate("x", "x-1")
	if s.Mapping.Sonnet != "x-1" {
		t.Errorf("换服务商后 Sonnet 槽位应改成新服务商的模型:%q", s.Mapping.Sonnet)
	}
	env = map[string]string{}
	for _, e := range s.Env(keys) {
		env[e.Key] = e.Value
	}
	if env["ANTHROPIC_API_KEY"] != "KEY-X" || env["ANTHROPIC_AUTH_TOKEN"] != "" {
		t.Errorf("x-api-key 鉴权的环境变量不对:%v", env)
	}
	// 回到官方
	s.Activate("", "")
	if s.ActiveID() != "" || s.Env(keys) != nil {
		t.Error("回到官方后不应再注入环境变量")
	}
}

func TestBlocked(t *testing.T) {
	s := demoStore()
	if r := s.Blocked(s.Providers[0], map[string]string{}); r != "未填写 API Key" {
		t.Errorf("缺 Key:%q", r)
	}
	if r := s.Blocked(Provider{ID: "e", BaseURL: "https://e", Models: nil}, map[string]string{"e": "k"}); r != "模型清单为空" {
		t.Errorf("清单为空:%q", r)
	}
	if r := s.Blocked(Provider{ID: "n"}, nil); r != "未填写 Base URL" {
		t.Errorf("缺地址:%q", r)
	}
	if r := s.Blocked(s.Providers[0], map[string]string{"ds": "k"}); r != "" {
		t.Errorf("应该可用:%q", r)
	}
}

func TestCleanMapping(t *testing.T) {
	s := demoStore()
	s.Activate("ds", "deepseek-chat")
	s.Mapping.Opus = "deepseek-reasoner"
	s.Providers[0].Models = []string{"deepseek-chat"} // 取消勾选 reasoner
	s.CleanMapping()
	if s.Mapping.Opus != "" {
		t.Errorf("可选槽位的失效模型应清空:%q", s.Mapping.Opus)
	}
	s.Providers[0].Models = []string{"other"}
	s.CleanMapping()
	if s.Mapping.Default != "other" {
		t.Errorf("默认槽位应退到清单第一项:%q", s.Mapping.Default)
	}
}

func TestNewID(t *testing.T) {
	s := demoStore()
	if id := s.NewID("DeepSeek"); id != "deepseek" {
		t.Errorf("id=%q", id)
	}
	s.Providers = append(s.Providers, Provider{ID: "deepseek"})
	if id := s.NewID("DeepSeek"); id != "deepseek-2" {
		t.Errorf("重名应加序号:%q", id)
	}
	if id := s.NewID("智谱"); id != "provider" {
		t.Errorf("纯中文名退回 provider:%q", id)
	}
}

func TestStoreRoundTripAndKeys(t *testing.T) {
	dir := useTempHome(t)
	s := demoStore()
	s.Activate("ds", "deepseek-chat")
	if err := SaveStore(s); err != nil {
		t.Fatal(err)
	}
	back, err := LoadStore()
	if err != nil || back.Mapping.Provider != "ds" || len(back.Providers) != 2 {
		t.Fatalf("往返失败:%+v %v", back, err)
	}
	if err := SetProviderKey("ds", "KEY-DS"); err != nil {
		t.Fatal(err)
	}
	if ProviderKey("ds") != "KEY-DS" || !HasProviderKey("ds") || HasProviderKey("x") {
		t.Error("密钥读写不对")
	}
	st, err := os.Stat(filepath.Join(dir, "provider-keys.json"))
	if err != nil || st.Mode().Perm() != 0o600 {
		t.Errorf("密钥文件权限应为 0600:%v %v", st, err)
	}
	// 密钥绝不能写进 providers.json
	b, _ := os.ReadFile(filepath.Join(dir, "providers.json"))
	if strings.Contains(string(b), "KEY-DS") {
		t.Error("providers.json 里出现了密钥")
	}
	env := ProviderEnv()
	if len(env) == 0 {
		t.Fatal("ProviderEnv 应读出环境变量")
	}
}

// 旧版纯数组与 Rust 旧版的 base_url 字段都要能读。
func TestLegacyFormats(t *testing.T) {
	dir := useTempHome(t)
	path := filepath.Join(dir, "providers.json")

	os.WriteFile(path, []byte(`[{"id":"a","name":"A","baseUrl":"https://a","active":true,"status":"未测"}]`), 0o600)
	s, err := LoadStore()
	if err != nil || len(s.Providers) != 1 || s.Providers[0].BaseURL != "https://a" {
		t.Fatalf("纯数组迁移失败:%+v %v", s, err)
	}

	os.WriteFile(path, []byte(`{"providers":[{"id":"deepseek","name":"DeepSeek","base_url":"https://api.deepseek.com/anthropic","auth":"bearer","models":["deepseek-chat"]}],
		"mapping":{"enabled":true,"provider":"deepseek","default":"deepseek-chat"}}`), 0o600)
	s, err = LoadStore()
	if err != nil || s.Providers[0].BaseURL != "https://api.deepseek.com/anthropic" || s.ActiveID() != "deepseek" {
		t.Fatalf("base_url 旧字段没读出来:%+v %v", s, err)
	}

	os.WriteFile(path, []byte(`{坏掉的`), 0o600)
	if s, err = LoadStore(); err != nil || len(s.Providers) != 0 {
		t.Errorf("损坏文件应按空配置处理:%+v %v", s, err)
	}
}
