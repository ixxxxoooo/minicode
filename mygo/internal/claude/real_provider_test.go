package claude

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/ixxxxoooo/minicode/internal/claudecfg"
	"github.com/ixxxxoooo/minicode/internal/core"
)

// mockAnthropic 是本机的 Anthropic 协议桩:记录每个 /v1/messages 请求的鉴权头与模型名,回一段固定文字。
type mockAnthropic struct {
	srv  *httptest.Server
	mu   sync.Mutex
	seen []mockReq
}

type mockReq struct{ Auth, APIKey, Model string }

func newMockAnthropic(reply string) *mockAnthropic {
	m := &mockAnthropic{}
	m.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/v1/messages") {
			http.NotFound(w, r)
			return
		}
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Model  string `json:"model"`
			Stream bool   `json:"stream"`
		}
		_ = json.Unmarshal(body, &req)
		m.mu.Lock()
		m.seen = append(m.seen, mockReq{r.Header.Get("Authorization"), r.Header.Get("x-api-key"), req.Model})
		m.mu.Unlock()
		msg := func(text string) string {
			return fmt.Sprintf(`{"id":"msg_mock","type":"message","role":"assistant","model":%q,"content":[{"type":"text","text":%q}],"stop_reason":"end_turn","stop_sequence":null,"usage":{"input_tokens":10,"output_tokens":3}}`, req.Model, text)
		}
		if !req.Stream {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, msg(reply))
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		ev := func(name, data string) { fmt.Fprintf(w, "event: %s\ndata: %s\n\n", name, data) }
		ev("message_start", fmt.Sprintf(`{"type":"message_start","message":{"id":"msg_mock","type":"message","role":"assistant","model":%q,"content":[],"stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":10,"output_tokens":1}}}`, req.Model))
		ev("content_block_start", `{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`)
		ev("content_block_delta", fmt.Sprintf(`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":%q}}`, reply))
		ev("content_block_stop", `{"type":"content_block_stop","index":0}`)
		ev("message_delta", `{"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":3}}`)
		ev("message_stop", `{"type":"message_stop"}`)
	}))
	return m
}

func (m *mockAnthropic) requests() []mockReq {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]mockReq(nil), m.seen...)
}

func cleanClaudeSession(t *testing.T, r *rig, cwd string) {
	r.b.Close()
	_ = DeleteSession(ProjectsDir(), cwd, r.b.SessionID())
	if real, err := filepath.EvalSymlinks(cwd); err == nil {
		_ = DeleteSession(ProjectsDir(), real, r.b.SessionID())
		_ = os.RemoveAll(filepath.Join(ProjectsDir(), EncodeProjectDir(real)))
	}
	_ = os.RemoveAll(filepath.Join(ProjectsDir(), EncodeProjectDir(cwd)))
}

func replyOf(r *rig) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var s string
	for _, it := range r.st.Items {
		if a, ok := it.(*core.Assistant); ok {
			s += a.Text
		}
	}
	return s
}

