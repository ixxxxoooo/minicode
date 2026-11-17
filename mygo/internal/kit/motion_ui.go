package kit

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

// shimmerMode: 0=流光 1=呼吸 2=关闭(见 appearance.shimmer)。
var shimmerMode int

// SetShimmer 设置运行中文字的样式(与 SetPulse 配合)。
func SetShimmer(mode int) {
	shimmerMode = mode
	if mode == 2 {
		pulseOn = false
	} else {
		pulseOn = true
	}
}

// RunText 对「进行中」文字应用流光或呼吸(依偏好);无附文案时仍走旧透明度路径。
func RunText(e ui.Element) ui.Element {
	switch shimmerMode {
	case 0:
		return TextShimmer(e)
	default:
		return Pulse(e)
	}
}

// TextShimmer 降级:长文案或未走 RunTextS 时整段呼吸。
func TextShimmer(e ui.Element) ui.Element {
	if !MotionOn() || shimmerMode != 0 || !viewOK() {
		return e
	}
	x := e.Loop("shimmer", DurLoop, ui.Linear)
	bright := .35 + .65*(.5-.5*float32(math.Cos(float64(x*2*math.Pi))))
	return e.Opacity(bright)
}

const dockGlowPeriod = 2200 * time.Millisecond

// DockGlow 权限 / 提问交互卡的描边呼吸(设计稿 a-glow,2.2s)。
func DockGlow(e ui.Element, frame func()) ui.Element {
	if !MotionOn() {
		return e
	}
	x := e.Loop("dock-glow", dockGlowPeriod, ui.Linear)
	tri := 1 - float32(math.Abs(float64(2*x-1)))
	e.Border(1, T.AcLine.Mix(T.Ac, tri*.55))
	e.Shadow(0, 0, 0, 3+tri*2, T.Ac.Mix(ui.Transparent, .88-tri*.08))
	for _, s := range T.Shadow2() {
		e.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
	}
	if frame != nil {
		frame()
	}
	return e
}

// DrawerSlide 抽屉自侧边滑入(22px)+淡入。
func DrawerSlide(panel ui.Element, openedAt time.Time, fromRight bool, frame func()) {
	if !MotionOn() || openedAt.IsZero() {
		return
	}
	p := float32(Since(openedAt)) / float32(DurSlow)
	if p >= 1 {
		return
	}
	if frame != nil {
		frame()
	}
	ease := 1 - (1-p)*(1-p)
	dx := 22 * (1 - ease)
	if fromRight {
		panel.Translate(dx, 0).Opacity(ease)
	} else {
		panel.Translate(-dx, 0).Opacity(ease)
	}
}

// ToastSlide 右上角提示条自右侧滑入。
func ToastSlide(card ui.Element, at time.Time, frame func()) ui.Element {
	return ToastSlideKeep(card, at, frame, false)
}

// ToastSlideKeep 同 ToastSlide;keep 为真时只做入场滑入,之后一直显示(带操作按钮的通知,等用户自己关 / 点)。
func ToastSlideKeep(card ui.Element, at time.Time, frame func(), keep bool) ui.Element {
	if !MotionOn() || at.IsZero() {
		return card
	}
	age := Since(at)
	if keep && age > DurSlow {
		return card
	}
	if age > 6*time.Second {
		return card.Opacity(0).Translate(40, 0)
	}
	if age > 3800*time.Millisecond {
		p := float32(age-3800*time.Millisecond) / float32(DurSlow)
		if p >= 1 {
			return card.Opacity(0).Translate(40, 0)
		}
		if frame != nil {
			frame()
		}
		ease := p * p
		return card.Opacity(1-ease).Translate(24*ease, 0)
	}
	p := float32(age) / float32(DurSlow)
	if p >= 1 {
		return card
	}
	if frame != nil {
		frame()
	}
	ease := 1 - (1-p)*(1-p)
	return card.Opacity(ease).Translate(24*(1-ease), 0)
}

// DragLift 拖拽源上浮与加深阴影。
func DragLift(row ui.Element, dragging bool) {
	if !MotionOn() || !dragging {
		return
	}
	row.Translate(0, -3)
	for _, s := range T.Shadow2() {
		row.Shadow(s.X, s.Y+2, s.Blur+2, s.Spread, s.Color)
	}
}

// OkPop 成功勾弹出(spring 缩放)。
func OkPop(icon ui.Element, at time.Time, frame func()) {
	if !MotionOn() || at.IsZero() {
		return
	}
	p := float32(Since(at)) / float32(DurBase)
	if p >= 1 {
		return
	}
	if frame != nil {
		frame()
	}
	var sc float32 = 1
	switch {
	case p < .45:
		sc = .4 + p/.45*.78
	case p < .75:
		sc = 1.18 - (p-.45)/.3*.18
	default:
		sc = 1
	}
	icon.Scale(sc, sc)
}

// FoldHeight 折叠区高度插值(避免挤动下方:绝对定位 + 高度动画)。
func FoldHeight(el ui.Element, open bool, fullH float32, toggledAt time.Time, frame func()) {
	if !MotionOn() || toggledAt.IsZero() {
		if !open {
			el.Height(0).Opacity(0).Clip()
		}
		return
	}
	p := float32(Since(toggledAt)) / float32(DurSlow)
	if p >= 1 {
		if open {
			return
		}
		el.Height(0).Opacity(0).Clip()
		return
	}
	if frame != nil {
		frame()
	}
	io := .5 - .5*float32(math.Cos(float64(p*math.Pi)))
	var h, op float32
	if open {
		h, op = fullH*io, io
	} else {
		h, op = fullH*(1-io), 1-io
	}
	el.Height(h).Opacity(op).Clip()
}
