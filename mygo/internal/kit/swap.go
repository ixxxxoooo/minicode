package kit

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// EaseCSS 是 CSS 默认的 `ease` 曲线(cubic-bezier(.25,.1,.25,1)):设计稿里只写了时长、没写曲线的
// transition(如 `opacity var(--dur-fast)`)用它。
var EaseCSS = ui.CubicBezier(.25, .1, .25, 1)

// SwapProgress 返回一次「就地换内容」(设计稿 .a-swap)的进度 0..1:DurBase 时长、EaseOut 曲线。
// at 为零值(从未触发)或动效关闭时恒为 1。进行中时调用 frame 请求下一帧。
func SwapProgress(at time.Time, frame func()) float32 {
	if !MotionOn() || at.IsZero() || DurBase <= 0 {
		return 1
	}
	p := float32(time.Since(at)) / float32(DurBase)
	if p >= 1 {
		return 1
	}
	if p < 0 {
		p = 0
	}
	if frame != nil {
		frame()
	}
	return EaseOut(p)
}

// SwapIn 给内容区套上换内容入场(设计稿 `@keyframes swap`:自左 8px 滑入 + 淡入,180ms,ease-out)。
// 位移加在内容元素上、裁剪留给外层(设计稿里 .side-body 是 overflow:hidden),所以左缘会被裁掉一小截。
func SwapIn(el ui.Element, at time.Time, frame func()) ui.Element {
	p := SwapProgress(at, frame)
	if p >= 1 {
		return el
	}
	return el.Opacity(p).Translate(-8*(1-p), 0)
}
