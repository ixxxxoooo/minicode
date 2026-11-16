package kit

import (
	"math"

	"github.com/egoist/mygo/ui"
)

// SpinRing 是 11px 旋转圆环(仅按钮 loading 等紧凑场景)。
func SpinRing(c *ui.Context, col ui.Color) ui.Element {
	e := ui.Box(c).Size(11, 11).Shrink(0)
	e.Draw(func(p *ui.Painter, r ui.Rect) {
		cx, cy := r.X+r.W/2, r.Y+r.H/2
		pts := make([][2]float32, 0, 24)
		for i := 0; i <= 22; i++ {
			a := float64(i) / 22 * 4.7
			pts = append(pts, [2]float32{cx + 4.75*float32(math.Cos(a)), cy + 4.75*float32(math.Sin(a))})
		}
		p.StrokePolyline(pts, 1.5, col)
	})
	if MotionOn() {
		e.Rotate(e.Loop("spin", DurLoop/2, ui.Linear) * 360)
	}
	return e
}
