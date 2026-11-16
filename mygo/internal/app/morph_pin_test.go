package app

import (
	"testing"
	"time"
)

// 形态切换期间会话流钉在终态宽度:有可信记录才钉,布局签名变了就不钉。
func TestMorphPinFlowWidth(t *testing.T) {
	a, tt := newTestApp(t)
	a.pinInTest = true
	tt.Frame()
	tt.Frame()
	agent := a.flowPinBy[ModeAgent]
	if agent.flowW <= 0 || agent.paneW <= 0 {
		t.Fatalf("Agent 形态稳定后应记下会话流宽度,得到 %+v", agent)
	}
	if _, ok := a.flowPinFor(); ok {
		t.Fatal("没有切换形态时不应钉宽")
	}

	a.SetMode(ModeIDE)
	if _, ok := a.flowPinFor(); ok {
		t.Fatal("IDE 形态还没量过宽度,不应钉宽")
	}
	tt.Frame()
	if got := a.flowPinBy[ModeIDE]; got.flowW != 0 {
		t.Fatalf("过渡期间量到的是中间宽度,不应记录,得到 %+v", got)
	}
	// 过渡结束(真实时钟管记录,测试器时钟管动画)
	time.Sleep(morphSettle + 50*time.Millisecond)
	tt.Advance(morphSettle)
	tt.Frame()
	tt.Frame()
	if _, ok := a.flowPinFor(); ok {
		t.Fatal("过渡结束后不应再钉宽")
	}
	if got := a.flowPinBy[ModeIDE]; got.flowW <= 0 {
		t.Fatalf("IDE 形态稳定后应记下宽度,得到 %+v", got)
	}

	a.SetMode(ModeAgent)
	pin, ok := a.flowPinFor()
	if !ok || pin.flowW != agent.flowW {
		t.Fatalf("切回 Agent 应钉在记下的宽度 %v,得到 %+v ok=%v", agent.flowW, pin, ok)
	}
	tt.Frame()
	if a.chatW != agent.paneW {
		t.Fatalf("钉宽期间三档应按终态对话栏宽 %v 判断,得到 %v", agent.paneW, a.chatW)
	}

	a.ly.side = a.sideW() + 40 // 布局变了:旧记录不可信
	if _, ok := a.flowPinFor(); ok {
		t.Fatal("栏宽变化后不应沿用旧宽度")
	}
}
