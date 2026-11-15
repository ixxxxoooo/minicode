package app

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/demo"
)

// newTestApp 建一个带演示数据的应用与无窗口测试器。
func newTestApp(t *testing.T) (*App, *ui.Tester) {
	t.Helper()
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	a.SetProjects(demo.Projects(), "fixbug")
	a.SetClock(func() time.Time { return time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC) })
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
func TestPaletteWelcomeCommands(t *testing.T) {
	a, _ := newTestApp(t)
	cmds, _ := a.palCommands()
	titles := map[string]bool{}
	for _, c := range cmds {
		titles[c.title] = true
	}
	for _, w := range []string{"压缩上下文", "新会话", "切换权限模式"} {
		if !titles[w] {
			t.Fatalf("命令列表缺 %q", w)
		}
	}
	modeSw := false
	for title := range titles {
		if stringsContains(title, "切换到") && stringsContains(title, "模式(重启)") {
			modeSw = true
		}
	}
	if !modeSw {
		t.Fatal("应有切换真实/演示运行模式命令")
	}
	a.curSess = ""
	for _, it := range a.palItems() {
		if it.title == "压缩上下文" && !it.off {
			t.Fatal("无会话时压缩应置灰")
		}
	}
}

func stringsContains(s, sub string) bool {
	return len(sub) <= len(s) && (s == sub || len(s) > 0 && (containsSub(s, sub)))
}

func containsSub(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestPaletteGotoLineQuery(t *testing.T) {
	a, tt := newTestApp(t)
	a.SetMode(ModeIDE)
	a.ide.openFile("src/router.rs", true)
	a.OpenPalette(":9999")
	tt.Frame()
	found := false
	for _, it := range a.palItems() {
		if stringsContains(it.title, "转到第") {
			found = true
			it.run()
			break
		}
	}
	if !found {
		t.Fatal("查询 :9999 应出现转到行项")
	}
	tb := a.ide.cur()
	if tb.st.CursorPos().Line != tb.buf.LineCount()-1 {
		t.Fatalf("超界应夹到末行 %d", tb.st.CursorPos().Line)
	}
}

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
