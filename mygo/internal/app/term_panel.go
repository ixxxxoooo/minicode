package app

import (
	"math"
	"os"
	"path/filepath"
	"strings"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/egoist/mygo/ui"

	"github.com/ixxxxoooo/minicode/internal/term"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 真实终端面板:界面只做三件事——量出能放下多少列 / 行并通知内核改大小、把内核的画面快照画成
// 带色文字、把按键翻译成仿真器的按键事件。shell 的启动、输出解析都在 internal/term。

const (
	termFont    = 11.5
	termLineH   = 18.4
	termHistory = 200 // 面板里最多带的历史行数
)

// termPanel 是 IDE 底部终端的状态。
type termPanel struct {
	t       *term.Term
	err     string // 启动失败的原因
	cellW   float32
	st      ui.ScrollState
	waits   int  // 等待量出大小的帧数(防止面板根本没有尺寸时空转)
	focusIt bool // 下一帧把键盘焦点给终端(刚启动 / 刚重启)
}

// redraw 让窗口重画一帧(终端输出到达时由内核的通知调用,任意 goroutine)。
func (a *App) redraw() {
	if a.win != nil {
		a.win.Update(func() {})
	}
}

// closeTerm 结束当前终端进程(切项目、重启、退出时)。
func (a *App) closeTerm() {
	tp := &a.ide.tp
	if tp.t != nil {
		tp.t.Close()
	}
	*tp = termPanel{cellW: tp.cellW}
}

// CloseTerminals 应用退出前调用:结束所有终端子进程。
func (a *App) CloseTerminals() { a.closeTerm() }

// shellName 是标签上显示的 shell 名。
func shellName() string {
	if s := os.Getenv("SHELL"); s != "" {
		return filepath.Base(s)
	}
	return "zsh"
}

// realTerminal 画真实终端(项目目录下的登录 shell)。
func (a *App) realTerminal(c *ui.Context) {
	tp := &a.ide.tp
	t := a.tk
	if tp.cellW == 0 {
		w, _ := c.MeasureText(10000, ui.Span{Text: "MMMMMMMMMM", Font: "monospace", Size: termFont})
		tp.cellW = w / 10
		if tp.cellW < 4 {
			tp.cellW = 6.9
		}
	}
	const padX, padY = 10, 6
	body := ui.Column(c).Key("termbody").Grow(1).MinHeight(0).Focusable()
	b := body.Bounds()
	cols := int((b.W - 2*padX) / tp.cellW)
	rows := int((b.H - 2*padY) / termLineH)

	if tp.t == nil && tp.err == "" && cols >= 4 && rows >= 1 {
		tm, err := term.Start(term.Options{Cwd: a.proj, Cols: cols, Rows: rows, Notify: a.redraw})
		if err != nil {
			tp.err = err.Error()
			log.Error("启动终端失败", "err", err)
		} else {
			tp.t, tp.focusIt = tm, true
		}
	}
	if tp.t != nil && cols >= 4 && rows >= 1 {
		tp.t.Resize(cols, rows)
	}
	if tp.focusIt {
		body.Focus()
		tp.focusIt = false
	}
	focused := body.Focused()

	var scr *term.Screen
	if tp.t != nil {
		scr = tp.t.Snapshot(termHistory)
		// 输入法的候选框跟着光标走
		body.TextCaret(ui.Rect{
			X: b.X + padX + float32(scr.CursorX)*tp.cellW,
			Y: b.Y + padY + float32(scr.CursorY)*termLineH - tp.st.Y,
			W: tp.cellW, H: termLineH,
		})
	}
	body.HandleInput(func(ev ui.InputEvent) bool { return a.termInput(c, ev) })
	body.Children(func() {
		switch {
		case tp.err != "":
			ui.Column(c).Grow(1).Padding(padY, padX).Children(func() {
				ui.Text(c, "终端启动失败:"+tp.err).TextColor(t.Err)
			})
			return
		case tp.t == nil:
			if tp.waits++; tp.waits < 8 {
				a.redraw() // 第一帧还没量出大小,下一帧再启动
			}
			return
		}
		exited, why := tp.t.Exited()
		if tp.st.Y >= tp.st.MaxY-1 { // 停在底部就一直跟随输出
			tp.st.Y = math.MaxFloat32
		}
		ui.Scroll(c).Key("term").TrackScroll(&tp.st).Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).Padding(padY, padX).Children(func() {
				for y, ln := range scr.Lines {
					cx := -1
					if y == scr.CursorY && !scr.CursorHidden && !exited {
						cx = scr.CursorX
					}
					ui.Row(c).Height(termLineH).Shrink(0).AlignItems(ui.Center).Children(func() {
						ui.RichText(c, termSpans(t, ln, cx, focused)...).NoWrap()
					})
				}
				if exited {
					ui.Row(c).Height(termLineH).Margin(4, 0, 0, 0).Gap(8).AlignItems(ui.Center).TextColor(t.Tx3).Children(func() {
						ui.Text(c, why+" · 按回车重新开始").NoWrap()
					})
				}
			})
		})
	})
}

