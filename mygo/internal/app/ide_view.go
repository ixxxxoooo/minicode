package app

import (
	"math"
	"path"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// editorPane 是 IDE 形态的编辑区:标签条 + 面包屑 + 代码 / 对比 + 底部面板。
func (a *App) editorPane(c *ui.Context, gap float32) {
	t := a.tk
	if a.mode != ModeIDE {
		// Agent 形态:编辑区收成 0 宽(IDE 形态独有,不渲染内容)
		ui.Box(c).Key("editor").Width(0).Shrink(0).Transition(morph)
		return
	}
	p := a.pane(c).Key("editor").Grow(1).Basis(0).Transition(morph)
	p.Children(func() {
		a.etabs(c)
		cur := a.ide.cur()
		switch {
		case cur == nil:
			a.ideEmpty(c)
		case cur.err != "":
			a.crumbs(c, cur)
			ui.Column(c).Grow(1).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
				ui.Text(c, cur.err).TextColor(t.Tx3)
			})
		case cur.diff:
			a.diffTab(c, cur)
		default:
			a.crumbs(c, cur)
			a.codeView(c, cur)
		}
		if a.ide.bottomOpen {
			a.bottomPanel(c)
		}
	})
}

// ───────── 标签条 ─────────

func (a *App) etabs(c *ui.Context) {
	t := a.tk
	bar := ui.Row(c).Height(theme.HTab).Shrink(0).AlignItems(ui.Stretch)
	switch t.Scheme.ID {
	case "a":
		bar.Background(t.Chrome).BorderWidth(0, 0, 1, 0).BorderColor(t.Bd2)
	case "b":
		bar.Padding(4, 4, 0, 4).Gap(2)
	default:
		bar.BorderWidth(0, 0, 1, 0).BorderColor(t.Bd1)
	}
	bar.Children(func() {
		ui.Row(c).Grow(1).MinWidth(0).ClipX().AlignItems(ui.Stretch).Children(func() {
			for _, tb := range a.ide.tabs {
				a.etab(c, tb)
			}
		})
		ui.Row(c).Gap(1).PaddingX(6).AlignItems(ui.Center).Shrink(0).Children(func() {
			kit.Ib(c, "cols", kit.IbOpt{Small: true, Tip: "向右拆分"})
			kit.Ib(c, "more", kit.IbOpt{Small: true})
		})
	})
}

func (a *App) etab(c *ui.Context, tb *ideTab) {
	t := a.tk
	on := tb.id == a.ide.active
	el := ui.ButtonBase(c).Key("etab-" + tb.id).PaddingX(10).Gap(6).AlignItems(ui.Center).Shrink(0).
		TextColor(t.Tx3).Cursor(ui.CursorDefault).Transition(kit.Fade)
	hov := el.Hovered()
	if on || hov {
		el.TextColor(t.Tx1)
	}
	switch t.Scheme.ID {
	case "a":
		el.BorderWidth(0, 1, 0, 0).BorderColor(t.Bd1)
		if on {
			el.Background(t.BG)
		}
	case "b":
		el.Radius(t.RSm()).Height(24)
		if on {
			el.Background(t.Fill2)
		}
	}
	if el.Clicked() {
		a.ide.active, a.ide.sel = tb.id, tb.path
	}
	if el.DoubleClicked() {
		tb.preview = false
	}
	el.Children(func() {
		if on && t.Scheme.ID == "a" {
			ui.Box(c).Absolute().Left(0).Right(0).Top(0).Height(2).Background(t.Ac)
		}
		if on && t.Scheme.ID == "c" {
			ui.Box(c).Absolute().Left(10).Right(10).Bottom(0).Height(2).Radius(2, 2, 0, 0).Background(t.Ac)
		}
		icon := "code"
		switch {
		case tb.diff:
			icon = "diff"
		case strings.HasSuffix(tb.path, ".toml"):
			icon = "gear"
		case strings.HasSuffix(tb.path, ".md"):
			icon = "fileText"
		}
		kit.Ic(c, icon, kit.S12)
		nm := ui.Text(c, tb.title()).SingleLine()
		if tb.preview {
			nm.Italic()
		}
		// 右侧:未保存圆点,悬停 / 当前标签显示关闭
		slot := ui.Row(c).Size(14, 14).Justify(ui.Center).AlignItems(ui.Center).Radius(t.RXs()).Shrink(0)
		if tb.dirty() && !hov {
			slot.Children(func() { ui.Box(c).Size(6, 6).Radius(3).Background(t.Tx3) })
			return
		}
		x := slot.Key("x-" + tb.id)
		if x.Hovered() {
			x.Background(t.Fill2)
		}
		if x.Clicked() {
			a.ide.closeTab(tb.id)
		}
		x.Children(func() {
			if on || hov {
				kit.Ic(c, "x", kit.S12).TextColor(t.Tx3)
			}
		})
	})
}

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

