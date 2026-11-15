package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// TestRealAppendSystemPrompt 用 DeepSeek 验证 --append-system-prompt 生效(前缀【测试】)。
// MINI_REAL_CLAUDE=1 + scripts/with-deepseek.sh
func TestRealAppendSystemPrompt(t *testing.T) {
	if os.Getenv("MINI_REAL_CLAUDE") != "1" {
		t.Skip("设置 MINI_REAL_CLAUDE=1 才跑真实 claude")
	}
	if _, ok := FindClaude(""); !ok {
		t.Skip("本机没有 claude")
	}
	sys := "每次回复都以【测试】开头,只回复一句确认,不要调用任何工具。"
	mode := core.ModePlan
	cwd := t.TempDir()
	r := newRealRig(t, Config{
		Cwd:                cwd,
		AppendSystemPrompt: sys,
		Mode:               &mode,
		InitTimeout:        120 * time.Second,
	})
	t.Cleanup(func() {
		r.b.Close()
		_ = DeleteSession(ProjectsDir(), cwd, r.b.SessionID())
		_ = os.RemoveAll(filepath.Join(ProjectsDir(), EncodeProjectDir(cwd)))
	})
	r.b.Send(core.CmdPrompt{Text: "你好,请确认收到。"})
	r.wait("回合结束", func() bool { return turnsEnded(r) == 1 })
	r.mu.Lock()
	defer r.mu.Unlock()
	var reply string
	for _, it := range r.st.Items {
		if as, ok := it.(*core.Assistant); ok {
			reply += as.Text
		}
	}
	t.Logf("回复:%q", reply)
	if strings.Contains(strings.ToLower(reply), "limit") {
		t.Skip("账号限额,跳过内容检查")
	}
	if !strings.Contains(reply, "【测试】") {
		t.Errorf("应包含系统提示前缀【测试】,得:%q", reply)
	}
}
