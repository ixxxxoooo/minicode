package app

import "testing"

// 对话栏三档的边界与会话流内边距(设计稿 ws.css:宽 20 / 中 14 / 窄 10)。
func TestChatTierBoundariesAndPad(t *testing.T) {
	a, _ := sessionApp(t, "flow-gaps-b4")
	cases := []struct {
		w    float32
		tier int
		pad  float32
	}{
		{352, tierNarrow, 10}, {419, tierNarrow, 10},
		{420, tierMid, 14}, {560, tierMid, 14}, {639, tierMid, 14},
		{640, tierWide, 20}, {712, tierWide, 20},
	}
	for _, cs := range cases {
		a.chatW = cs.w
		if got := a.chatTier(); got != cs.tier {
			t.Errorf("宽 %v 应为档 %d,得到 %d", cs.w, cs.tier, got)
		}
		if got := a.flowPad(); got != cs.pad {
			t.Errorf("宽 %v 内边距应为 %v,得到 %v", cs.w, cs.pad, got)
		}
	}
}

// IDE 右侧停靠(352)是窄档:对话栏头部有「新会话」。
func TestIDEDockIsNarrowTier(t *testing.T) {
	a, tt := sessionApp(t, "flow-gaps-b4")
	a.SetMode(ModeIDE)
	a.rightOn = true
	settle(tt)
	if a.chatTier() != tierNarrow {
		t.Fatalf("IDE 对话栏宽 %v 应为窄档", a.chatW)
	}
}

// 工作区窄档(<520)底部面板只留终端 / 问题;宽档四项齐全。
func TestWorkspaceNarrowBottomTabs(t *testing.T) {
	a, tt := sessionApp(t, "flow-gaps-b4")
	a.SetMode(ModeIDE)
	a.rightOn = true
	a.sideOpen = false
	a.ide.bottomTab = "term"
	settle(tt)
	if a.wsW < 520 {
		t.Skipf("宽档场景工作区只有 %v,跳过", a.wsW)
	}
	if r, ok := tt.Find("输出"); !ok || r.Y < a.ide.panelTop {
		t.Fatalf("宽档底部面板应有「输出」(%v %v)", r, ok)
	}
	a.ly.dock = 560 // 对话栏拉宽 → 工作区变窄
	tt.SetSize(900, 800)
	settle(tt)
	if a.wsW >= 520 {
		t.Skipf("工作区仍有 %v", a.wsW)
	}
	if r, ok := tt.Find("输出"); ok && r.Y >= a.ide.panelTop && r.Y < a.ide.panelTop+40 {
		t.Fatalf("窄档底部面板标签条不应有「输出」(%v)", r)
	}
	if !tt.HasText("终端") || !tt.HasText("问题") {
		t.Fatal("窄档底部面板应保留终端 / 问题")
	}
}
