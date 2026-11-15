package kit

import (
	"math"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// ───────── 图标按钮 .ib ─────────

// IbOpt 是图标按钮的可选项。
type IbOpt struct {
	On       bool   // 选中态(fill-2)
	Small    bool   // .sm:20×20
	Tip      string // 悬停提示
	Color    ui.Color
	NoHover  bool
	Disabled bool // 40% 透明、不响应点击;Tip 可写原因
	Loading  bool // 图标区换成 Spin
	Error    bool // 错误色描边
}

// Ib 是 24×24 的图标按钮(设计稿 .ib):默认 tx-2,悬停 fill-1 + tx-1,选中 fill-2。
func Ib(c *ui.Context, icon string, o ...IbOpt) ui.Element {
	var op IbOpt
	if len(o) > 0 {
		op = o[0]
	}
	sz := float32(24)
	if op.Small {
		sz = 20
	}
	b := ui.ButtonBase(c).Size(sz, sz).Radius(T.RSm()).Justify(ui.Center).AlignItems(ui.Center)
	col := T.Tx2
	if op.Color != (ui.Color{}) {
		col = op.Color
	}
	if op.Disabled || op.Loading {
		b.Disabled(true).Opacity(.4).Cursor(ui.CursorDefault)
	} else {
		b.Cursor(ui.CursorPointer)
	}
	switch {
	case op.On:
		b.Background(T.Fill2)
		col = T.Tx1
	case b.Hovered() && !op.NoHover && !op.Disabled && !op.Loading:
		b.Background(T.Fill1)
		col = T.Tx1
	}
	if op.Error {
		b.Border(1.5, T.Err)
	}
	clickFocusRing(b)
	if !op.Disabled && !op.Loading {
		clickPressScale(b)
	}
	b.TextColor(col).Transition(Fade)
	if op.Tip != "" {
		b.Tooltip(op.Tip).Label(op.Tip)
	} else {
		b.Label(icon)
	}
	return b.Children(func() {
		if op.Loading {
			Spin(c, T.Ac)
		} else {
			Ic(c, icon, S14)
		}
	})
}

// ───────── 按钮 .btn ─────────

// BtnVariant 是按钮样式。
type BtnVariant int

const (
	BtnDefault     BtnVariant = iota
	BtnPri                    // 强调色实心
	BtnSoft                   // 强调色淡底
	BtnGhost                  // 无底
	BtnDanger                 // 危险色文字(带描边)
	BtnDangerGhost            // 危险色文字,无底
)

// BtnOpt 是按钮的可选项。
type BtnOpt struct {
	Variant  BtnVariant
	Small    bool   // .sm:高 20,字 11.5
	Icon     string // 前置图标(12px)
	Tip      string
	Disabled bool // 42% 透明且不响应点击
	Loading  bool // 显示旋转圈并禁点
}

// Btn 是文字按钮(设计稿 .btn)。
func Btn(c *ui.Context, text string, o ...BtnOpt) ui.Element {
	var op BtnOpt
	if len(o) > 0 {
		op = o[0]
	}
	h, px, fs := float32(24), float32(9), float32(12)
	if op.Small {
		h, px, fs = 20, 7, 11.5
	}
	b := ui.ButtonBase(c).Height(h).PaddingX(px).Radius(T.RSm()).Gap(5).AlignItems(ui.Center).FontSize(fs).Shrink(0)
	if op.Disabled {
		b.Disabled(true).Opacity(.42).Cursor(ui.CursorDefault)
	} else if op.Loading {
		b.Disabled(true).Cursor(ui.CursorDefault)
	} else {
		b.Cursor(ui.CursorPointer)
	}
	clickFocusRing(b)
	if !op.Disabled && !op.Loading {
		clickPressScale(b)
	}
	hov := b.Hovered() && !op.Disabled && !op.Loading
	col := T.Tx1
	switch op.Variant {
	case BtnPri:
		bg := T.Ac
		if hov {
			bg = T.Ac.Mix(T.Tx1, .10)
		}
		b.Background(bg).FontWeight(500)
		col = T.OnAc
	case BtnSoft:
		b.Background(T.AcSoft).FontWeight(500)
		col = T.AcTx
	case BtnGhost:
		col = T.Tx2
		if hov {
			b.Background(T.Fill1)
			col = T.Tx1
		}
	case BtnDangerGhost:
		col = T.Err
		if hov {
			b.Background(T.ErrSoft)
		}
	case BtnDanger:
		b.Background(T.Raised).Border(1, T.Bd2)
		col = T.Err
	default:
		bg := T.Raised
		if hov {
			bg = T.Raised.Mix(T.Tx1, .04)
		}
		b.Background(bg).Border(1, T.Bd2)
	}
	b.TextColor(col).Transition(Fade)
	if op.Tip != "" {
		b.Tooltip(op.Tip)
	}
	showSpin := op.Loading && btnLoadingSpin(b, col)
	return b.Children(func() {
		if showSpin {
			spinCol := col
			if op.Variant == BtnDefault || op.Variant == BtnDanger {
				spinCol = T.Ac
			}
			Spin(c, spinCol)
		} else if op.Icon != "" {
			Ic(c, op.Icon, S12)
		}
		ui.Text(c, text).SingleLine()
	})
}

// ───────── 胶囊 .chip ─────────

// ChipOpt 是胶囊的可选项。
type ChipOpt struct {
	Icon string
	On   bool
	Line bool // 带描边的白底胶囊
	Tip  string
	Car  bool // 末尾小箭头(.car)
}

// Chip 是 22 高的胶囊按钮(设计稿 .chip)。
func Chip(c *ui.Context, text string, o ...ChipOpt) ui.Element {
	var op ChipOpt
	if len(o) > 0 {
		op = o[0]
	}
	b := ui.ButtonBase(c).Height(22).PaddingX(7).Radius(T.RSm()).Gap(4).AlignItems(ui.Center).FontSize(11.5).Cursor(ui.CursorPointer).Shrink(0)
	clickFocusRing(b)
	clickPressScale(b)
	col := T.Tx2
	if op.Line {
		b.Background(T.Raised).Border(1, T.Bd2)
	}
	switch {
	case op.On:
		b.Background(T.Fill2)
		col = T.Tx1
	case b.Hovered():
		if !op.Line {
			b.Background(T.Fill1)
		}
		col = T.Tx1
	}
	b.TextColor(col).Transition(Fade)
	if op.Tip != "" {
		b.Tooltip(op.Tip)
	}
	return b.Children(func() {
		if op.Icon != "" {
			Ic(c, op.Icon, S12)
		}
		ui.Text(c, text).SingleLine()
		if op.Car {
			Ic(c, "chevD", S12).Opacity(.5).Margin(0, 0, 0, -1)
		}
	})
}

// ───────── 徽标 ─────────

// TagKind 是标签 / 计数 / 圆点的语义色。
type TagKind int

const (
	KNeutral TagKind = iota
	KAc
	KOk
	KErr
	KWarn
	KInfo
)

func tagColors(k TagKind) (bg, fg ui.Color) {
	switch k {
	case KAc:
		return T.AcSoft, T.AcTx
	case KOk:
		return T.OkSoft, T.Ok
	case KErr:
		return T.ErrSoft, T.Err
	case KWarn:
		return T.WarnSoft, T.Warn
	case KInfo:
		return T.InfoSoft, T.Info
	}
	return T.Fill2, T.Tx2
}

// Tag 是 16 高的小标签(设计稿 .tag)。
func Tag(c *ui.Context, text string, k ...TagKind) ui.Element {
	kind := KNeutral
	if len(k) > 0 {
		kind = k[0]
	}
	bg, fg := tagColors(kind)
	return ui.Row(c).Height(16).PaddingX(5).Radius(T.RXs()).Background(bg).AlignItems(ui.Center).Shrink(0).Children(func() {
		ui.Text(c, text).FontSize(10.5).FontWeight(500).TextColor(fg).SingleLine()
	})
}

// Cnt 是计数徽标(设计稿 .cnt):accent 为强调色实心。
func Cnt(c *ui.Context, n string, accent bool) ui.Element {
	bg, fg := T.Fill3, T.Tx2
	if accent {
		bg, fg = T.Ac, T.OnAc
	}
	w := float32(15)
	if r := utf8.RuneCountInString(n); r > 1 {
		w = float32(r)*6 + 8
	}
	return ui.Row(c).MinWidth(15).Width(w).Height(15).Radius(8).Background(bg).Justify(ui.Center).AlignItems(ui.Center).Shrink(0).Children(func() {
		ui.Text(c, n).FontSize(10).FontWeight(700).TextColor(fg).SingleLine()
	})
}

// Kbd 是一排按键帽(设计稿 kbd / .kbds)。
func Kbd(c *ui.Context, keys ...string) ui.Element {
	return ui.Row(c).Gap(2).Shrink(0).AlignItems(ui.Center).Children(func() {
		for _, k := range keys {
			ui.Row(c).MinWidth(16).Height(15).PaddingX(4).Radius(T.RXs()).Background(T.Fill1).Border(1, T.Bd2).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
				ui.Text(c, k).Font("monospace").FontSize(10).TextColor(T.Tx3).SingleLine()
			})
		}
	})
}

