package app

import "testing"

func TestCompactFolderTail(t *testing.T) {
	a := New(Options{Mode: ModeIDE, Demo: true})
	a.SetMode(ModeIDE)
	tail, label, ok := a.ide.compactFolderTail("compact/only")
	if !ok || tail != "compact/only/a/b" || label != "only/a/b" {
		t.Fatalf("链尾=%q 标签=%q ok=%v", tail, label, ok)
	}
	tail, _, ok = a.ide.compactFolderTail("src")
	if ok {
		t.Fatalf("src 下有多个子项,不应合并,得 %q", tail)
	}
}