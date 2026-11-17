package app

import (
	"testing"

	"github.com/ixxxxoooo/minicode/internal/kit"
)

func TestMotionCatalogOpenClose(t *testing.T) {
	a, tt := newTestApp(t)
	a.openMotionCatalog()
	tt.Frame()
	if !a.motionCatOpen || !tt.HasText("动效总览") {
		t.Fatal("应打开动效总览")
	}
	if err := tt.Click("▶ 重放"); err == nil {
		tt.Frame()
		if a.motionReplayKey == "" {
			t.Fatal("点重放应选中一条动效")
		}
	}
	a.closeMotionCatalog()
	for i := 0; i < 12; i++ {
		tt.Advance(kit.DurFast / 4)
		tt.Frame()
	}
	if a.motionCatOpen {
		t.Fatal("关闭后 motionCatOpen 应为 false")
	}
}
