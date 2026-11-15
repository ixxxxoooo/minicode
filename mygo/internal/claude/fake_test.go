package claude

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// TestMain 让测试二进制兼任「假 claude」:后端用它当 claude 可执行文件启动(环境变量 FAKE_CLAUDE=1),
// 按 stream-json 协议应答,这样不需要真实的 claude 也能验证进程编排。
func TestMain(m *testing.M) {
	if os.Getenv("FAKE_CLAUDE") == "1" {
		fakeClaude()
		return
	}
	os.Exit(m.Run())
}

func out(v any) {
	b, _ := json.Marshal(v)
	fmt.Println(string(b))
}

func fakeClaude() {
	if os.Getenv("FAKE_MODE") == "silent" { // 握手永不应答
		time.Sleep(time.Minute)
		return
	}
	sc := bufio.NewScanner(os.Stdin)
	sc.Buffer(make([]byte, 1<<20), 16<<20)
	turn := 0
	for sc.Scan() {
		var v map[string]any
		if json.Unmarshal(sc.Bytes(), &v) != nil {
			continue
		}
		switch v["type"] {
		case "control_request":
			req := v["request"].(map[string]any)
			body := map[string]any{}
			if req["subtype"] == "initialize" {
				body = map[string]any{"models": []any{map[string]any{"value": "sonnet", "displayName": "Sonnet 5.5", "description": "d"}}, "current_permission_mode": "default"}
			}
			out(map[string]any{"type": "control_response", "response": map[string]any{"subtype": "success", "request_id": v["request_id"], "response": body}})
		case "user":
			turn++
			content := v["message"].(map[string]any)["content"].([]any)
			text := content[len(content)-1].(map[string]any)["text"].(string)
			if text == "die" {
				fmt.Fprintln(os.Stderr, "boom")
				os.Exit(3)
			}
			reply := "你好"
			if text == "tool" {
				fakeTool()
				reply = "完成"
			}
			fakeText(turn, reply)
		}
	}
}

func fakeText(turn int, reply string) {
	id := fmt.Sprintf("m%d", turn)
	ev := func(e map[string]any) { out(map[string]any{"type": "stream_event", "event": e}) }
	ev(map[string]any{"type": "message_start", "message": map[string]any{"id": id, "usage": map[string]any{"input_tokens": 10, "output_tokens": 1}}})
	ev(map[string]any{"type": "content_block_start", "index": 0, "content_block": map[string]any{"type": "text"}})
	ev(map[string]any{"type": "content_block_delta", "index": 0, "delta": map[string]any{"type": "text_delta", "text": reply}})
	ev(map[string]any{"type": "content_block_stop", "index": 0})
	out(map[string]any{"type": "assistant", "message": map[string]any{"id": id, "content": []any{map[string]any{"type": "text", "text": reply}}}})
	out(map[string]any{"type": "result", "subtype": "success", "is_error": false, "total_cost_usd": 0.01, "num_turns": 1, "stop_reason": "end_turn",
		"usage": map[string]any{"input_tokens": 10, "output_tokens": 5}})
}

// fakeTool 模拟一次需要权限的 Bash:发 tool_use → can_use_tool,等应答后回 tool_result。
func fakeTool() {
	out(map[string]any{"type": "assistant", "message": map[string]any{"id": "mt", "content": []any{
		map[string]any{"type": "tool_use", "id": "t1", "name": "Bash", "input": map[string]any{"command": "echo hi", "description": "打招呼"}}}}})
	out(map[string]any{"type": "control_request", "request_id": "cr1", "request": map[string]any{
		"subtype": "can_use_tool", "tool_name": "Bash", "tool_use_id": "t1", "input": map[string]any{"command": "echo hi"}}})
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		var v map[string]any
		if json.Unmarshal(sc.Bytes(), &v) == nil && v["type"] == "control_response" {
			break
		}
	}
	out(map[string]any{"type": "user", "message": map[string]any{"content": []any{
		map[string]any{"type": "tool_result", "tool_use_id": "t1", "content": "hi\n"}}},
		"tool_use_result": map[string]any{"stdout": "hi\n", "stderr": ""}})
}

// ---------- 测试夹具 ----------

type rig struct {
	t    *testing.T
	b    *Backend
	mu   sync.Mutex
	st   *core.State
	evs  []core.Event
	cond *sync.Cond
}

func newRig(t *testing.T, cfg Config, env ...string) *rig {
	t.Helper()
	t.Setenv("FAKE_CLAUDE", "1")
	for _, e := range env {
		k, v, _ := strings.Cut(e, "=")
		t.Setenv(k, v)
	}
	exe, _ := os.Executable()
	cfg.ClaudePath = exe
	return startRig(t, cfg)
}

// newRealRig 用真实的 claude(不设假 claude 环境变量、不指定路径)。
func newRealRig(t *testing.T, cfg Config) *rig { return startRig(t, cfg) }

