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

// 回归:没点重命名时,任何终端标签(尤其是第 0 个)都不该处于重命名状态。
func TestTermRenameIdleByDefault(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a"})
	a.ide.tps = []*termPanel{{}, {}}
	for i := range a.ide.tps {
		if a.termRenaming(i) {
			t.Fatalf("默认状态下终端 %d 不该在重命名", i)
		}
	}
	a.beginTermRename(0)
	if !a.termRenaming(0) || a.termRenaming(1) {
		t.Fatal("只有被点的那个在重命名")
	}
	a.commitTermRename()
	if a.termRenaming(0) {
		t.Fatal("提交后应退出重命名")
	}
}

// 起始页的终端列表与标签条同名(含用户重命名)。
func TestTermEntriesFollowRename(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a"})
	a.proj = "/tmp/x"
	a.ide.tps = []*termPanel{{label: "build"}, {}}
	es := a.termEntries()
	if len(es) != 2 || es[0].Name != "build" || es[1].Name != a.termDefaultName(1) {
		t.Fatalf("名称不一致:%+v", es)
	}
}
