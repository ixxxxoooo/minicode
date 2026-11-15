package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ixxxxoooo/minicode/internal/projects"
)

// 用户从没改过相关偏好、项目里也没有 .claude/settings.json:开会话同步时不能凭空创建它。
func TestSyncClaudeSettingsDoesNotCreateInProject(t *testing.T) {
	a, _ := newTestApp(t)
	proj := t.TempDir()
	a.proj = proj
	a.reg = projects.Open(filepath.Join(t.TempDir(), "workspaces.json")) // 非演示:registry 持久化
	a.SyncClaudeSettingsFromPrefs()
	if _, err := os.Stat(filepath.Join(proj, ".claude")); err == nil {
		t.Fatal("没有显式设置时不应在项目里创建 .claude/")
	}
}