// restartTerm 在同一目录重开一个 shell。
func (a *App) restartTerm() {
	a.closeTerm()
	a.redraw()
}

// termInput 把窗口输入交给终端:按键走仿真器编码(应用光标键、修饰键),文字与粘贴直接送入。
func (a *App) termInput(c *ui.Context, ev ui.InputEvent) bool {
	tp := &a.ide.tp
	if tp.t == nil {
		return false
	}
	if exited, _ := tp.t.Exited(); exited {
		if ev.Kind == ui.InputKeyDown && ev.Key == ui.KeyEnter {
			a.restartTerm()
			return true
		}
		return false
	}
	switch ev.Kind {
	case ui.InputText:
		if s := printable(ev.Text); s != "" {
			tp.t.SendText(s)
			return true
		}
	case ui.InputCommand:
		if ev.Text == "paste" {
			if s := c.ReadClipboard(); s != "" {
				tp.t.Paste(s)
			}
			return true
		}
	case ui.InputKeyDown:
		if ev.Mods&ui.Super != 0 {
			return false // ⌘ 组合留给窗口快捷键
		}
		if code, text, ok := termKey(ev.Key, ev.Mods); ok {
			tp.t.SendKey(code, text, termMods(ev.Mods))
			return true
		}
	}
	return false
}

// printable 去掉控制字符(回车、退格等由按键事件处理,免得重复发送)。
func printable(s string) string {
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return ""
		}
	}
	return s
}

func termMods(m ui.Modifiers) uv.KeyMod {
	var o uv.KeyMod
	if m&ui.Shift != 0 {
		o |= uv.ModShift
	}
	if m&ui.Ctrl != 0 {
		o |= uv.ModCtrl
	}
	if m&ui.Alt != 0 {
		o |= uv.ModAlt
	}
	return o
}

var termSpecial = map[ui.Key]rune{
	ui.KeyEnter: uv.KeyEnter, ui.KeyEscape: uv.KeyEscape, ui.KeyBackspace: uv.KeyBackspace, ui.KeyTab: uv.KeyTab,
	ui.KeyDelete: uv.KeyDelete, ui.KeyInsert: uv.KeyInsert, ui.KeyHome: uv.KeyHome, ui.KeyEnd: uv.KeyEnd,
	ui.KeyPageUp: uv.KeyPgUp, ui.KeyPageDown: uv.KeyPgDown,
	ui.KeyLeft: uv.KeyLeft, ui.KeyRight: uv.KeyRight, ui.KeyUp: uv.KeyUp, ui.KeyDown: uv.KeyDown,
	ui.KeyF1: uv.KeyF1, ui.KeyF2: uv.KeyF2, ui.KeyF3: uv.KeyF3, ui.KeyF4: uv.KeyF4, ui.KeyF5: uv.KeyF5, ui.KeyF6: uv.KeyF6,
	ui.KeyF7: uv.KeyF7, ui.KeyF8: uv.KeyF8, ui.KeyF9: uv.KeyF9, ui.KeyF10: uv.KeyF10, ui.KeyF11: uv.KeyF11, ui.KeyF12: uv.KeyF12,
}

// termKey 决定一个 KeyDown 要不要由这里发给终端:功能键始终发;Ctrl / Alt 与字母、空格组合也发
// (Ctrl-C、Alt-B 等);其余会打出字的按键不处理,等随后的 InputText 事件。
func termKey(k ui.Key, m ui.Modifiers) (code rune, text string, ok bool) {
	if r, found := termSpecial[k]; found {
		return r, "", true
	}
	if m&(ui.Ctrl|ui.Alt) == 0 {
		return 0, "", false
	}
	switch {
	case k >= ui.KeyA && k <= ui.KeyZ:
		return 'a' + rune(k-ui.KeyA), "", true
	case k >= ui.Key0 && k <= ui.Key9:
		return '0' + rune(k-ui.Key0), "", true
	case k == ui.KeySpace:
		return ' ', "", true
	case k == ui.KeyBracketLeft:
		return '[', "", true
	case k == ui.KeyBracketRight:
		return ']', "", true
	case k == ui.KeyBackslash:
		return '\\', "", true
	case k == ui.KeySlash:
		return '/', "", true
	case k == ui.KeyMinus:
		return '-', "", true
	case k == ui.KeyPeriod:
		return '.', "", true
	case k == ui.KeyComma:
		return ',', "", true
	}
	return 0, "", false
}

