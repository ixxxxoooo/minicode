package claude

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// TestRealClaude 对本机真实的 claude 做一次端到端对话(会产生极少量 token 消耗)。
// 默认跳过:MINI_REAL_CLAUDE=1 go test ./internal/claude -run TestRealClaude -v
func TestRealClaude(t *testing.T) {
	if os.Getenv("MINI_REAL_CLAUDE") != "1" {
		t.Skip("设置 MINI_REAL_CLAUDE=1 才跑真实 claude")
	}
	if _, ok := FindClaude(""); !ok {
		t.Skip("本机没有 claude")
	}
	mode := core.ModePlan // 计划模式:不会改任何文件
	r := newRealRig(t, Config{Mode: &mode})
	r.b.Send(core.CmdPrompt{Text: "只回复两个字母:ok,不要调用任何工具。"})
	r.wait("回合结束", func() bool { return turnsEnded(r) == 1 })
	r.mu.Lock()
	defer r.mu.Unlock()
	var reply string
	for _, it := range r.st.Items {
		if a, ok := it.(*core.Assistant); ok {
			reply += a.Text
		}
	}
	t.Logf("模型列表 %d 个,当前模型 %s,模式 %s", len(r.st.Models), r.st.Model, r.st.Mode.Label())
	t.Logf("回复:%q", reply)
	for _, e := range r.evs {
		if ts, ok := e.(core.EvTurnStats); ok {
			t.Logf("回合统计:%s | %s", ts.Stats.ShortText(), ts.Stats.ModelLabel())
		}
		if n, ok := e.(core.EvNotice); ok {
			t.Logf("提示:%v %s", n.Level, n.Text)
		}
	}
	if strings.Contains(reply, "limit") { // 账号限额用完:协议链路已验证(握手、模式、限额提示),没有真实回复可比
		t.Skip("账号触及限额,跳过回复内容检查")
	}
	if !strings.Contains(strings.ToLower(reply), "ok") {
		t.Errorf("应得到包含 ok 的回复:%q", reply)
	}
	if r.st.Mode != core.ModePlan {
		t.Errorf("应保持计划模式:%v", r.st.Mode)
	}
}

// TestRealHistory 扫描本机真实的 ~/.claude/projects 并回放其中最大的会话,检查不 panic、耗时合理。
func TestRealHistory(t *testing.T) {
	if os.Getenv("MINI_REAL_CLAUDE") != "1" {
		t.Skip("设置 MINI_REAL_CLAUDE=1 才扫真实历史")
	}
	t0 := time.Now()
	ix := BuildIndex(ProjectsDir(), time.Now())
	t.Logf("索引 %d 个项目,耗时 %v", len(ix.Projects), time.Since(t0).Round(time.Millisecond))
	var best Located
	var bestSize int64
	for _, p := range ix.Projects {
		for _, s := range p.Sessions {
			loc, _ := ix.Locate(s.ID)
			if st, err := os.Stat(loc.Path); err == nil && st.Size() > bestSize {
				best, bestSize = loc, st.Size()
			}
		}
	}
	if best.Path == "" {
		t.Skip("没有历史会话")
	}
	t0 = time.Now()
	evs, err := LoadSession(best.Path, best.Cwd)
	if err != nil {
		t.Fatal(err)
	}
	st := core.NewState()
	for _, e := range evs {
		st.Apply(e)
	}
	t.Logf("回放最大会话(%.1f MB):%d 个事件 → %d 个条目,耗时 %v", float64(bestSize)/1e6, len(evs), len(st.Items), time.Since(t0).Round(time.Millisecond))
}
