package customize

import (
	"testing"

	"github.com/ixxxxoooo/minicode/internal/claude"
)

func TestPluginInstallMissingCLI(t *testing.T) {
	// 找不到 claude 时应返回可行动错误,不把退出码当成成功。
	t.Setenv("PATH", t.TempDir())
	res := RunPluginInstall("nope", "claude-plugins-official")
	if res.Code == 0 {
		t.Fatal("应失败")
	}
	if res.Err == "" {
		t.Fatal("应有错误说明")
	}
}

func TestMarketplaceListOffline(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if _, ok := claude.FindClaude(""); ok {
		t.Skip("claude 仍可从绝对路径找到,跳过离线兜底测试")
	}
	if mk := RunMarketplaceList(); len(mk) != 0 {
		t.Fatal("离线应返回 nil 由 OpenDisk 兜底")
	}
}
