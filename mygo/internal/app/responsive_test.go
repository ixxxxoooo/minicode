package app

import "testing"

func TestFitNeeds(t *testing.T) {
	s, r := fitNeeds(ModeAgent, 228, 352)
	if s != 228+400+360 || r != 400+360 {
		t.Fatalf("Agent 阈值 %v %v", s, r)
	}
	s, r = fitNeeds(ModeIDE, 228, 352)
	if s != 228+480+352 || r != 480+352 {
		t.Fatalf("IDE 阈值 %v %v", s, r)
	}
}

// 变窄先收导航栏、再收右侧栏;变宽(过了回差)只恢复自动收起的;用户手动打开的不被再收回。
func TestFitPanelsShrinkAndRestore(t *testing.T) {
	a, tt := demoIDE(t)
	tt.Frame()
	a.SetMode(ModeAgent)
	a.rightOn, a.sideOpen = true, true
	a.fit = autoFit{mode: ModeAgent}
	a.winW = 1280
	a.fitPanels()
	if !a.sideOpen || !a.rightOn {
		t.Fatal("1280 两侧都放得下")
	}
	a.winW = 1000 // < 228+400+360 = 988? 不,1000 ≥ 988,仍放得下
	a.fitPanels()
	if !a.sideOpen {
		t.Fatal("1000 导航栏还放得下")
	}
	a.winW = 960
	a.fitPanels()
	if a.sideOpen || !a.rightOn || !a.fit.side {
		t.Fatalf("960 应只收导航栏:side=%v right=%v", a.sideOpen, a.rightOn)
	}
	a.winW = 700
	a.fitPanels()
	if a.rightOn || !a.fit.right {
		t.Fatal("700 应再收右侧栏")
	}
	a.winW = 1300
	a.fitPanels()
	if !a.sideOpen || !a.rightOn || a.fit.side || a.fit.right {
		t.Fatal("变宽后两侧都应恢复")
	}
	// 手动收起的不被自动恢复
	a.sideOpen = false
	a.winW = 1400
	a.fitPanels()
	if a.sideOpen {
		t.Fatal("手动收起的导航栏不该被恢复")
	}
}
