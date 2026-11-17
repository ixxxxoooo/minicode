package app

import (
	"testing"
	"time"

	"github.com/ixxxxoooo/minicode/internal/kit"
)

func TestUsageChartBarTooltip(t *testing.T) {
	was := kit.MotionOn()
	kit.SetMotion(false)
	t.Cleanup(func() { kit.SetMotion(was) })
	a, tt := newTestApp(t)
	a.screen = ScreenUsage
	a.use.revealAt = time.Time{}
	tt.SetSize(1280, 800)
	for range 8 {
		tt.Frame()
	}
	head, ok := tt.Find("每日费用")
	if !ok {
		t.Fatal("找不到每日费用图")
	}
	// 柱区域在标题下方一行内,取中部偏左第一根柱
	tt.Move(head.X+60, head.Y+head.H+48)
	for range 40 {
		tt.Advance(20 * time.Millisecond)
	}
	want := usageDayBarTip(a.usage().Days[0].Label, a.usage().Days[0].Cost, usageDayMean(a.usage()))
	okTip := false
	for _, ac := range tt.AccessAll() {
		if ac.Description == want {
			okTip = true
			break
		}
	}
	if !okTip && !tt.HasText(want) {
		t.Fatalf("悬停柱应显示 %q", want)
	}
}

func TestUsageModelRowTooltip(t *testing.T) {
	was := kit.MotionOn()
	kit.SetMotion(false)
	t.Cleanup(func() { kit.SetMotion(was) })
	a, tt := newTestApp(t)
	a.screen = ScreenUsage
	a.use.revealAt = time.Time{}
	tt.SetSize(1280, 800)
	for range 6 {
		tt.Frame()
	}
	u := a.usage()
	if len(u.Models) == 0 {
		t.Fatal("演示数据应有模型行")
	}
	tip := usageModelTip(u.Models[0].Name, u.Models[0].Fraction, u.Models[0].Cost, u.Total)
	if txt, ok := tt.Find(u.Models[0].Name); ok {
		tt.Move(txt.X+txt.W/2, txt.Y+txt.H/2)
		for range 40 {
			tt.Advance(20 * time.Millisecond)
		}
		if !tt.HasText(tip) {
			t.Fatalf("模型行悬停应显示 %q,当前 %v", tip, tt.Texts())
		}
		return
	}
	t.Fatal("找不到模型名")
}
