package app

import "testing"

// TestTermEntries 验证起始页终端组的数据来源:没有终端 / 两个终端 / 当前与后台的标注;演示模式不列。
func TestTermEntries(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	if got := a.termEntries(); len(got) != 0 {
		t.Fatalf("演示模式不该列终端,得到 %v", got)
	}
	a.proj = t.TempDir()
	if got := a.termEntries(); len(got) != 0 {
		t.Fatalf("还没开终端时不该列,得到 %v", got)
	}
	a.ide.tps = []*termPanel{{}, {}}
	a.ide.tpSel = 1
	got := a.termEntries()
	if len(got) != 2 {
		t.Fatalf("应有 2 个终端,得到 %d", len(got))
	}
	if got[0].Name != a.termDefaultName(0) || got[1].Name != a.termDefaultName(1) || got[0].Name == got[1].Name {
		t.Fatalf("默认名应按目录命名且互不相同,得到 %q %q", got[0].Name, got[1].Name)
	}
	if !got[1].Current || got[0].Current {
		t.Fatalf("当前应是第二个: %+v", got)
	}
	if got[0].Sub != "尚未启动" {
		t.Fatalf("没启动的终端应标「尚未启动」,得到 %q", got[0].Sub)
	}
	a.ide.termSel = "cmd-1" // 选中的是 Claude 的只读命令标签时,没有「当前」终端
	for _, e := range a.termEntries() {
		if e.Current {
			t.Fatalf("选中只读标签时不应有当前终端: %+v", e)
		}
	}
}

// TestRevealTermAndNew 验证起始页入口:点终端行会展开面板并切到该终端,新建会追加并选中。
func TestRevealTermAndNew(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	a.proj = t.TempDir()
	a.ide.tps = []*termPanel{{}, {}}
	a.ide.bottomOpen, a.ide.termSel = false, "x"
	a.revealTerm(0)
	if !a.ide.bottomOpen || a.ide.bottomTab != "term" || a.ide.termSel != "" || a.ide.tpSel != 0 {
		t.Fatalf("展开后状态不对: open=%v tab=%q sel=%q tp=%d", a.ide.bottomOpen, a.ide.bottomTab, a.ide.termSel, a.ide.tpSel)
	}
	a.revealTerm(99) // 越界按最后一个
	if a.ide.tpSel != 1 {
		t.Fatalf("越界应落到最后一个,得到 %d", a.ide.tpSel)
	}
	a.openNewTerm()
	if len(a.ide.tps) != 3 || a.ide.tpSel != 2 {
		t.Fatalf("新建后应有 3 个且选中第三个,得到 %d / %d", len(a.ide.tps), a.ide.tpSel)
	}
}
