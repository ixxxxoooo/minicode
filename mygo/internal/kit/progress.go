package kit

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

// Ring 画 14px 进度环;pct 0..100,小于 0 为不确定(旋转弧)。
func Ring(c *ui.Context, pct float32, col ui.Color) ui.Element {
	e := ui.Box(c).Size(14, 14).Shrink(0)
	e.Draw(func(p *ui.Painter, r ui.Rect) {
		cx, cy := r.X+r.W/2, r.Y+r.H/2
		if pct < 0 {
			pts := make([][2]float32, 0, 24)
			for i := 0; i <= 22; i++ {
				a := float64(i) / 22 * 4.7
				pts = append(pts, [2]float32{cx + 5*float32(math.Cos(a)), cy + 5*float32(math.Sin(a))})
			}
			p.StrokePolyline(pts, 1.6, col)
			return
		}
		sweep := max(0, min(100, pct)) / 100 * 360
		pts := make([][2]float32, 0, 32)
		for i := 0; i <= 30; i++ {
			a := float64(-90+i) / 30 * float64(sweep) * math.Pi / 180
			pts = append(pts, [2]float32{cx + 5*float32(math.Cos(a)), cy + 5*float32(math.Sin(a))})
		}
		p.StrokePolyline(pts, 1.6, col)
	})
	if pct < 0 {
		e.Rotate(e.Loop("ring", DurLoop/2, ui.Linear) * 360)
	}
	return e
}

// IndetMeter 不确定进度条(1.2s 循环扫光)。
func IndetMeter(c *ui.Context) ui.Element {
	bar := ui.Box(c).Height(3).Radius(2).Background(T.Fill2).Clip().Children(func() {
		i := ui.Box(c).HeightPercent(100).WidthPercent(35).Radius(2).Background(T.Ac)
		if MotionOn() {
			x := i.Loop("indet", 1200*time.Millisecond, ui.Linear)
			i.Margin(0, 0, 0, x*200-50)
		}
	})
	return bar
}
