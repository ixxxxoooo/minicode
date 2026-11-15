package app

import "testing"

func TestTermRenameCommitAndCancel(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a"})
	a.ide.tps = []*termPanel{{}, {}}
	a.beginTermRename(1)
	a.termRen.buf = "deploy"
	a.commitTermRename()
	if a.ide.tps[1].label != "deploy" {
		t.Fatalf("自定义名: %q", a.ide.tps[1].label)
	}
	a.beginTermRename(1)
	a.termRen.buf = ""
	a.commitTermRename()
	if a.ide.tps[1].label != "" {
		t.Fatal("空名应恢复默认")
	}
	a.beginTermRename(0)
	a.termRen.buf = "tmp"
	a.termRen = termRen{}
	if a.ide.tps[0].label != "" {
		t.Fatal("Esc 取消不应改名")
	}
}
