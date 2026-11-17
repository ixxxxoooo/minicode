package app

import "testing"

func TestAutoRevealWhenVisibleDefers(t *testing.T) {
	a := New(Options{Mode: ModeIDE})
	a.prefs.Set("explorer.autoRevealWhenVisible", true)
	a.nav, a.sideOpen = "scm", true
	a.ide.openFile("src/handlers/users.rs", true)
	a.autoReveal()
	if a.ide.revealed != "" {
		t.Fatal("文件树不可见时不应记入 revealed")
	}
	if a.ide.open["src/handlers"] {
		t.Fatal("不应展开")
	}
	a.nav = "files"
	a.autoReveal()
	if a.ide.sel != "src/handlers/users.rs" || !a.ide.open["src/handlers"] {
		t.Fatalf("回到文件树后应定位,sel=%q open=%v", a.ide.sel, a.ide.open)
	}
}

func TestAutoRevealNoScroll(t *testing.T) {
	a := New(Options{Mode: ModeIDE})
	a.prefs.Set("explorer.autoRevealScroll", false)
	a.nav, a.sideOpen = "files", true
	a.ide.openFile("src/handlers/users.rs", true)
	a.autoReveal()
	if a.ide.revealTo != "" {
		t.Fatalf("关闭滚动时不应设置 revealTo:%q", a.ide.revealTo)
	}
	if a.ide.sel != "src/handlers/users.rs" {
		t.Fatalf("仍应选中文件,%q", a.ide.sel)
	}
}
