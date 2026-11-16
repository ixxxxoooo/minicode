package app

import (
	"math"

	"github.com/egoist/mygo/ui"
)

// zoomAt 以 (cx, cy)(相对画布框)为中心把缩放乘上 f,并同步调整平移,使光标下的那一点不动。
// 缩放结果夹在 [lo, hi];夹住后不再改平移,避免「已到上限还在漂」。
func zoomAt(scale, panX, panY, cx, cy, f, lo, hi float32) (float32, float32, float32) {
	if scale <= 0 || f <= 0 {
		return scale, panX, panY
	}
	next := clamp(scale*f, lo, hi)
	if next == scale {
		return scale, panX, panY
	}
	ratio := next / scale
	return next, cx - (cx-panX)*ratio, cy - (cy-panY)*ratio
}

// wheelFactor 把一次滚轮增量换成缩放倍数:向上滚(DY<0)放大、向下滚缩小,
// 一格(约 40 DIP)≈ 1.1 倍,单次夹在 [0.5, 2] 内,免得惯性滚动一下飞到尽头。
func wheelFactor(dy float32) float32 {
	f := math.Pow(1.1, float64(-dy)/40)
	return float32(math.Min(2, math.Max(0.5, f)))
}

// viewerInput 返回全屏查看器(Mermaid / 画布图表)画布的输入处理:
//   - 鼠标滚轮刻度:以光标为中心缩放;
//   - 触控板两指滑动:平移;按住 Ctrl / ⌘ 滚动或触控板捏合:缩放;
//
// 任何一种都会退出「适应」模式。没处理的事件返回 false,交还给拖拽平移与快捷键。
func viewerInput(fit *bool, scale, panX, panY *float32, lo, hi float32) func(ev ui.InputEvent) bool {
	return func(ev ui.InputEvent) bool {
		switch ev.Kind {
		case ui.InputZoom:
			if ev.Scale <= 0 {
				return false
			}
			*fit = false
			*scale, *panX, *panY = zoomAt(*scale, *panX, *panY, ev.X, ev.Y, ev.Scale, lo, hi)
			return true
		case ui.InputScroll:
			if ev.Precise && ev.Mods&(ui.Ctrl|ui.Cmd) == 0 {
				// 触控板两指滑动 = 平移(DY>0 内容上移,与滚动方向一致)
				*fit = false
				*panX -= ev.DX
				*panY -= ev.DY
				return true
			}
			if ev.DY == 0 {
				return false
			}
			*fit = false
			*scale, *panX, *panY = zoomAt(*scale, *panX, *panY, ev.X, ev.Y, wheelFactor(ev.DY), lo, hi)
			return true
		}
		return false
	}
}
