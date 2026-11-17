package app

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/workspace"
)

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

func TestScrollRowIntoView(t *testing.T) {
	// 已可见:不动;在下方:滚到恰好露出并多留一行;在上方:回到上方并多留一行
	if y := scrollRowIntoView(100, 300, 150, 24); y != 100 {
		t.Fatalf("可见时不该滚动: %v", y)
	}
	if y := scrollRowIntoView(0, 300, 500, 24); y != 500+48-300 {
		t.Fatalf("下方行应滚入视口: %v", y)
	}
	if y := scrollRowIntoView(400, 300, 100, 24); y != 76 {
		t.Fatalf("上方行应滚回: %v", y)
	}
	if y := scrollRowIntoView(40, 0, 100, 24); y != 40 {
		t.Fatalf("视口未知时不动: %v", y)
	}
}

// 端到端:文件很多、视口很矮时,自动定位后目标行要滚进可见范围(滚动位置变大)。
func TestRevealScrollsTreeAfterOpen(t *testing.T) {
	files := map[string]string{}
	for i := 0; i < 40; i++ { // 文件夹排在文件前,所以用 40 个文件夹把目标挤到树的最下面
		files[fmt.Sprintf("d%02d/f.txt", i)] = "x"
	}
	a := New(Options{Mode: ModeIDE})
	a.ide.fs = workspace.NewMem("demo", files)
	a.ide.listCache, a.ide.filesCache = nil, nil
	tt := ui.NewTester(func(c *ui.Context) { a.filesView(c) }, 300, 260)
	for i := 0; i < 3; i++ {
		tt.Frame()
	}
	a.ide.openFile("d39/f.txt", true)
	a.nav, a.sideOpen = "files", true
	for i := 0; i < 6; i++ {
		a.autoReveal()
		tt.Frame()
	}
	if a.ide.revealTo != "" {
		t.Fatalf("待滚动的行应已消费: %q", a.ide.revealTo)
	}
	if a.ide.treeSt.Y <= 0 {
		t.Fatalf("目标行在最下面,树应已滚动,Y=%v", a.ide.treeSt.Y)
	}
}
