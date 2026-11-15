package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// ───────── 面包屑 ─────────

func (a *App) crumbs(c *ui.Context, tb *ideTab) {
	t := a.tk
	ui.Row(c).Height(24).Shrink(0).PaddingX(14).Gap(8).AlignItems(ui.Center).TextColor(t.Tx3).FontSize(11.5).Children(func() {
		ui.Row(c).Grow(1).MinWidth(0).Gap(3).AlignItems(ui.Center).Clip().Children(func() {
			parts := strings.Split(tb.path, "/")
			for i, p := range parts {
				if i > 0 {
					kit.Ic(c, "chevR", kit.S12).TextColor(t.Tx4)
				}
				a.crumbSegment(c, tb, p, i, i == len(parts)-1)
			}
			// 光标所在的符号链:impl Router › match_path(点击打开符号选择器)
			if tb.st != nil && tb.buf != nil {
				syms := a.ide.fileSyms(tb.path)
				for _, k := range enclosing(syms, tb.st.CursorPos().Line) {
					kit.Ic(c, "chevR", kit.S12).TextColor(t.Tx4)
					b := ui.ButtonBase(c).Key("crumb-sym-" + itoa(k)).Height(18).PaddingX(3).Gap(3).Radius(t.RXs()).AlignItems(ui.Center).Cursor(ui.CursorPointer)
					if b.Hovered() {
						b.Background(t.Fill1)
					}
					if b.Clicked() {
						a.OpenQuick("symbols", "")
					}
					sy := syms[k]
					b.Children(func() {
						kit.Ic(c, "cube", kit.S12).TextColor(t.AcTx)
						ui.Text(c, sy.Name)
					})
				}
			}
		})
		a.crumbTools(c, tb)
	})
}

// ───────── 代码 ─────────

// editorTheme 把当前令牌映射成编辑器配色(随主题缓存)。
func (a *App) editorTheme() *editor.Theme {
	if a.edTheme != nil && a.edThemeKey == a.pal.ID {
		return a.edTheme
	}
	t := a.tk
	th := editor.DefaultTheme(t.UITheme())
	th.Background = ui.Transparent
	th.Text = t.Tx1
	th.LineNumber = t.Tx4
	th.ActiveLineNumber = t.Tx1
	th.ActiveLine = t.Fill1
	th.Cursor = t.Ac
	th.Selection = t.Ac.Alpha(.22)
	th.InactiveSelection = t.Ac.Alpha(.12)
	th.IndentGuide = t.Bd1
	th.ActiveIndentGuide = t.Bd3
	th.Error, th.Warning, th.Info = t.Err, t.Warn, t.Info
	th.Added, th.Deleted, th.Modified = t.Ok, t.Err, t.Info
	set := func(k editor.TokenKind, col ui.Color, italic bool) {
		th.Tokens[k] = editor.TokenStyle{Color: col, Italic: italic}
	}
	set(editor.TokenKeyword, t.SynKw, false)
	set(editor.TokenOperator, t.Tx2, false)
	set(editor.TokenPunctuation, t.Tx2, false)
	set(editor.TokenType, t.SynType, false)
	set(editor.TokenFunction, t.SynFn, false)
	set(editor.TokenString, t.SynStr, false)
	set(editor.TokenNumber, t.SynNum, false)
	set(editor.TokenConstant, t.SynNum, false)
	set(editor.TokenComment, t.SynCom, true)
	set(editor.TokenMeta, t.SynMac, false)
	a.edTheme, a.edThemeKey = th, a.pal.ID
	return th
}

// ───────── 空编辑器 ─────────

func (a *App) ideEmpty(c *ui.Context) {
	t := a.tk
	ui.Column(c).Grow(1).Justify(ui.Center).AlignItems(ui.Center).Gap(10).Children(func() {
		kit.Ic(c, "mark", 28).TextColor(t.Tx4)
		ui.Text(c, "从左侧文件树打开文件").TextColor(t.Tx3)
		ui.Row(c).Gap(6).Children(func() {
			kit.Kbd(c, "⌘", "P")
			ui.Text(c, "快速打开").TextColor(t.Tx4).FontSize(theme.FsCap)
		})
	})
}
