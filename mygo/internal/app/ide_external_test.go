package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui/editor"
	"github.com/ixxxxoooo/minicode/internal/lsp"
)

func TestJumpToExternalReadOnly(t *testing.T) {
	dir := t.TempDir()
	proj := filepath.Join(dir, "proj")
	extDir := filepath.Join(dir, "stdlib")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(extDir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := "package builtin\n\nfunc println() {}\n"
	extFile := filepath.Join(extDir, "builtin.go")
	if err := os.WriteFile(extFile, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(proj, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644)

	a, tt := newTestApp(t)
	a.SetProject(proj)
	a.SetMode(ModeIDE)
	a.ide.openFile("main.go", true)
	a.drainUI()

	r := lsp.Range{Start: lsp.Position{Line: 2, Character: 0}}
	a.finishOpenDiskReadOnly(extFile, src, r)
	tt.Frame()
	a.drainUI()

	tb := a.ide.cur()
	if tb == nil || tb.path != extFile {
		t.Fatalf("应打开外部文件,得 %q", tb)
	}
	if !tb.diskRO || tb.st == nil || !tb.st.ReadOnly {
		t.Fatalf("应是只读标签:%+v", tb)
	}
	if p := tb.st.CursorPos(); p.Line != 2 {
		t.Fatalf("应跳到第 3 行,得 %+v", p)
	}
}

func TestJumpToLSPExternal(t *testing.T) {
	dir := t.TempDir()
	proj := filepath.Join(dir, "proj")
	extFile := filepath.Join(dir, "ext.go")
	_ = os.MkdirAll(proj, 0o755)
	_ = os.WriteFile(extFile, []byte("func Helper() {}\n"), 0o644)
	_ = os.WriteFile(filepath.Join(proj, "a.go"), []byte("package p\n"), 0o644)

	a, tt := newTestApp(t)
	a.SetProject(proj)
	a.SetMode(ModeIDE)
	a.ide.openFile("a.go", true)

	a.jumpToLSP(extFile, lsp.Range{Start: lsp.Position{Line: 0, Character: 5}})
	a.drainUI()
	tt.Frame()
	a.drainUI()

	tb := a.ide.tab(extFile)
	if tb == nil || !tb.diskRO {
		t.Fatal("jumpToLSP 应只读打开工作区外文件")
	}
}

func TestExternalSaveBlocked(t *testing.T) {
	a, _ := newTestApp(t)
	tb := &ideTab{
		path:   "/tmp/x.go",
		diskRO: true,
		buf:    editor.NewBuffer("x"),
		saved:  "",
		st:     &editor.State{Buffer: editor.NewBuffer("x"), ReadOnly: true},
	}
	a.ide.tabs = []*ideTab{tb}
	a.ide.active = tb.id
	if err := a.ide.save(); err == nil {
		t.Fatal("只读外部标签应拒绝保存")
	}
}