// TestRealClaudeViaMappedProvider 用真实 claude 进程验证「服务商映射 → 环境变量 → 请求」整条链路:
// 映射生效后,claude 应向服务商地址发请求,带上正确的鉴权头与映射的模型名;中途换服务商(CmdSetEnv)后下一条消息走新的地址与模型。
// 服务商用本机桩服务,不消耗任何额度、不需要真实密钥。
// 运行:MINI_REAL_CLAUDE=1 go test ./internal/claude -run ViaMappedProvider -v
func TestRealClaudeViaMappedProvider(t *testing.T) {
	if os.Getenv("MINI_REAL_CLAUDE") != "1" {
		t.Skip("设置 MINI_REAL_CLAUDE=1 才跑真实 claude")
	}
	if _, ok := FindClaude(""); !ok {
		t.Skip("本机没有 claude")
	}
	mockA := newMockAnthropic("来自A家的回复")
	defer mockA.srv.Close()
	mockB := newMockAnthropic("来自B家的回复")
	defer mockB.srv.Close()

	store := claudecfg.Store{Providers: []claudecfg.Provider{
		{ID: "a", Name: "A家", BaseURL: mockA.srv.URL + "/anthropic", Models: []string{"a-chat", "a-think"}},
		{ID: "b", Name: "B家", BaseURL: mockB.srv.URL, Auth: claudecfg.AuthXAPIKey, Models: []string{"b-1"}},
	}}
	keys := map[string]string{"a": "KEY-A", "b": "KEY-B"}
	envOf := func() []core.EnvVar {
		var out []core.EnvVar
		for _, e := range store.Env(keys) {
			out = append(out, core.EnvVar{Key: e.Key, Value: e.Value})
		}
		return out
	}

	store.Activate("a", "a-chat")
	mode := core.ModePlan
	cwd := t.TempDir()
	r := startRealRig(t, Config{Mode: &mode, Cwd: cwd, Env: envOf()})
	t.Cleanup(func() { cleanClaudeSession(t, r, cwd) })

	// 1) 映射生效:请求落到 A 家,Bearer + 映射的模型
	r.b.Send(core.CmdPrompt{Text: "说点什么"})
	r.wait("第一个回合结束", func() bool { return turnsEnded(r) == 1 })
	reqs := mockA.requests()
	if len(reqs) == 0 {
		t.Fatalf("A 家没有收到任何请求(环境变量没有生效?)。回复=%q", replyOf(r))
	}
	main := reqs[0]
	for _, q := range reqs {
		if q.Auth != "Bearer KEY-A" || q.APIKey != "" {
			t.Errorf("A 家鉴权头不对:%+v", q)
		}
		if !strings.HasPrefix(q.Model, "a-") {
			t.Errorf("发给 A 家的模型名应来自映射:%q", q.Model)
		}
	}
	t.Logf("A 家共收到 %d 个请求,首个模型=%s", len(reqs), main.Model)
	if got := replyOf(r); !strings.Contains(got, "来自A家的回复") {
		t.Errorf("应显示 A 家桩的回复:%q", got)
	}

	// 2) 切到 B 家(x-api-key 鉴权):下一条消息走新地址与新模型,会话继续
	store.Activate("b", "b-1")
	r.b.Send(core.CmdSetEnv{Env: envOf()})
	r.b.Send(core.CmdPrompt{Text: "再说点什么"})
	r.wait("第二个回合结束", func() bool { return turnsEnded(r) == 2 })
	breqs := mockB.requests()
	if len(breqs) == 0 {
		r.mu.Lock()
		for _, e := range r.evs {
			if n, ok := e.(core.EvNotice); ok {
				t.Logf("提示:%v %s", n.Level, n.Text)
			}
			if te, ok := e.(core.EvTurnEnded); ok {
				t.Logf("回合结束:%v", te.Reason)
			}
		}
		r.mu.Unlock()
		t.Fatalf("切换后 B 家没有收到请求;回复=%q", replyOf(r))
	}
	for _, q := range breqs {
		if q.APIKey != "KEY-B" || q.Auth != "" {
			t.Errorf("B 家应使用 x-api-key 鉴权:%+v", q)
		}
		if q.Model != "b-1" {
			t.Errorf("发给 B 家的模型名应为 b-1:%q", q.Model)
		}
	}
	if got := replyOf(r); !strings.Contains(got, "来自B家的回复") {
		t.Errorf("应显示 B 家桩的回复:%q", got)
	}
	// 切换后不能再有请求落到 A 家
	if n := len(mockA.requests()); n != len(reqs) {
		t.Errorf("切换后 A 家又收到了 %d 个请求", n-len(reqs))
	}
}

// startRealRig 与 startRig 相同,但不把 ProjectsDir 换成临时目录:
// 真实应用读写真实的 ~/.claude/projects,切换环境后「会话已落盘就 --resume」依赖这一点。
func startRealRig(t *testing.T, cfg Config) *rig {
	t.Helper()
	r := &rig{t: t, st: core.NewState()}
	r.cond = sync.NewCond(&r.mu)
	r.b = New(cfg)
	r.b.Start(func(e core.Event) {
		r.mu.Lock()
		r.st.Apply(e)
		r.evs = append(r.evs, e)
		r.cond.Broadcast()
		r.mu.Unlock()
	})
	t.Cleanup(r.b.Close)
	return r
}
