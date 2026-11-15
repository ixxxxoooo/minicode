package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

// flowFadeMask 在贴底跟随时于输入框上方叠一层渐隐,避免流突然截断。
func (a *App) flowFadeMask(c *ui.Context) {
	if a.flowNoFollow || a.flowSt.MaxY-a.flowSt.Y > 8 {
		return
	}
	t := a.tk
	if !kit.MotionOn() {
		return
	}
	h := float32(56)
	ui.Box(c).Absolute().Left(0).Right(0).Bottom(96).Height(h).HitTest(func(_, _ float32, _ ui.Rect) bool { return false }).
		MaskLinearGradient(ui.LinearGradient{
			Angle: 0,
			Stops: []ui.GradientStop{
				{ui.Transparent, 0},
				{t.BG.Alpha(.85), .55},
				{t.BG, 1},
			},
		})
}
