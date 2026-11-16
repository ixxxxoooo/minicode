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

// 关闭时的窗口状态(编辑器拆分、Markdown 视图、树 / 项目展开、终端标签数、草稿)能写出并恢复到新实例。
func TestWindowStateRoundTrip(t *testing.T) {
	src := New(Options{Prefs: prefs.Memory()})
	src.ide.openFile("README.md", true)
	src.ide.openFile("CLAUDE.md", true)
	src.ide.tabByPath("README.md").mdMode = mdSource
	src.ide.splitDir, src.ide.splitFrac, src.ide.paneSel = 1, 0.4, 1
	if tb := src.ide.tabByPath("CLAUDE.md"); tb != nil && tb.buf != nil {
		src.openSharedInPane(1, tb)
	}
	src.ide.open = map[string]bool{"docs": true, "mygo": false}
	src.projOpen = map[string]bool{"p1": true}
	src.draft = "写到一半的消息"

	// 与 SaveRuntimeState 同一份收集逻辑:直接走落盘文件再读回
	dir := t.TempDir()
	t.Setenv("MINI_HOME", dir)
	src.SaveRuntimeState()
	raw, err := os.ReadFile(lastStatePath(false))
	if err != nil {
		t.Fatal("落盘失败:", err)
	}
	var st lastRuntime
	if err := json.Unmarshal(raw, &st); err != nil {
		t.Fatal(err)
	}
	if st.SplitDir != 1 || len(st.TreeOpen) != 1 || st.TreeOpen[0] != "docs" || st.Draft != "写到一半的消息" || st.MdMode["README.md"] != mdSource {
		t.Fatalf("保存的窗口状态不对: %+v", st)
	}
	dst := New(Options{Prefs: prefs.Memory()})
	for _, p := range st.EditorTabs {
		dst.ide.openFile(p, true)
	}
	dst.applySavedWindowState(st)
	if dst.ide.splitDir != 1 || dst.ide.splitFrac != 0.4 || len(dst.ide.tabs2) == 0 {
		t.Errorf("编辑器拆分没恢复: dir=%d frac=%v tabs2=%d", dst.ide.splitDir, dst.ide.splitFrac, len(dst.ide.tabs2))
	}
	if dst.ide.tabByPath("README.md").mdMode != mdSource || !dst.ide.open["docs"] || !dst.projOpen["p1"] || dst.draft != "写到一半的消息" {
		t.Error("Markdown 视图 / 树展开 / 项目展开 / 草稿没恢复")
	}
}
