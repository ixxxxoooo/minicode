package app

import "testing"

func TestTbCodeEditable(t *testing.T) {
	a := New(Options{Mode: ModeIDE})
	a.ide.openFile("src/router.rs", true)
	tb := a.ide.cur()
	if ok, _ := a.tbCodeEditable(tb); !ok {
		t.Fatal("演示文件应可编辑")
	}
	tb.diff = true
	if ok, why := a.tbCodeEditable(tb); ok || why != "差异标签的此侧不可编辑" {
		t.Fatalf("diff=%v why=%q", ok, why)
	}
}
