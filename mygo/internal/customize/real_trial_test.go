package customize

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ixxxxoooo/minicode/internal/claude"
)

func realClaude(t *testing.T) {
	t.Helper()
	if os.Getenv("MINI_REAL_CLAUDE") != "1" {
		t.Skip("设置 MINI_REAL_CLAUDE=1 才跑真实 claude 联调")
	}
	if _, ok := claude.FindClaude(""); !ok {
		t.Skip("未安装 claude")
	}
}

func TestRealCommandTrialDeepSeek(t *testing.T) {
	realClaude(t)
	proj := t.TempDir()
	out, err := claude.RunPrompt(proj, "只回复一个词:你好", "plan", 90*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out) == "" {
		t.Fatal("试运行输出为空")
	}
}

func TestRealMCPListTempHome(t *testing.T) {
	realClaude(t)
	home := t.TempDir()
	proj := filepath.Join(t.TempDir(), "proj")
	t.Setenv("HOME", home)
	_ = os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	_ = os.WriteFile(filepath.Join(home, ".claude.json"), []byte(`{"mcpServers":{"noop":{"command":"true"}}}`), 0o600)
	if _, err := RunMCPList(proj); err != nil {
		t.Log("mcp list:", err)
	}
}