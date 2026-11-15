package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 真实模式的 Claude 配置摘要来自磁盘上的 settings.json,并按层统计规则数。
func TestRealCfgSummary(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	proj := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"permissions":{"allow":["Bash(git status)","Read"],"deny":["Bash(rm *)"]},"env":{"FOO":"x"}}`
	if err := os.WriteFile(filepath.Join(home, ".claude", "settings.json"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	a, _ := newTestApp(t)
	a.proj = proj
	lines := a.realCfgSummary() // 测试里 a.win == nil,后台任务同步执行
	if len(lines) == 0 {
		t.Fatal("摘要不应为空")
	}
	all := strings.Join(lines, "\n")
	for _, want := range []string{"用户层权限规则:允许 2 · 询问 0 · 拒绝 1", "环境变量:1 个"} {
		if !strings.Contains(all, want) {
			t.Errorf("摘要缺少 %q,实际:\n%s", want, all)
		}
	}
}
