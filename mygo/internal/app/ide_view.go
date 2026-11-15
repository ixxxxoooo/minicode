package app

import (
	"path"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// ───────── 面包屑 ─────────

func (a *App) crumbs(c *ui.Context, tb *ideTab) {
	t := a.tk
	ui.Row(c).Height(24).Shrink(0).PaddingX(14).Gap(3).AlignItems(ui.Center).TextColor(t.Tx3).FontSize(11.5).Children(func() {
		parts := strings.Split(tb.path, "/")
		for i, p := range parts {
			if i > 0 {
				kit.Ic(c, "chevR", kit.S12).TextColor(t.Tx4)
			}
			x := ui.Text(c, p)
			if i == len(parts)-1 {
				x.TextColor(t.Tx1).FontWeight(500)
			}
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

// ───────── 对比标签 ─────────

func (a *App) diffTab(c *ui.Context, tb *ideTab) {
	t := a.tk
	dv := tb.dv
	dv.Theme = a.editorTheme()
	dv.Font = ui.Font{Family: "monospace", Size: 12}
	adds, dels := dv.Stats()
	ui.Row(c).Height(30).Shrink(0).PaddingX(12).Gap(8).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(t.Bd1).Children(func() {
		kit.Ic(c, "diff", kit.S12)
		ui.Text(c, path.Base(tb.path)).FontWeight(600)
		ui.Text(c, tb.diffLabel()).TextColor(t.Tx3)
		kit.Tag(c, "+"+itoa(adds)+" −"+itoa(dels), kit.KOk)
		ui.Spacer(c).Grow(1)
		kit.Ib(c, "chevU", kit.IbOpt{Small: true, Tip: "上一处差异"})
		kit.Ib(c, "chevD", kit.IbOpt{Small: true, Tip: "下一处差异"})
		items := []kit.SegItem{{Text: "左右"}, {Text: "统一"}}
		on := 0
		if dv.Layout == editor.DiffUnified {
			on = 1
		}
		if i := kit.Seg(c, items, on, kit.SegOpt{Key: "dvseg"}); i >= 0 {
			if i == 0 {
				dv.Layout = editor.DiffSplit
			} else {
				dv.Layout = editor.DiffUnified
			}
		}
		kit.Btn(c, "打开文件", kit.BtnOpt{Variant: kit.BtnGhost, Small: true, Icon: "ext"}).OnClick(func() {
			a.ide.openFile(tb.path, true)
		})
	})
	editor.DiffView(c.Key("dv-"+tb.id), dv).Grow(1).MinHeight(0).FillWidth()
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
