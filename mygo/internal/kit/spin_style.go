package kit

import (
	"math"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo/ui"
)

// WaitSpinStyle:0 沙漏 / 1 圆环 / 2 点阵 / 3 脉冲。
var WaitSpinStyle atomic.Int32

// SpinStyled 按 appearance.waitAnim 偏好渲染等待动画。
func SpinStyled(c *ui.Context, col ui.Color) ui.Element {
	switch WaitSpinStyle.Load() {
	case 1:
		return SpinRing(c, col)
	case 2:
		return spinDots(c, col)
	case 3:
		return spinPulse(c, col)
	default:
		return Spin(c, col)
	}
}

func spinDots(c *ui.Context, col ui.Color) ui.Element {
	e := ui.Row(c).Gap(2).Size(14, 11).Shrink(0)
	phase := e.Loop("dots", 900*time.Millisecond, ui.Linear)
	for i := 0; i < 3; i++ {
		i := i
		e.Children(func() {
			b := ui.Box(c).Size(4, 4).Radius(2).Background(col)
			if phase*3-float32(i) < 0.35 {
				b.Opacity(1)
			} else {
				b.Opacity(0.25)
			}
		})
	}
	return e
}

func spinPulse(c *ui.Context, col ui.Color) ui.Element {
	wrap := ui.Box(c).Size(11, 11).Shrink(0)
	s := 0.7 + 0.3*float32(math.Sin(float64(wrap.Loop("pulse", 900*time.Millisecond, ui.Linear)*2*math.Pi)))
	wrap.Scale(s, s)
	wrap.Children(func() { SpinRing(c, col) })
	return wrap
}
