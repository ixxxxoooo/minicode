package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

// flowFadeMask 在贴底跟随时于输入框上方叠一层渐隐,避免流突然截断。
func (a *App) flowFadeMask(c *ui.Context) {
	if a.flowNoFollow || !a.flowList.AtEnd() {
		return
	}
	t := a.tk
	if t.Glass() {
		return // 流光:输入区是玻璃,内容在它背后被磨砂,不需要再叠一层渐隐
	}
	if !kit.MotionOn() {
		return
	}
	h := float32(56)
	ui.Box(c).Absolute().Left(0).Right(0).Bottom(96).Height(h).HitTest(func(_, _ float32, _ ui.Rect) bool { return false }).
		MaskLinearGradient(ui.LinearGradient{
			Angle: 0,
			Stops: []ui.GradientStop{
				{Color: ui.Transparent, At: 0},
				{Color: t.BG.Alpha(.85), At: .55},
				{Color: t.BG, At: 1},
			},
		})
}
