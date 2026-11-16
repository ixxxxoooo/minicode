package app

import (
	"testing"

	"github.com/egoist/mygo/ui/editor"
)

func TestFoldRunAllAndUnfold(t *testing.T) {
	a := New(Options{Mode: ModeIDE})
	a.ide.openFile("src/router.rs", true)
	tb := a.ide.cur()
	// 先让编辑器布局一次,折叠区间才算得出
	a.foldRun(tb, editor.FoldAll, "全部折叠")
	if why := a.foldWhy(tb); why != "" {
		t.Fatalf("默认应可折叠,实际原因 %q", why)
	}
	a.foldRun(tb, editor.UnfoldAll, "全部展开")
	if n := len(tb.st.Folded()); n != 0 {
		t.Fatalf("全部展开后仍有 %d 处折叠", n)
	}
}

func TestFoldDisabledWhenOff(t *testing.T) {
	a := New(Options{Mode: ModeIDE})
	a.ide.openFile("src/router.rs", true)
	tb := a.ide.cur()
	tb.st.NoFolding = true
	if why := a.foldWhy(tb); why == "" {
		t.Fatal("关闭折叠时应给出置灰原因")
	}
	a.toast = ""
	a.foldRun(tb, editor.FoldAll, "全部折叠")
	if a.toast == "" {
		t.Fatal("关闭折叠时执行应提示")
	}
}
