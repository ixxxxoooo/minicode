package kit

import (
	"github.com/egoist/mygo/ui"
)

// SkelLine 是一条骨架占位条;height 默认 12,可指定宽度比例 0..1(0 = 撑满)。
func SkelLine(c *ui.Context, wFrac float32, h float32) ui.Element {
	if h == 0 {
		h = 12
	}
	b := ui.Box(c).Height(h).Radius(4).Background(T.Fill2).Shrink(0)
	if wFrac <= 0 {
		b.Grow(1).MinWidth(0)
	} else {
		b.WidthPercent(wFrac * 100)
	}
	if MotionOn() {
		o := b.Loop("skel", DurLoop, ui.Linear)
		tri := 1 - float32(int(o*2)%2)
		b.Opacity(.55 + .25*tri)
	}
	return b
}

// SkelRows 画几行典型列表骨架(会话行 / 设置行)。
func SkelRows(c *ui.Context, n int) {
	ui.Column(c).Gap(8).Children(func() {
		for i := 0; i < n; i++ {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				SkelLine(c, 0, 10).Width(12)
				ui.Column(c).Grow(1).Gap(6).Children(func() {
					SkelLine(c, .7, 12)
					SkelLine(c, .45, 10)
				})
			})
		}
	})
}
