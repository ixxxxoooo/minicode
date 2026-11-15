package claudecfg

import (
	"os"
	"path/filepath"
	"testing"
)

// 内容没变时重复同步:不重写文件、不产生 .mini-backup(否则每次开会话都弄脏用户仓库)。
func TestSyncSessionPrefsNoopNoBackup(t *testing.T) {
	proj := t.TempDir()
	on := true
	sp := SessionPrefs{Layer: LayerProject, Project: proj, AlwaysThinking: &on, DefaultMode: "default"}
	if SettingsFileExists(LayerProject, proj) {
		t.Fatal("前提:文件还不存在")
	}
	if err := SyncSessionPrefs(sp); err != nil {
		t.Fatal(err)
	}
	if !SettingsFileExists(LayerProject, proj) {
		t.Fatal("首次同步应创建文件")
	}
	for i := 0; i < 3; i++ {
		if err := SyncSessionPrefs(sp); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(filepath.Join(proj, ".claude", "settings.json.mini-backup")); err == nil {
		t.Fatal("内容没变不应产生备份文件")
	}
	// 真有改动时才备份
	on = false
	if err := SyncSessionPrefs(SessionPrefs{Layer: LayerProject, Project: proj, AlwaysThinking: &on}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(proj, ".claude", "settings.json.mini-backup")); err != nil {
		t.Fatal("内容变化应备份上一份")
	}
}
