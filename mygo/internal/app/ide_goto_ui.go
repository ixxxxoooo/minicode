package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// gotoLineUI 是编辑器内「转到行」浮条状态(设计稿 ide-nav「转到行 ⌃G」)。
type gotoLineUI struct {
	open  bool
	text  string
	err   string
	info  string
	focus bool
}

var gotoLog = diag.L("ide.goto")

func (a *App) openGotoLine() {
	tb := a.ide.cur()
	if tb == nil || tb.st == nil || tb.buf == nil || tb.diff || tb.virt != "" {
		a.toast = "先打开一个文件"
		return
	}
	g := &a.ide.gotoLn
	g.open, g.err, g.info = true, "", ""
	g.focus = true
	// 输入框留空(库没有「聚焦即全选」,预填会让新输入被追加成 199 之类);当前行号放在占位符与提示里
	g.text = ""
	gotoLog.Info("打开转到行", "path", tb.path, "lines", tb.buf.LineCount())
}

func (a *App) closeGotoLine() {
	g := &a.ide.gotoLn
	g.open, g.text, g.err, g.focus = false, "", "", false
}

func (a *App) submitGotoLine() {
	g := &a.ide.gotoLn
	tb := a.ide.cur()
	if tb == nil || tb.st == nil || tb.buf == nil {
		a.closeGotoLine()
		return
	}
	n := tb.buf.LineCount()
	if n < 1 {
		return
	}
	line, col, ok := parseLineCol(strings.TrimSpace(g.text))
	if !ok {
		g.err = "请输入行号"
		return
	}
	clamped, jumped := a.jumpToLineCol(tb, line, col)
	if !jumped {
		return
	}
	if clamped {
		g.info = "共 " + itoa(n) + " 行,已跳到末行"
	}
	gotoLog.Info("转到行", "path", tb.path, "line", line+1, "col", col+1, "clamped", clamped)
	g.open, g.text, g.err, g.focus = false, "", "", false
}

func (a *App) jumpToLineCol(tb *ideTab, line, col int) (clamped, ok bool) {
	if tb == nil || tb.buf == nil || tb.st == nil {
		return false, false
	}
	n := tb.buf.LineCount()
	if n < 1 {
		return false, false
	}
	if line < 0 {
		line = 0
	}
	clamped = line >= n
	if clamped {
		line = n - 1
	}
	ln := tb.buf.Line(line)
	if col < 0 {
		col = 0
	}
	if col > len(ln) {
		col = len(ln)
	}
	p := editor.Pos{Line: line, Col: col}
	off := tb.buf.Offset(p)
	tb.st.GoTo(p)
	tb.st.Select(off, off)
	return clamped, true
}

func (a *App) gotoLineStrip(c *ui.Context, tb *ideTab) {
	g := &a.ide.gotoLn
	if !g.open {
		if g.info != "" {
			a.gotoLineNotice(c, g.info)
		}
		return
	}
	t := a.tk
	n := 0
	if tb.buf != nil {
		n = tb.buf.LineCount()
	}
	cur := 1
	if tb.st != nil && n > 0 {
		cur = tb.st.CursorPos().Line + 1
	}
	ui.Column(c).Key("goto-line").Absolute().Top(8).Left(0).Right(0).AlignItems(ui.Center).PassThrough().Children(func() {
		card := ui.Column(c).MaxWidth(520).Shrink(0).Background(t.Raised).Radius(t.RMd()).Border(1, t.Bd1).Clip()
		for _, s := range t.Shadow2() {
			card.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
		}
		kit.Float(card) // 流光:跳转卡是玻璃
		card.Children(func() {
			ui.Row(c).Height(theme.Px(36)).PaddingX(12).Gap(8).AlignItems(ui.Center).Children(func() {
				kit.Ic(c, "list", kit.S12).TextColor(t.Tx3)
				ui.Text(c, "转到").TextColor(t.Tx2).FontWeight(500)
				in := ui.TextInputBase(c.Key("goto-in"), &g.text).Width(84).Font("monospace").FontFeatures("tnum").
					Placeholder(map[bool]string{true: "—", false: itoa(cur)}[n < 1]).Disabled(n < 1)
				if g.err != "" {
					in.Border(1, t.Err)
				}
				if g.focus {
					g.focus = false
					in.Focus()
				}
				if in.Changed() {
					g.err = ""
				}
				if in.Submitted() && n >= 1 {
					a.submitGotoLine()
				}
				if in.Shortcut(0, ui.KeyEscape) {
					a.closeGotoLine()
				}
				hint := map[bool]string{true: "空文件 · 共 0 行", false: "行 1–" + itoa(n) + " · 当前 " + itoa(cur)}[n < 1]
				ui.Text(c, hint).FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine()
				ui.Spacer(c).Grow(1)
				kit.Ib(c, "x", kit.IbOpt{Small: true, Tip: "Esc 取消"}).OnClick(a.closeGotoLine)
			})
			if g.err != "" {
				ui.Row(c).Padding(4, 12, 8, 12).Gap(6).AlignItems(ui.Center).Children(func() {
					kit.Ic(c, "errc", kit.S12).TextColor(t.Err)
					ui.Text(c, g.err).FontSize(theme.FsCap).TextColor(t.Err)
				})
			}
			if g.info != "" {
				a.gotoLineNotice(c, g.info)
			}
		})
	})
}

func (a *App) gotoLineNotice(c *ui.Context, msg string) {
	t := a.tk
	ui.Row(c).Height(theme.Px(32)).PaddingX(12).Gap(8).AlignItems(ui.Center).Background(t.InfoSoft).
		BorderWidth(1, 0, 0, 0).BorderColor(t.Bd1).Children(func() {
		kit.Ic(c, "info", kit.S12).TextColor(t.Info)
		ui.Text(c, msg).Grow(1).TextColor(t.Tx2)
	})
}
