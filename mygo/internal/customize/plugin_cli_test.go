package customize

import (
	"testing"

	"github.com/ixxxxoooo/minicode/internal/claude"
)

// noClaude 让 FindClaude 找不到 claude(PATH 与 HOME 都换成空目录);
// 本机若仍能从固定的绝对路径找到,就跳过——否则测试会真的去执行 claude(联网安装插件)。
func noClaude(t *testing.T) {
	t.Helper()
	t.Setenv("PATH", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	if p, ok := claude.FindClaude(""); ok {
		t.Skipf("claude 仍可从 %s 找到,跳过「找不到 claude」的测试", p)
	}
}

// 回归:找不到 claude 时这几个入口原先 Code 留 0,调用方按 Code != 0 判失败,
// 于是没装 claude 时安装 / 卸载 / 启停插件、添加市场都被当成成功(CI 机器上没有 claude,测试由此暴露)。
func TestPluginCommandsFailWithoutCLI(t *testing.T) {
	noClaude(t)
	for name, res := range map[string]PluginInstallResult{
		"安装":   RunPluginInstall("nope", "claude-plugins-official"),
		"卸载":   RunPluginUninstall("nope"),
		"启停":   RunPluginToggle("nope", true, ""),
		"添加市场": RunMarketplaceAdd("owner/repo"),
	} {
		if res.Code == 0 {
			t.Errorf("%s:找不到 claude 时应失败(Code != 0),得到 %+v", name, res)
		}
		if res.Err == "" {
			t.Errorf("%s:应有可行动的错误说明", name)
		}
	}
}

func TestMarketplaceListOffline(t *testing.T) {
	noClaude(t)
	if mk := RunMarketplaceList(); len(mk) != 0 {
		t.Fatal("离线应返回 nil 由 OpenDisk 兜底")
	}
}
