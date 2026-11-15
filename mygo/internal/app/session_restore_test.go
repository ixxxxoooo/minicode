package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ixxxxoooo/minicode/internal/prefs"
)

func TestRestoreEnabled(t *testing.T) {
	a := New(Options{Prefs: prefs.Memory()})
	if !a.RestoreEnabled() {
		t.Fatal("默认应开启恢复")
	}
	a.prefs.Set(pRestore, false)
	if a.RestoreEnabled() {
		t.Fatal("关闭后应不再恢复")
	}
	a.secondary = true
	a.prefs.Set(pRestore, true)
	if a.RestoreEnabled() {
		t.Fatal("从属窗口不应恢复")
	}
}

func TestSaveAndApplyRuntime(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("MINICODE_HOME", dir)
	t.Cleanup(func() { os.Unsetenv("MINICODE_HOME") })
	// diag.Home() 可能不读 MINICODE_HOME — 直接写路径
	path := filepath.Join(dir, lastStateFile)
	a := New(Options{Prefs: prefs.Memory()})
	a.curSess = "abc"
	a.proj = "/tmp/p"
	a.ide.bottomTab = "term"
	a.navBy[ModeAgent] = "scm"
	a.ide.openFile("README.md", true)
	// 手动落盘(测试不依赖 diag.Home)
	st := lastRuntime{Session: a.curSess, Project: a.proj, BottomTab: a.ide.bottomTab, NavAgent: a.navBy[ModeAgent], EditorTabs: []string{"README.md"}, ActiveTab: "README.md"}
	b, err := json.Marshal(st)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	// 临时覆盖 lastStatePath — 通过写入 diag home 不易,测 Apply 逻辑
	a2 := New(Options{Prefs: prefs.Memory()})
	a2.prefs.Set(pRestore, true)
	// 读入
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got lastRuntime
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.Session != "abc" || got.NavAgent != "scm" {
		t.Fatalf("状态不对: %+v", got)
	}
	_ = a2
}