// Dot 是 6px 状态圆点。
func Dot(c *ui.Context, k TagKind) ui.Element {
	col := T.Tx4
	switch k {
	case KAc:
		col = T.Ac
	case KOk:
		col = T.Ok
	case KErr:
		col = T.Err
	case KWarn:
		col = T.Warn
	case KInfo:
		col = T.Info
	}
	return ui.Box(c).Size(6, 6).Radius(3).Background(col).Shrink(0)
}

// Sep 是水平分隔线(设计稿 .sep:1px bd-1,上下 4px)。
func Sep(c *ui.Context) ui.Element {
	return ui.Box(c).Height(1).Background(T.Bd1).Margin(theme.Sp2, 0).Shrink(0)
}

// VSep 是竖分隔线(.vsep:1×14 bd-2)。
func VSep(c *ui.Context) ui.Element {
	return ui.Box(c).Size(1, 14).Background(T.Bd2).Shrink(0)
}

// ───────── 进度 ─────────

// Meter 是 3px 的细进度条(.meter);pct 0..100。
func Meter(c *ui.Context, pct float32) ui.Element {
	return ui.Box(c).Height(3).Radius(2).Background(T.Fill2).Clip().Children(func() {
		ui.Box(c).HeightPercent(100).WidthPercent(max(0, min(100, pct))).Radius(2).Background(T.Ac)
	})
}

