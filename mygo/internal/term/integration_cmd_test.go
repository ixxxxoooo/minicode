package term

import (
	"strings"
	"testing"
)

// 注入命令必须以清屏收尾,否则用户会在终端里看到那行 source 的回显。
func TestIntegrationCmdQuiet(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	for _, sh := range []string{"/bin/zsh", "/bin/bash"} {
		cmd := IntegrationCmd(sh)
		if !strings.HasPrefix(cmd, "source ") || !strings.HasSuffix(cmd, "\n") {
			t.Errorf("%s: 命令格式不对: %q", sh, cmd)
		}
		if !strings.Contains(cmd, `\033[2J`) || !strings.Contains(cmd, `\033[3J`) {
			t.Errorf("%s: 缺少清屏转义: %q", sh, cmd)
		}
	}
}
