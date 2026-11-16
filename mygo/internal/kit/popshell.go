package kit

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

// PopUpIn 自下而上的卡片入场(设计稿 a-pop-up:scale .97 + translateY 8px,120ms,DurBase)。
func PopUpIn(panel ui.Element, openedAt time.Time, frame func()) ui.Element {
	if !MotionOn() || openedAt.IsZero() {
		return panel
	}
	p := float32(time.Since(openedAt)) / float32(DurBase)
	if p >= 1 {
		return panel
	}
	if frame != nil {
		frame()
	}
	ease := 1 - (1-p)*(1-p)
	sc := .97 + .03*ease
	return panel.Opacity(ease).Scale(sc, sc).Translate(0, (1-ease)*8)
}

// PopIn 弹层入场(透明度 + 自下方 4px 升起,120ms)。
func PopIn(panel ui.Element, openedAt time.Time) ui.Element {
	if !MotionOn() || openedAt.IsZero() {
		return panel
	}
	p := float32(time.Since(openedAt)) / float32(DurFast)
	if p >= 1 {
		return panel
	}
	p = 1 - (1-p)*(1-p)
	return panel.Opacity(p).Translate(0, (1-p)*4)
}

// PopOut 弹层退场(比入场略快,向上 3px 淡出)。
func PopOut(panel ui.Element, closedAt time.Time) ui.Element {
	if !MotionOn() || closedAt.IsZero() {
		return panel
	}
	p := float32(time.Since(closedAt)) / float32(DurFast)
	if p >= 1 {
		return panel.Opacity(0)
	}
	p = p * p // ease-in
	return panel.Opacity(1-p).Translate(0, -3*p)
}

// DialogIn 居中浮层入场:遮罩 + 卡片下落并轻放大(180ms)。
func DialogIn(backdrop, panel ui.Element, openedAt time.Time, frame func()) {
	if !MotionOn() || openedAt.IsZero() {
		return
	}
	p := float32(time.Since(openedAt)) / float32(DurBase)
	if p >= 1 {
		return
	}
	if frame != nil {
		frame()
	}
	p = 1 - (1-p)*(1-p)*(1-p)
	backdrop.Opacity(p)
	panel.Translate(0, (p-1)*8).Scale(.97+.03*p, .97+.03*p)
}

// DialogOut 居中浮层退场(遮罩淡出 + 卡片略缩小)。
func DialogOut(backdrop, panel ui.Element, closedAt time.Time, frame func()) {
	if !MotionOn() || closedAt.IsZero() {
		return
	}
	p := float32(time.Since(closedAt)) / float32(DurFast)
	if p >= 1 {
		return
	}
	if frame != nil {
		frame()
	}
	p = p * p
	backdrop.Opacity(1 - p)
	panel.Translate(0, -4*p).Scale(1-.02*p, 1-.02*p).Opacity(1 - p*.6)
}

// StaggerRise 列表项依次淡入下落(间隔 30ms)。
func StaggerRise(el ui.Element, index int, baseAt time.Time, frame func()) {
	if !MotionOn() || baseAt.IsZero() {
		return
	}
	delay := time.Duration(index) * 30 * time.Millisecond
	p := float32(time.Since(baseAt)-delay) / float32(DurBase)
	if p <= 0 {
		el.Opacity(0).Translate(0, 6)
		if frame != nil {
			frame()
		}
		return
	}
	if p >= 1 {
		return
	}
	if p < 1 && frame != nil {
		frame()
	}
	ease := 1 - (1-p)*(1-p)
	el.Opacity(ease).Translate(0, (1-ease)*6)
}

// CollapseHide 删除折叠:高度收到 0 并淡出,时长 DurSlow。
func CollapseHide(el ui.Element, started time.Time, fullH float32, frame func()) {
	CollapseHideFor(el, started, fullH, DurSlow, frame)
}

// CollapseHideFor 与 CollapseHide 相同,可指定时长(状态行结束用 DurStatus)。
func CollapseHideFor(el ui.Element, started time.Time, fullH float32, dur time.Duration, frame func()) {
	if !MotionOn() || started.IsZero() {
		return
	}
	p := float32(time.Since(started)) / float32(dur)
	if p >= 1 {
		el.Opacity(0).Height(0).Clip()
		return
	}
	if frame != nil {
		frame()
	}
	io := .5 - .5*float32(math.Cos(float64(p*math.Pi))) // ease-io
	el.Height(fullH * (1 - io)).Opacity(1 - io).Clip()
}