// Spin 是 11px 的旋转圈(.spin:强调色描边,一圈 0.8 s)。
func Spin(c *ui.Context, col ui.Color) ui.Element {
	e := ui.Box(c).Size(11, 11).Shrink(0)
	e.Draw(func(p *ui.Painter, r ui.Rect) {
		cx, cy := r.X+r.W/2, r.Y+r.H/2
		pts := make([][2]float32, 0, 24)
		for i := 0; i <= 22; i++ { // 约 3/4 圈的弧
			a := float64(i) / 22 * 4.7
			pts = append(pts, [2]float32{cx + 4.75*float32(math.Cos(a)), cy + 4.75*float32(math.Sin(a))})
		}
		p.StrokePolyline(pts, 1.5, col)
	})
	e.Rotate(e.Loop("spin", DurLoop/2, ui.Linear) * 360)
	return e
}

// Chk 是待办的勾选圆(.chk):0 未开始(描边圈)/ 1 进行中(旋转圈)/ 2 已完成(ok 实心 + 白勾)。
func Chk(c *ui.Context, state int) ui.Element {
	r := ui.Row(c).Size(14, 14).Radius(7).Justify(ui.Center).AlignItems(ui.Center).Shrink(0)
	switch state {
	case 2:
		r.Background(T.Ok).Transition(ui.ElementTransition{Colors: true, Duration: DurBase, Ease: EaseSpring})
		r.Children(func() { Ic(c, "check", 9).TextColor(T.OnAc) })
	case 1:
		r.Children(func() { Spin(c, T.Ac) })
	default:
		r.Border(1.4, T.Tx4)
	}
	return r
}

// Caret 是流式输出末尾的光标(2px 宽的强调色竖条,1 s 闪烁)。
func Caret(c *ui.Context) ui.Element {
	b := ui.Box(c).Size(2, 12).Radius(1).Background(T.Ac).Margin(0, 0, 0, 2).Shrink(0)
	if b.Loop("blink", time.Second, ui.Linear) < .5 {
		return b
	}
	return b.Opacity(0)
}

// Pulse 让运行中的文字呼吸(透明度在 55%~100% 间往返,约 1.4 秒一轮),代替设计稿里的文字流光。
func Pulse(e ui.Element) ui.Element {
	if !motionOn || !pulseOn {
		return e
	}
	x := e.Loop("pulse", DurLoop, ui.Linear)
	tri := 1 - float32(math.Abs(float64(2*x-1))) // 0 → 1 → 0
	return e.Opacity(1 - .45*tri)
}
