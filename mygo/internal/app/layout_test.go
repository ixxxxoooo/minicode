package app

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

func dragTo(tt *ui.Tester, x0, y0, x1, y1 float32) {
	tt.Move(x0, y0)
	tt.Press(x0, y0)
	tt.Frame()
	const n = 6
	for i := 1; i <= n; i++ {
		f := float32(i) / n
		tt.Move(x0+(x1-x0)*f, y0+(y1-y0)*f)
		tt.Frame()
	}
	tt.Release(x1, y1)
	tt.Frame()
	tt.Frame()
}

func TestResizeSideAndPersist(t *testing.T) {
	a, tt := quickApp(t)
	tt.Advance(time.Second)
	tt.Frame()
	w0 := a.sideW()
	if w0 != theme.WSide {
		t.Fatalf("默认导航栏宽 %v", w0)
	}
	dragTo(tt, w0-2+a.tk.Scheme.GapPane, 300, w0+58+a.tk.Scheme.GapPane, 300)
	if got := a.sideW(); got < w0+40 {
		t.Fatalf("拖动后导航栏应变宽:%v → %v", w0, got)
	}
	if !a.prefs.Has("layout.side") {
		t.Fatal("松手后应写偏好")
	}
	// 拖得太窄:夹在 180
	tt.Advance(time.Second)
	tt.Frame()
	dragTo(tt, a.sideW()-2+a.tk.Scheme.GapPane, 300, 10, 300)
	if got := a.sideW(); got != 180 {
		t.Fatalf("最小宽 180,得到 %v", got)
	}
}

func TestResizeWorkspaceAndPanel(t *testing.T) {
	a, tt := demoIDE(t)
	tt.Advance(time.Second) // 等形态变形过渡走完
	tt.Frame()
	// IDE 形态:右侧对话栏拖宽
	d0 := a.dockW()
	x := 1280 - d0 + 2
	dragTo(tt, x, 300, x-80, 300)
	if got := a.dockW(); got < d0+60 {
		t.Fatalf("对话栏应变宽:%v → %v", d0, got)
	}
	// 底部面板拖高:把手在面板上缘
	p0 := a.panelHeight()
	r := a.ide.panelTop
	if r <= 0 {
		t.Skip("面板上缘位置未量到")
	}
	dragTo(tt, 400, r+1, 400, r-60)
	if got := a.panelHeight(); got < p0+40 {
		t.Fatalf("面板应变高:%v → %v", p0, got)
	}
}

func TestChatMaxExitAndPanelMax(t *testing.T) {
	a, tt := demoIDE(t)
	a.toggleChatMax()
	tt.Frame()
	if !a.chatMax {
		t.Fatal("应进入对话占满")
	}
	if !tt.HasText("还原") && a.wsActive(true) {
		t.Fatal("占满时工作区应让位")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if a.chatMax {
		t.Fatal("Esc 应退出对话占满")
	}
	a.togglePanelMax()
	tt.Frame()
	if !a.ly.panelMax || !a.ide.bottomOpen {
		t.Fatal("面板应最大化")
	}
	a.togglePanelMax()
	if a.ly.panelMax {
		t.Fatal("再切一次应还原")
	}
}

func TestAgentBackForward(t *testing.T) {
	a, tt := quickApp(t)
	tt.Frame()
	if a.canBack() || a.canForward() {
		t.Fatal("初始没有历史")
	}
	a.OpenSession("s-a")
	a.OpenSession("s-b")
	if !a.canBack() || a.canForward() {
		t.Fatalf("两次打开后应能后退:i=%d n=%d", a.sessI, len(a.sessNav))
	}
	a.navGo(-1)
	if a.curSess != "s-a" || !a.canForward() {
		t.Fatalf("后退应回到 s-a:%q", a.curSess)
	}
	a.navGo(1)
	if a.curSess != "s-b" {
		t.Fatalf("前进应回到 s-b:%q", a.curSess)
	}
}

func TestWorkspaceStartPage(t *testing.T) {
	a, tt := demoIDE(t)
	for len(a.ide.tabs) > 0 {
		a.ide.closeTab(a.ide.tabs[0].id)
	}
	tt.Frame()
	if !tt.HasText("审查本轮改动") || !tt.HasText("终端") {
		t.Fatal("空工作区应显示起始页")
	}
}

// 面板上缘向上拖到头会吸附成最大化;最大化时从上缘向下拖一段还原。
func TestPanelDragSnapMax(t *testing.T) {
	a, tt := demoIDE(t)
	tt.Advance(time.Second)
	tt.Frame()
	r := a.ide.panelTop
	if r <= 0 {
		t.Skip("面板上缘位置未量到")
	}
	dragTo(tt, 400, r+1, 400, 20)
	tt.Frame()
	if !a.ly.panelMax {
		t.Fatalf("拖到顶应吸附成最大化(高度 %v)", a.panelHeight())
	}
	tt.Advance(time.Second)
	tt.Frame()
	r2 := a.ide.panelTop
	dragTo(tt, 400, r2+1, 400, r2+200)
	tt.Frame()
	if a.ly.panelMax {
		t.Fatal("在最大化的上缘向下拖应还原")
	}
}
