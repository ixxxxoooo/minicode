package app

import (
	"testing"

	"github.com/egoist/mygo/ui/editor"
)

func TestFoldMemRestoreAfterReopen(t *testing.T) {
	a := New(Options{Mode: ModeIDE})
	a.ide.openFile("src/router.rs", true)
	tb := a.ide.cur()
	a.foldRun(tb, editor.FoldAll, "全部折叠")
	if n := len(tb.st.Folded()); n == 0 {
		t.Fatal("应有折叠")
	}
	a.ide.saveFolds(tb)
	id := tb.id
	a.ide.closeTab(id)
	if _, ok := a.ide.fileFolds["src/router.rs"]; !ok {
		t.Fatal("关标签后应记住折叠")
	}
	a.ide.openFile("src/router.rs", true)
	tb2 := a.ide.cur()
	if len(tb2.st.Folded()) == 0 {
		t.Fatal("重新打开应恢复折叠")
	}
}

func TestUnfoldLineForNavRevealsFoldedLine(t *testing.T) {
	a := New(Options{Mode: ModeIDE})
	a.ide.openFile("src/router.rs", true)
	tb := a.ide.cur()
	if !tb.st.Run(editor.FoldAll) {
		t.Skip("该演示文件没有可折叠块")
	}
	line := 16
	if line >= tb.buf.LineCount() {
		line = tb.buf.LineCount() / 2
	}
	unfoldLineForNav(tb, line)
	if tb.st.CursorPos().Line != line {
		t.Fatalf("光标应在行 %d,实际 %+v", line, tb.st.CursorPos())
	}
}
