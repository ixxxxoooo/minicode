package app

import "testing"

func TestWinTier(t *testing.T) {
	a, _ := demoIDE(t)
	for _, tc := range []struct {
		w    float32
		want int
	}{
		{1280, 0},
		{1000, 1},
		{800, 2},
		{640, 3},
	} {
		a.winW = tc.w
		if got := a.winTier(); got != tc.want {
			t.Fatalf("w=%v tier=%d want %d", tc.w, got, tc.want)
		}
	}
}

func TestUseWsOverlay(t *testing.T) {
	a, _ := demoIDE(t)
	a.SetMode(ModeAgent)
	a.winW = 1000
	if !a.useWsOverlay() {
		t.Fatal("1000 Agent 应用工作区浮层")
	}
	a.winW = 1200
	if a.useWsOverlay() {
		t.Fatal("1200 不应浮层")
	}
}

func TestFitPanelsDrawerClosed(t *testing.T) {
	a, _ := demoIDE(t)
	a.SetMode(ModeAgent)
	a.sideOpen = true
	a.winW = 700
	a.fitPanels()
	if a.sideOpen || !a.fit.side {
		t.Fatal("<720 导航抽屉默认应收起并记自动标记")
	}
}

func TestFitPanelsSideRestore(t *testing.T) {
	a, _ := demoIDE(t)
	a.SetMode(ModeIDE)
	a.sideOpen, a.rightOn = true, true
	a.fit = autoFit{w: 1280, mode: ModeIDE}
	a.winW = 800
	a.fitPanels()
	if a.sideOpen || !a.fit.side {
		t.Fatal("900 以下应自动收起导航栏")
	}
	a.winW = 1150
	a.fitPanels()
	if !a.sideOpen || a.fit.side {
		t.Fatal("变宽应恢复自动收起的导航栏")
	}
}
