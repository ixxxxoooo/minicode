package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/claude"
	"github.com/ixxxxoooo/minicode/internal/core"
)

// TestRealFlowIdleReply 真实 claude(可走 DeepSeek 映射)发一句短话,等回合结束并清理测试会话。
func TestRealFlowIdleReply(t *testing.T) {
	if os.Getenv("MINI_REAL_CLAUDE") != "1" {
		t.Skip("设置 MINI_REAL_CLAUDE=1 才跑真实 claude")
	}
	if _, ok := claude.FindClaude(""); !ok {
		t.Skip("本机没有 claude")
	}
	mode := core.ModePlan
	cwd := t.TempDir()
	be := claude.New(claude.Config{Cwd: cwd, Mode: &mode})
	t.Cleanup(func() {
		be.Close()
		_ = claude.DeleteSession(claude.ProjectsDir(), cwd, be.SessionID())
		if real, err := filepath.EvalSymlinks(cwd); err == nil {
			_ = claude.DeleteSession(claude.ProjectsDir(), real, be.SessionID())
			_ = os.RemoveAll(filepath.Join(claude.ProjectsDir(), claude.EncodeProjectDir(real)))
		}
		_ = os.RemoveAll(filepath.Join(claude.ProjectsDir(), claude.EncodeProjectDir(cwd)))
	})
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	a.SetProject(cwd)
	a.Attach(be)
	tt := ui.NewTester(a.View, 1280, 800)
	tt.Frame()
	a.send(core.CmdPrompt{Text: "只回复两个字母 ok,不要调用任何工具。"})
	deadline := time.Now().Add(120 * time.Second)
	var sawAssistant bool
	for time.Now().Before(deadline) {
		tt.Frame()
		for _, it := range a.st.Items {
			if as, ok := it.(*core.Assistant); ok && strings.Contains(strings.ToLower(as.Text), "ok") {
				sawAssistant = true
			}
		}
		if a.st.Phase == core.PhaseIdle && sawAssistant {
			break
		}
		time.Sleep(150 * time.Millisecond)
	}
	if !sawAssistant {
		t.Fatalf("应收到包含 ok 的助手回复,条目 %d 条,phase=%v", len(a.st.Items), a.st.Phase)
	}
}

// TestRealFlowQueueWhileRunning 回合进行中再发一条应进入排队。
func TestRealFlowQueueWhileRunning(t *testing.T) {
	if os.Getenv("MINI_REAL_CLAUDE") != "1" {
		t.Skip("设置 MINI_REAL_CLAUDE=1 才跑真实 claude")
	}
	if _, ok := claude.FindClaude(""); !ok {
		t.Skip("本机没有 claude")
	}
	mode := core.ModePlan
	cwd := t.TempDir()
	be := claude.New(claude.Config{Cwd: cwd, Mode: &mode})
	t.Cleanup(func() {
		be.Close()
		_ = claude.DeleteSession(claude.ProjectsDir(), cwd, be.SessionID())
	})
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	a.SetProject(cwd)
	a.Attach(be)
	tt := ui.NewTester(a.View, 1280, 800)
	tt.Frame()
	a.send(core.CmdPrompt{Text: "先想 3 秒,然后用一句话介绍你自己,不要工具。"})
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		tt.Frame()
		if a.st.Phase == core.PhaseRunning {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if a.st.Phase != core.PhaseRunning {
		t.Skip("模型过快结束,无法测排队")
	}
	a.submitKey("排队:第二条消息", false)
	tt.Frame()
	if a.q.len() != 1 {
		t.Fatalf("运行中发送应排队 1 条,实际 %d", a.q.len())
	}
	if !tt.HasText("待发送") {
		t.Fatal("排队卡应可见")
	}
	be.Close()
}
