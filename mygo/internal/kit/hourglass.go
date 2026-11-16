package kit

import (
	"time"

	"github.com/egoist/mygo/ui"
)

const (
	hourglassPeriod = 2400 * time.Millisecond
	hourglassFlipAt = 1900 * time.Millisecond
)

// globalHourglassPhase 0..1,全界面沙漏同步。
func globalHourglassPhase() float32 {
	if !MotionOn() {
		return 0
	}
	ns := time.Now().UnixNano() % int64(hourglassPeriod)
	return float32(ns) / float32(hourglassPeriod)
}

// Spin 画 11px 沙漏(默认等待指示);动效关闭时静态中位沙量。
func Spin(c *ui.Context, col ui.Color) ui.Element {
	e := ui.Box(c).Size(11, 11).Shrink(0)
	phase := globalHourglassPhase()
	if MotionOn() {
		_ = e.Loop("hg", 33*time.Millisecond, ui.Linear)
	}
	e.Draw(func(p *ui.Painter, r ui.Rect) {
		drawHourglass(p, r, col, phase)
	})
	flipStart := float32(hourglassFlipAt) / float32(hourglassPeriod)
	if MotionOn() && phase >= flipStart {
		t := (phase - flipStart) / (1 - flipStart)
		ease := t * t * (3 - 2*t)
		e.Rotate(ease * 180)
	}
	return e
}

func drawHourglass(p *ui.Painter, r ui.Rect, col ui.Color, phase float32) {
	flipStart := float32(hourglassFlipAt) / float32(hourglassPeriod)
	var flow float32
	switch {
	case !MotionOn():
		flow = .45
	case phase < flipStart:
		flow = phase / flipStart
	default:
		fp := (phase - flipStart) / (1 - flipStart)
		if fp >= 1 {
			flow = 0
		} else {
			flow = 1
		}
	}
	x, y, w, h := r.X, r.Y, r.W, r.H
	outline := [][2]float32{
		{x + w*.25, y + h*.08},
		{x + w*.75, y + h*.08},
		{x + w*.62, y + h*.38},
		{x + w*.52, y + h*.46},
		{x + w*.62, y + h*.62},
		{x + w*.75, y + h*.92},
		{x + w*.25, y + h*.92},
		{x + w*.38, y + h*.62},
		{x + w*.48, y + h*.46},
		{x + w*.38, y + h*.38},
	}
	p.StrokePolyline(outline, 1.2, col)
	topH := (1 - flow) * h * .28
	if topH > .5 {
		p.Fill(ui.Rect{X: x + w*.32, Y: y + h*.12, W: w * .36, H: topH}, col.Mix(ui.Transparent, .12), 1)
	}
	botH := flow * h * .28
	if botH > .5 {
		p.Fill(ui.Rect{X: x + w*.32, Y: y + h*.92 - botH, W: w * .36, H: botH}, col.Mix(ui.Transparent, .12), 1)
	}
	if flow > .05 && flow < .98 {
		p.Fill(ui.Rect{X: x + w*.46, Y: y + h*.45, W: w * .08, H: h * .08}, col, 1)
	}
}
