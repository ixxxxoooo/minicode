package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/demo"
)

// newTestApp 建一个带演示数据的应用与无窗口测试器。
func newTestApp(t *testing.T) (*App, *ui.Tester) {
	t.Helper()
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	a.SetProjects(demo.Projects(), "fixbug")
	a.SeedDemoExtras()
	demo.Replay(a.State(), demo.FixBug(false))
	tt := ui.NewTester(a.View, 1280, 800)
	tt.Frame()
	return a, tt
}

func TestFuzzy(t *testing.T) {
	if fuzzy("zz", "main") != -1 {
		t.Fatal("不应匹配")
	}
	if fuzzy("rt", "router.rs") < 0 {
		t.Fatal("子序列应匹配")
	}
	if fuzzy("rou", "router.rs") <= fuzzy("rou", "xx_r_o_u") {
		t.Fatal("连续命中应得分更高")
	}
}

// 主题二级列表:回车进入,下移即时预览,Esc 还原;再进入后回车确认。
func TestPaletteThemeFlow(t *testing.T) {
	a, tt := newTestApp(t)
	a.OpenPalette("主题")
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if a.pq.sub != "theme" {
		t.Fatalf("应进入主题列表,sub=%q", a.pq.sub)
	}
	tt.Key(0, ui.KeyDown)
	tt.Frame()
	if a.pal.ID == "paper" {
		t.Fatal("下移应即时预览别的主题")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if a.paletteOn || a.pal.ID != "paper" {
		t.Fatalf("Esc 应关闭并还原:on=%v pal=%s", a.paletteOn, a.pal.ID)
	}
	a.OpenPalette("主题")
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	tt.Key(0, ui.KeyDown)
	tt.Frame()
	want := a.pal.ID
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if a.paletteOn || a.pal.ID != want || want == "paper" {
		t.Fatalf("回车应确认预览的主题:on=%v pal=%s want=%s", a.paletteOn, a.pal.ID, want)
	}
}

// 文件搜索命中后回车:切到 IDE 并打开该文件。
func TestPaletteOpenFile(t *testing.T) {
	a, tt := newTestApp(t)
	a.OpenPalette("main.rs")
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if a.mode != ModeIDE || a.ide.cur() == nil || a.ide.cur().path != "src/main.rs" {
		t.Fatalf("应打开 src/main.rs:mode=%v cur=%v", a.mode, a.ide.cur())
	}
}
