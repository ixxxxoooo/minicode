package app

import (
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
	"github.com/ixxxxoooo/minicode/internal/workspace"
)

// ───────── 面包屑 ─────────

func (a *App) crumbs(c *ui.Context, tb *ideTab) {
	if a.wsNarrow() {
		return
	}
	t := a.tk
	ui.Row(c).Height(theme.Px(24)).Shrink(0).PaddingX(14).Gap(8).AlignItems(ui.Center).TextColor(t.Tx3).FontSize(theme.Px(11.5)).Children(func() {
		kit.HScroll(c).Key("crumb-scroll").Grow(1).MinWidth(0).Children(func() {
			ui.Row(c).Gap(3).AlignItems(ui.Center).Children(func() {
				if a.isExternalTab(tb) {
					a.fileIc(c, filepath.Base(tb.path))
					ui.Text(c, a.pathDisplay(tb.path)).SingleLine().TextColor(t.Tx2).FontSize(theme.Px(11.5))
					if !tb.diskRO {
						a.externalChip(c, tb)
					}
					return
				}
				sep := "/"
				if workspace.IsExternalPath(tb.path) {
					sep = string(filepath.Separator)
				}
				parts := strings.Split(tb.path, sep)
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
						a.crumbSymSegment(c, tb, syms, k)
					}
				}
			})
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
	a.scopeIndentGuideTheme(th)
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
