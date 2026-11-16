package app

import "testing"

func TestAutoRevealExpandsAndSelectsOnce(t *testing.T) {
	a := New(Options{Mode: ModeIDE})
	a.ide.openFile("src/handlers/users.rs", true)
	a.autoReveal()
	if a.ide.sel != "src/handlers/users.rs" || !a.ide.open["src"] || !a.ide.open["src/handlers"] {
		t.Fatalf("应展开并选中,sel=%q open=%v", a.ide.sel, a.ide.open)
	}
	// 用户自己折叠后,同一标签下不再被覆盖
	a.ide.open["src"] = false
	a.autoReveal()
	if a.ide.open["src"] {
		t.Fatal("同一标签只定位一次,不该再次展开")
	}
}

func TestAutoRevealOff(t *testing.T) {
	a := New(Options{Mode: ModeIDE})
	a.prefs.Set("explorer.autoReveal", false)
	a.ide.openFile("src/handlers/users.rs", true)
	delete(a.ide.open, "src/handlers")
	a.autoReveal()
	if a.ide.open["src/handlers"] {
		t.Fatal("偏好关闭时不应自动展开")
	}
}
