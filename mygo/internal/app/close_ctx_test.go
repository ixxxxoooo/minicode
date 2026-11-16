package app

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func inputDown() ui.InputEvent { return ui.InputEvent{Kind: ui.InputPointerDown} }

// ⌘W 看你在哪:编辑器标签组里只关标签,标签关完 / 不在标签区域才关窗口。
func TestCloseHereByZone(t *testing.T) {
	a, tt := newTestApp(t)
	a.SetMode(ModeIDE)
	a.ide.tabs, a.ide.active = nil, ""
	a.ide.openFile("docs/guide.md", true)
	a.ide.openFile("docs/api.md", true)
	tt.Frame()

	a.ide.cz.zone = zoneOther // 人在对话栏 / 导航栏
	if k := a.closeTarget(); k != closeWindow {
		t.Fatalf("不在标签区域应关窗口,得到 %s", k)
	}
	if n := len(a.ide.tabs); n != 2 {
		t.Fatalf("判断不应改动标签,剩 %d", n)
	}

	a.ide.cz.zone = zoneEditor
	if k := a.closeHere(); k != closeTab || len(a.ide.tabs) != 1 {
		t.Fatalf("在编辑区应只关当前标签:%s,剩 %d", k, len(a.ide.tabs))
	}
	if k := a.closeHere(); k != closeTab || len(a.ide.tabs) != 0 {
		t.Fatalf("第二次仍关标签:%s,剩 %d", k, len(a.ide.tabs))
	}
	if k := a.closeHere(); k != closeWindow {
		t.Fatalf("标签关完后应关窗口,得到 %s", k)
	}
}

// 终端:两个及以上关当前终端标签;只剩一个时收起面板、不结束 shell。
func TestCloseHereTerminal(t *testing.T) {
	a, tt := newTestApp(t)
	a.SetMode(ModeIDE)
	a.ide.bottomOpen, a.ide.bottomTab = true, "term"
	a.ide.tps = []*termPanel{{}, {}}
	a.ide.tpSel = 1
	tt.Frame()
	a.ide.cz.zone = zonePanel
	if k := a.closeHere(); k != closeTermT || len(a.ide.tps) != 1 {
		t.Fatalf("应关当前终端标签:%s,剩 %d", k, len(a.ide.tps))
	}
	if k := a.closeHere(); k != closePanel || a.ide.bottomOpen || len(a.ide.tps) != 1 {
		t.Fatalf("只剩一个终端应收起面板:%s open=%v 剩 %d", k, a.ide.bottomOpen, len(a.ide.tps))
	}
	if a.ide.cz.zone != zoneEditor {
		t.Fatal("面板收起后位置应回到编辑器标签组")
	}
}

// 拆分:只关焦点组的标签;空着的组再按一次是关掉这一组,另一组的标签留下。
func TestCloseHereSplit(t *testing.T) {
	a, tt := newTestApp(t)
	a.SetMode(ModeIDE)
	a.ide.tabs, a.ide.active = nil, ""
	a.ide.openFile("docs/guide.md", true)
	a.splitEditor(1) // 次组打开同一文件,焦点在次组
	tt.Frame()
	a.ide.cz.zone = zoneEditor
	if k := a.closeHere(); k != closeTab || len(a.ide.tabs2) != 0 || len(a.ide.tabs) != 1 {
		t.Fatalf("应只关次组的标签:%s 主 %d 次 %d", k, len(a.ide.tabs), len(a.ide.tabs2))
	}
	if k := a.closeHere(); k != closeGroup || a.ide.splitDir != 0 || len(a.ide.tabs) != 1 {
		t.Fatalf("空组应被关掉:%s split=%d 主 %d", k, a.ide.splitDir, len(a.ide.tabs))
	}
}

// 未保存的标签:⌘W 先确认,不直接丢修改。
func TestCloseHereDirtyAsks(t *testing.T) {
	a, tt := newTestApp(t)
	a.SetMode(ModeIDE)
	a.ide.tabs, a.ide.active = nil, ""
	a.ide.openFile("docs/guide.md", true)
	tt.Frame()
	tb := a.ide.cur()
	if tb == nil || tb.buf == nil {
		t.Skip("演示沙盒里没有可编辑的 guide.md")
	}
	tb.saved = tb.buf.String() + "x" // 造出「与已保存内容不同」
	a.ide.cz.zone = zoneEditor
	if k := a.closeHere(); k != closeTab || len(a.ide.tabs) != 1 {
		t.Fatalf("未保存时不应直接关:%s,剩 %d", k, len(a.ide.tabs))
	}
}

// 「你在哪」:焦点换区域、刚点开标签会跟过去;Claude 自己开的标签(离上次输入很久)不改变位置。
func TestZoneSync(t *testing.T) {
	a, tt := newTestApp(t)
	a.SetMode(ModeIDE)
	a.ide.tabs, a.ide.active = nil, ""
	tt.Frame()
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	a.SetClock(func() time.Time { return now })

	a.zoneObserve(zoneOther)(inputDown()) // 点了文件树
	a.ide.openFile("docs/guide.md", true)
	tt.Frame()
	if a.ide.cz.zone != zoneEditor {
		t.Fatal("刚点开的标签应把位置带到编辑器标签组")
	}

	a.zoneObserve(zoneOther)(inputDown()) // 回到对话栏
	now = now.Add(5 * time.Second)
	a.ide.openFile("docs/api.md", true) // 过了很久才出现的标签:不是这次点击开的
	tt.Frame()
	if a.ide.cz.zone != zoneOther {
		t.Fatal("不是用户刚点开的标签不应改变位置")
	}
}
