package app

import "testing"

func TestReorderPinnedTabs(t *testing.T) {
	tabs := []*ideTab{
		{id: "a", path: "a", pinned: false},
		{id: "b", path: "b", pinned: true},
		{id: "c", path: "c", pinned: false},
		{id: "d", path: "d", pinned: true},
	}
	got := reorderPinnedTabs(tabs)
	if len(got) != 4 || got[0].id != "b" || got[1].id != "d" || got[2].id != "a" {
		t.Fatalf("固定标签应排到最前:%v", []string{got[0].id, got[1].id, got[2].id, got[3].id})
	}
}

func TestCloseOtherKeepsPinned(t *testing.T) {
	a, _ := demoIDE(t)
	a.ide.openFile("src/router.rs", true)
	a.ide.openFile("src/main.rs", true)
	a.ide.openFile("Cargo.toml", true)
	pin := a.ide.tabByPath("Cargo.toml")
	if pin == nil {
		t.Fatal("缺少标签")
	}
	a.setTabPinned(pin, true)
	cur := a.ide.tabByPath("src/main.rs")
	n := a.closeTabsWhere(func(x *ideTab) bool { return x.id == cur.id || x.pinned })
	if n < 1 || len(a.ide.tabs) != 2 { // 演示种子里本来就开着几个标签,所以不写死关了几个
		t.Fatalf("关闭其他应留当前+固定,关 %d 剩 %d", n, len(a.ide.tabs))
	}
	if !a.ide.tabByPath("Cargo.toml").pinned {
		t.Fatal("固定标签应仍在")
	}
}