func startRig(t *testing.T, cfg Config) *rig {
	t.Helper()
	if cfg.Cwd == "" {
		cfg.Cwd = t.TempDir()
	}
	cfg.ProjectsDir = t.TempDir()
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

// wait 等条件成立(在锁内判断),超时则失败。
func (r *rig) wait(what string, cond func() bool) {
	r.t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	timer := time.AfterFunc(10*time.Second, func() { r.mu.Lock(); r.cond.Broadcast(); r.mu.Unlock() })
	defer timer.Stop()
	r.mu.Lock()
	defer r.mu.Unlock()
	for !cond() {
		if time.Now().After(deadline) {
			r.t.Fatalf("等待超时:%s\n事件:%v", what, r.evs)
		}
		r.cond.Wait()
	}
}

func (r *rig) count(f func(core.Event) bool) int {
	n := 0
	for _, e := range r.evs {
		if f(e) {
			n++
		}
	}
	return n
}

func turnsEnded(r *rig) int {
	return r.count(func(e core.Event) bool { _, ok := e.(core.EvTurnEnded); return ok })
}

func TestSimpleChat(t *testing.T) {
	r := newRig(t, Config{})
	r.b.Send(core.CmdPrompt{Text: "hi"})
	r.wait("回合结束", func() bool { return turnsEnded(r) == 1 })
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.st.Items) != 2 {
		t.Fatalf("应有 用户 + 助手 两条,实际 %d:%v", len(r.st.Items), r.st.Items)
	}
	if u, ok := r.st.Items[0].(*core.User); !ok || u.Text != "hi" {
		t.Fatalf("第一条应是用户消息:%v", r.st.Items[0])
	}
	if a, ok := r.st.Items[1].(*core.Assistant); !ok || a.Text != "你好" || a.Streaming {
		t.Fatalf("助手正文应去重为一条且已结束:%#v", r.st.Items[1])
	}
	if r.st.Phase != core.PhaseIdle {
		t.Fatalf("回合结束后应空闲:%v", r.st.Phase)
	}
	if len(r.st.Models) != 1 || r.st.Models[0].ID != "sonnet" {
		t.Fatalf("模型列表应来自握手应答:%v", r.st.Models)
	}
}

func TestPermissionFlow(t *testing.T) {
	r := newRig(t, Config{})
	r.b.Send(core.CmdPrompt{Text: "tool"})
	r.wait("权限请求", func() bool {
		for _, it := range r.st.Items {
			if p, ok := it.(*core.PermissionRequest); ok && p.Resolved == "" {
				return true
			}
		}
		return false
	})
	r.b.Send(core.CmdResolvePermission{ID: "t1", Option: "allow_once"})
	r.wait("回合结束", func() bool { return turnsEnded(r) == 1 })
	r.mu.Lock()
	defer r.mu.Unlock()
	var perm *core.PermissionRequest
	for _, it := range r.st.Items {
		if p, ok := it.(*core.PermissionRequest); ok {
			perm = p
		}
	}
	if perm == nil || perm.Resolved != "allow_once" {
		t.Fatalf("权限请求应已按允许一次处理:%#v", perm)
	}
	if perm.Tool.Status != core.ToolCompleted {
		t.Fatalf("工具应已完成:%v", perm.Tool.Status)
	}
	var term *core.TerminalBlock
	for _, c := range perm.Tool.Content {
		if tb, ok := c.(*core.TerminalBlock); ok {
			term = tb
		}
	}
	if term == nil || !strings.Contains(term.Output, "hi") || term.ExitCode == nil || *term.ExitCode != 0 {
		t.Fatalf("应带终端输出与退出码 0:%#v", term)
	}
}

func TestProcessDiesAndRestarts(t *testing.T) {
	r := newRig(t, Config{})
	r.b.Send(core.CmdPrompt{Text: "die"})
	r.wait("异常退出收尾", func() bool { return turnsEnded(r) == 1 })
	r.mu.Lock()
	var gotNotice bool
	for _, e := range r.evs {
		if n, ok := e.(core.EvNotice); ok && n.Level == core.NoticeError && strings.Contains(n.Text, "退出码 3") && strings.Contains(n.Text, "boom") {
			gotNotice = true
		}
	}
	reason := core.StopEndTurn
	for _, e := range r.evs {
		if te, ok := e.(core.EvTurnEnded); ok {
			reason = te.Reason
		}
	}
	r.mu.Unlock()
	if !gotNotice || reason != core.StopError {
		t.Fatalf("应提示退出码与 stderr 并以错误结束回合:notice=%v reason=%v", gotNotice, reason)
	}
	// 进程退出后句柄依然有效:再发一条提示应重新启动并正常结束
	r.b.Send(core.CmdPrompt{Text: "hi"})
	r.wait("重启后回合结束", func() bool { return turnsEnded(r) == 2 })
}

func TestInitTimeout(t *testing.T) {
	r := newRig(t, Config{InitTimeout: 200 * time.Millisecond}, "FAKE_MODE=silent")
	r.b.Send(core.CmdPrompt{Text: "hi"})
	r.wait("握手超时收尾", func() bool { return turnsEnded(r) == 1 })
	r.mu.Lock()
	defer r.mu.Unlock()
	found := false
	for _, e := range r.evs {
		if n, ok := e.(core.EvNotice); ok && strings.Contains(n.Text, "启动超时") {
			found = true
		}
	}
	if !found {
		t.Fatalf("应提示启动超时:%v", r.evs)
	}
}

func TestClaudeNotFound(t *testing.T) {
	t.Setenv("PATH", "/nonexistent")
	t.Setenv("MINI_CLAUDE_PATH", "")
	t.Setenv("HOME", t.TempDir())
	var mu sync.Mutex
	var evs []core.Event
	b := New(Config{Cwd: t.TempDir(), ClaudePath: "/nonexistent/claude"})
	b.Start(func(e core.Event) { mu.Lock(); evs = append(evs, e); mu.Unlock() })
	defer b.Close()
	b.Send(core.CmdPrompt{Text: "hi"})
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := 0
		for _, e := range evs {
			if _, ok := e.(core.EvTurnEnded); ok {
				n++
			}
		}
		mu.Unlock()
		if n == 1 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("找不到 claude 时应提示并结束回合:%v", evs)
}
