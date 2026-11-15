package app

import (
	"testing"
	"time"

	"github.com/ixxxxoooo/minicode/internal/automation"
)

// 居中浮层(新建自动化等)的入场计时只在打开那一帧开始,不能被外壳每帧清掉。
func TestDialogEntryClockNotReset(t *testing.T) {
	a, tt := newTestApp(t)
	a.SetAutomations(automation.NewStore(nil))
	a.SetMode(ModeIDE)
	a.screen = ScreenAutomation
	tt.Frame()
	a.openAutoNewDlg()
	tt.Frame()
	first := a.dlgAt
	if first.IsZero() {
		t.Fatal("打开后应开始入场计时")
	}
	for i := 0; i < 5; i++ {
		time.Sleep(40 * time.Millisecond)
		tt.Frame()
	}
	if !a.dlgAt.Equal(first) {
		t.Fatal("入场计时被每帧重置,浮层会永远停在透明")
	}
}
