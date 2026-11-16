package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ixxxxoooo/minicode/internal/lsp"
)

// 启动中语言服务会话的 work 行(含 $/progress 百分比)应出现在状态栏。
func TestLspStatusBarShowsProgressPct(t *testing.T) {
	dir := t.TempDir()
	dir, _ = filepath.EvalSymlinks(dir)
	_ = os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644)
	a, tt := newTestApp(t)
	a.SetProject(dir)
	a.SetMode(ModeIDE)
	spec := lsp.ForFile("main.go")
	if spec == nil {
		t.Fatal("应有 gopls 规格")
	}
	abs := filepath.Join(dir, "main.go")
	root := spec.Root(abs, dir)
	key := root + "|" + spec.ID
	a.lspH.sess = map[string]*lspSess{
		key: {spec: spec, root: root, state: lspStarting, work: "索引中 42%", docs: map[string]*lspDoc{}},
	}
	a.ide.openFile("main.go", true)
	tt.Frame()
	if !tt.HasText("42%") {
		t.Fatal("状态栏启动中应显示索引进度百分比")
	}
}
