package app

import (
	"fmt"
	"testing"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// 长会话:只构建视口附近的行,而不是全部条目。
func TestFlowListIsVirtual(t *testing.T) {
	a, tt := sessionApp(t, "flow-gaps-b4")
	for i := 0; i < 400; i++ {
		a.st.Apply(core.EvAppend{Item: &core.User{ID: fmt.Sprintf("lu%d", i), Text: fmt.Sprintf("长会话第 %d 问", i)}})
		a.st.Apply(core.EvAppend{Item: &core.Assistant{ID: fmt.Sprintf("la%d", i), Text: fmt.Sprintf("长会话第 %d 答,带一些文字让它占几行高度。", i)}})
	}
	tt.Frame()
	if !a.flowList.AtEnd() {
		t.Fatal("新内容进来时停在底部的列表应继续跟随")
	}
	if tt.HasText("长会话第 3 问") {
		t.Fatal("远离视口的行不应被构建")
	}
	if !tt.HasText("长会话第 399 答") {
		t.Fatal("最新一条应可见")
	}
	first, last := a.flowList.Visible()
	if last-first > 60 {
		t.Fatalf("同屏构建的行过多: %d", last-first)
	}
}

// 用户滚离底部后,新内容进来不能把他拽回底部。
func TestFlowListKeepsPlaceWhenScrolledAway(t *testing.T) {
	a, tt := sessionApp(t, "flow-gaps-b4")
	for i := 0; i < 100; i++ {
		a.st.Apply(core.EvAppend{Item: &core.Assistant{ID: fmt.Sprintf("la%d", i), Text: fmt.Sprintf("第 %d 段输出", i)}})
	}
	tt.Frame()
	a.flowSt.Y = 0
	tt.Frame()
	tt.Frame()
	if a.flowList.AtEnd() {
		t.Fatal("滚到顶后不应还在底部")
	}
	a.st.Apply(core.EvAppend{Item: &core.Assistant{ID: "late", Text: "迟到的新输出"}})
	tt.Frame()
	tt.Frame()
	if a.flowList.AtEnd() {
		t.Fatal("滚离底部期间新内容不应把视图拽回底部")
	}
	a.flowList.ScrollToEnd()
	tt.Frame()
	tt.Frame()
	if !a.flowList.AtEnd() {
		t.Fatal("「回到最新」应回到底部")
	}
}

// 窗口宽度来回拖动后内容仍然完整、列表仍在底部。
func TestFlowListSurvivesResize(t *testing.T) {
	a, tt := sessionApp(t, "flow-gaps-b4")
	for _, w := range []int{1280, 1000, 900, 1100, 1280} {
		tt.SetSize(w, 800)
		tt.Frame()
	}
	if !a.flowList.AtEnd() {
		t.Fatal("缩放窗口后应仍停在底部")
	}
}
