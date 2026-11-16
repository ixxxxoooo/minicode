package app

import "testing"

// 标题栏 panelL/B/R 在展开时应走 Ib Toggle 选中态(不 panic 即可,视觉对照靠截图)。
func TestTitlebarPanelTogglesBuild(t *testing.T) {
	a, tt := newTestApp(t)
	for _, on := range []bool{true, false} {
		a.sideOpen = on
		a.ide.bottomOpen = on
		a.rightOn = on
		tt.Frame()
	}
}