// ---- 颜色与文字段 ----

var (
	ansiDark = [16]string{"#6e7681", "#ff7b72", "#7ee787", "#e3b341", "#79c0ff", "#d2a8ff", "#76e3ea", "#c9d1d9",
		"#8b949e", "#ffa198", "#56d364", "#f2cc60", "#a5d6ff", "#e2c5ff", "#b3f0ff", "#f0f6fc"}
	ansiLight = [16]string{"#24292f", "#cf222e", "#1a7f37", "#9a6700", "#0969da", "#8250df", "#1b7c83", "#6e7781",
		"#57606a", "#a40e26", "#2da44e", "#bf8700", "#218bff", "#a475f9", "#3192aa", "#8c959f"}
)

// ansiColor 把调色板索引换成颜色:0-15 跟随深浅色主题,16-231 是 6×6×6 色立方,232-255 是灰阶。
func ansiColor(dark bool, i uint8) ui.Color {
	switch {
	case i < 16:
		if dark {
			return ui.Hex(ansiDark[i])
		}
		return ui.Hex(ansiLight[i])
	case i < 232:
		n := int(i) - 16
		lv := func(v int) uint8 {
			if v == 0 {
				return 0
			}
			return uint8(55 + v*40)
		}
		return ui.RGB(lv(n/36), lv(n/6%6), lv(n%6))
	default:
		g := uint8(8 + (int(i)-232)*10)
		return ui.RGB(g, g, g)
	}
}

// termColor 解析一个终端颜色;默认色返回 def。
func termColor(t *theme.Tokens, c term.Color, def ui.Color) ui.Color {
	switch c.Kind {
	case 1:
		return ansiColor(t.Dark(), c.Idx)
	case 2:
		return ui.RGB(c.R, c.G, c.B)
	}
	return def
}

// termSpans 把一行的样式段变成富文本片段;cx >= 0 时在该列画光标(块状,越过行尾补空格)。
func termSpans(t *theme.Tokens, ln term.Line, cx int, focused bool) []ui.Span {
	var out []ui.Span
	mk := func(r term.Run, text string) ui.Span {
		fg, bg := termColor(t, r.Fg, t.Tx1), termColor(t, r.Bg, ui.Color{})
		if r.Reverse { // 反色:前景取原背景(默认背景用面板底色),背景取原前景
			fg, bg = termColor(t, r.Bg, t.BG), termColor(t, r.Fg, t.Tx1)
		}
		if r.Dim {
			fg.A = uint8(float32(fg.A) * 0.6)
		}
		sp := ui.Span{Text: text, Font: "monospace", Size: termFont, Color: fg, Background: bg,
			Italic: r.Italic, Underline: r.Underline, Strikethrough: r.Strike}
		if r.Bold {
			sp.Weight = 700
		}
		return sp
	}
	cursor := func(base term.Run, text string) ui.Span {
		sp := mk(base, text)
		sp.Background, sp.Color = t.Tx1, t.BG
		if !focused {
			sp.Background = ui.RGBA(t.Tx1.R, t.Tx1.G, t.Tx1.B, 0.35)
		}
		return sp
	}
	col, drawn := 0, false
	for _, r := range ln {
		if cx < 0 || drawn || col+ansi.StringWidth(r.Text) <= cx {
			out = append(out, mk(r, r.Text))
			col += ansi.StringWidth(r.Text)
			continue
		}
		// 光标落在这一段里:拆成「前 / 光标字符 / 后」
		var pre, cur, post strings.Builder
		for _, ch := range r.Text {
			w := ansi.StringWidth(string(ch))
			switch {
			case col+w <= cx:
				pre.WriteRune(ch)
			case col <= cx && !drawn:
				cur.WriteRune(ch)
				drawn = true
			default:
				post.WriteRune(ch)
			}
			col += w
		}
		if pre.Len() > 0 {
			out = append(out, mk(r, pre.String()))
		}
		out = append(out, cursor(r, cur.String()))
		if post.Len() > 0 {
			out = append(out, mk(r, post.String()))
		}
	}
	if cx >= 0 && !drawn {
		if pad := cx - col; pad > 0 {
			out = append(out, mk(term.Run{}, strings.Repeat(" ", pad)))
		}
		out = append(out, cursor(term.Run{}, " "))
	}
	if len(out) == 0 {
		out = append(out, ui.Span{Text: " ", Font: "monospace", Size: termFont})
	}
	return out
}
