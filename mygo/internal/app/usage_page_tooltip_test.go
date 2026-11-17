package app

import (
	"testing"
	"time"
)

func TestUsageChartBarTooltip(t *testing.T) {
	a, tt := newTestApp(t)
	a.screen = ScreenUsage
	tt.SetSize(1280, 800)
	tt.Frame()
	tt.Frame()
	bar, ok := tt.Find("bar-0")
	if !ok {
		t.Fatal("找不到柱状图第一根")
	}
	tt.Move(bar.X+bar.W/2, bar.Y+2)
	for i := 0; i < 40; i++ {
		tt.Advance(20 * time.Millisecond)
	}
	want := usageDayBarTip(a.usage().Days[0].Label, a.usage().Days[0].Cost, usageDayMean(a.usage()))
	if !tt.HasText(want) {
		t.Fatalf("悬停柱应显示 %q,当前文案 %v", want, tt.Texts())
	}
}

func TestUsageModelRowTooltip(t *testing.T) {
	a, tt := newTestApp(t)
	a.screen = ScreenUsage
	tt.SetSize(1280, 800)
	tt.Frame()
	u := a.usage()
	if len(u.Models) == 0 {
		t.Fatal("演示数据应有模型行")
	}
	tip := usageModelTip(u.Models[0].Name, u.Models[0].Fraction, u.Models[0].Cost, u.Total)
	// 点按模型标题附近
	if txt, ok := tt.Find(u.Models[0].Name); ok {
		tt.Move(txt.X+txt.W/2, txt.Y+txt.H/2)
		for i := 0; i < 40; i++ {
			tt.Advance(20 * time.Millisecond)
		}
		if !tt.HasText(tip) {
			t.Fatalf("模型行悬停应显示 %q,当前 %v", tip, tt.Texts())
		}
		return
	}
	t.Fatal("找不到模型名")
}