// codeView 是代码区:编辑器 + 右侧迷你地图。
func (a *App) codeView(c *ui.Context, tb *ideTab) {
	t := a.tk
	st := tb.st
	st.Theme = a.editorTheme()
	st.Font = ui.Font{Family: "monospace", Size: 12}
	st.LineHeight = 20.0 / 12.0
	st.NoScrollbars = true
	if r := tb.reveal; r != nil { // 搜索结果 / 跳转:选中命中处并滚到可见
		tb.reveal = nil
		p0 := editor.Pos{Line: r[0], Col: r[1]}
		st.GoTo(p0)
		st.Select(tb.buf.Offset(p0), tb.buf.Offset(editor.Pos{Line: r[0], Col: r[2]}))
	}
	ui.Row(c).Grow(1).MinHeight(0).Children(func() {
		editor.View(c.Key("ed-"+tb.id), st).Grow(1).MinWidth(0).FillHeight()
		a.minimap(c, tb)
	})
	_ = t
}

// minimap 是迷你地图:每行一条细线,视口滑块,点击 / 拖动滚动。
func (a *App) minimap(c *ui.Context, tb *ideTab) {
	t := a.tk
	st := tb.st
	n := tb.buf.LineCount()
	lines := make([]int, 0, n)
	for i := 0; i < n && i < 2000; i++ {
		lines = append(lines, len(strings.TrimRight(tb.buf.Line(i), " \t")))
	}
	box := ui.Box(c).Key("mm-"+tb.id).Width(64).FillHeight().Shrink(0).BorderWidth(0, 0, 0, 1).BorderColor(t.Bd1)
	total := float32(st.LineTop(n))
	view := float32(st.ViewportHeight())
	scroll := float32(st.ScrollY())
	if box.Pressed() || box.Dragging() {
		if x, y, ok := box.PointerPosition(); ok && total > 0 {
			_ = x
			h := box.Bounds().H
			rowH := float32(3.5)
			content := rowH * float32(len(lines))
			if content < 1 {
				content = 1
			}
			_ = h
			// 指针所在的行 → 把那一行放到视口中央
			target := y / content * total
			st.SetScrollY(float64(target - view/2))
		}
	}
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		rowH := float32(3.5) // 2px 线 + 1.5px 间距
		y0 := r.Y + 4
		if total > 0 && view > 0 {
			vy := y0 + scroll/total*rowH*float32(len(lines))
			vh := view / total * rowH * float32(len(lines))
			p.Fill(ui.Rect{X: r.X, Y: vy, W: r.W, H: float32(math.Max(float64(vh), 8))}, t.Fill2, 0)
		}
		for i, l := range lines {
			if l == 0 {
				continue
			}
			w := float32(math.Min(float64(l)*2.1, 100)) / 100 * (r.W - 12)
			p.Fill(ui.Rect{X: r.X + 6, Y: y0 + float32(i)*rowH, W: w, H: 2}, t.Tx3.Alpha(.32), 1)
		}
	})
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
		ui.Text(c, "HEAD ↔ 工作区").TextColor(t.Tx3)
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
