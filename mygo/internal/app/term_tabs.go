package app

import (
	"github.com/egoist/mygo/ui"

	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// termDotKind 是竖向列表行首状态点语义。
type termDotKind int

const (
	termDotIdle termDotKind = iota
	termDotRun
	termDotFail
	termDotExit
)

func (a *App) termDotKind(tp *termPanel) termDotKind {
	if tp == nil {
		return termDotIdle
	}
	if tp.err != "" {
		return termDotExit
	}
	if tp.t == nil {
		return termDotIdle
	}
	ex, _ := tp.t.Exited()
	if ex {
		return termDotExit
	}
	if tp.t.CmdRunning() {
		return termDotRun
	}
	if _, failed, ok := tp.t.LastCmdDone(); ok && failed {
		return termDotFail
	}
	return termDotIdle
}

func termDotColor(k termDotKind, t *theme.Tokens) ui.Color {
	switch k {
	case termDotRun:
		return t.Ac
	case termDotFail:
		return t.Err
	case termDotExit:
		return t.Tx4
	default:
		return t.Ok
	}
}

// termTabBar 在面板头部画横向终端标签(含滚动与压缩)。
func (a *App) termTabBar(c *ui.Context) {
	if a.proj == "" {
		kit.Chip(c, shellName(), kit.ChipOpt{Icon: "terminal", On: a.ide.termSel == ""}).OnClick(func() { a.ide.termSel = "" })
		return
	}
	n := len(a.ide.tps)
	if n == 0 {
		n = 1
	}
	texts := make([]string, n)
	for i := 0; i < n; i++ {
		texts[i] = a.termTabRawTitle(i)
	}
	ui.Row(c).Grow(1).MinWidth(0).AlignItems(ui.Center).Children(func() {
		holder := ui.Row(c).Key("term-tabs-holder").Grow(1).MinWidth(0).AlignItems(ui.Center)
		avail := holder.Bounds().W
		widths := termTabWidths(texts, avail)
		holder.Children(func() {
			strip := ui.ScrollHorizontal(c).Key("term-tabs-scroll").TrackScroll(&a.ide.termTabSt).Grow(1).MinWidth(0).Scrollbars(false)
			w := strip.Bounds().W
			if w > 0 && a.ide.termTabIx != a.ide.tpSel {
				var left float32
				if a.ide.termTabW != nil {
					for i := 0; i < a.ide.tpSel; i++ {
						if ww, ok := a.ide.termTabW[i]; ok {
							left += ww
						} else if i < len(widths) {
							left += widths[i]
						}
					}
					wid := a.ide.termTabW[a.ide.tpSel]
					if wid == 0 && a.ide.tpSel < len(widths) {
						wid = widths[a.ide.tpSel]
					}
					switch {
					case wid == 0:
					case left < a.ide.termTabSt.X:
						a.ide.termTabSt.X = left
					case left+wid > a.ide.termTabSt.X+w:
						a.ide.termTabSt.X = left + wid - w
					}
				}
				a.ide.termTabIx = a.ide.tpSel
			}
			strip.Children(func() {
				ui.Row(c).Gap(1).AlignItems(ui.Center).Children(func() {
					for i := 0; i < n; i++ {
						a.termChipSized(c, i, widths[i])
					}
				})
			})
		})
	})
}

// termVerticalList 画右侧竖向终端列表。
func (a *App) termVerticalList(c *ui.Context) {
	t := a.tk
	w := a.termListWidth()
	col := ui.Column(c).Key("term-vlist").Width(w).Shrink(0).Background(t.BG).
		BorderWidth(0, 0, 0, 1).BorderColor(t.Bd1).MinHeight(0)
	col.Children(func() {
		a.resizeHandle(c, lyTermList, "l", w, termListDefW, termListMinW, termListMaxW, func(v float32) {
			a.ly.termList = v
		})
		kit.VScroll(c).Key("term-vscroll").Grow(1).Children(func() {
			ui.Column(c).Children(func() {
				for i := range a.ide.tps {
					a.termSideRow(c, i, w-16)
				}
			})
		})
	})
}

func (a *App) termSideRow(c *ui.Context, i int, textW float32) {
	t := a.tk
	if i < 0 || i >= len(a.ide.tps) {
		return
	}
	tp := a.ide.tps[i]
	raw := a.termTabRawTitle(i)
	show := termTabTitle(raw, textW)
	on := a.ide.termSel == "" && a.ide.tpSel == i
	ren := a.termRenaming(i)
	row := kit.RowEl(c, on).Key("term-side-" + itoa(i)).Height(theme.Px(28)).PaddingX(8).Gap(6).AlignItems(ui.Center)
	if ren {
		row.Children(func() {
			ui.Box(c).Size(6, 6).Radius(3).Background(termDotColor(a.termDotKind(tp), t))
			in := ui.TextInputBase(c.Key("term-side-ren-"+itoa(i)), &a.termRen.buf).Grow(1).MinWidth(0).FontSize(theme.FsCap).
				AutoFocus().OnSubmit(a.commitTermRename)
			in.OnShortcut(0, ui.KeyEscape, func() { a.termRen = termRen{} })
		})
	} else {
		row = row.Tooltip(raw)
		row.OnClick(func() {
			a.ide.termSel, a.ide.tpSel = "", i
			a.ide.tpc().focusIt = true
		})
		row.OnShortcut(0, ui.KeyF2, func() { a.beginTermRename(i) })
		if row.DoubleClicked() {
			a.beginTermRename(i)
		}
		kit.CtxMenu(row, func(m *ui.Menu) { a.termTabContextMenu(m, i) })
		row.Children(func() {
			ui.Box(c).Size(6, 6).Radius(3).Background(termDotColor(a.termDotKind(tp), t))
			ui.Text(c, show).SingleLine().Grow(1).MinWidth(0).TextColor(map[bool]ui.Color{true: t.Tx1, false: t.Tx2}[on])
			if len(a.ide.tps) > 1 && (on || row.Hovered()) {
				kit.Ib(c, "x", kit.IbOpt{Small: true, Tip: "关闭这个终端"}).OnClick(func() { a.askCloseTermTab(i) })
			}
		})
	}
}

// termTabContextMenu 是终端标签 / 竖向行的右键菜单。
func (a *App) termTabContextMenu(m *ui.Menu, i int) {
	raw := a.termTabRawTitle(i)
	if menuItem(m, "重命名…").Trailing("F2").Chosen() {
		a.beginTermRename(i)
	}
	if menuItem(m, "复制标题").Disabled(raw == "").Chosen() {
		a.copyText = raw
		log.Info("复制终端标题", "index", i)
	}
	if menuItem(m, "关闭终端").Disabled(len(a.ide.tps) < 2).Tooltip(map[bool]string{true: "至少保留一个终端"}[len(a.ide.tps) < 2]).Chosen() {
		a.askCloseTermTab(i)
	}
	if menuItem(m, "关闭其他").Disabled(len(a.ide.tps) < 2).Chosen() {
		a.closeOtherTermTabs(i)
	}
}

func (a *App) closeOtherTermTabs(keep int) {
	for i := len(a.ide.tps) - 1; i >= 0; i-- {
		if i != keep {
			a.closeTermTab(i)
		}
	}
	log.Info("关闭其他终端", "keep", keep, "remaining", len(a.ide.tps))
}

func (a *App) setTermTabsMode(mode int) {
	mode = min(max(mode, 0), 2)
	a.prefs.Set(pTermTabs, mode)
	log.Info("切换终端标签排列", "mode", mode)
	a.redraw()
}

// ensureTermTabsDemo 为截图 / 调试造多个带长标题的终端。
func (a *App) ensureTermTabsDemo(n int) {
	if n < 1 {
		n = 3
	}
	if n > 8 {
		n = 8
	}
	a.SetMode(ModeIDE)
	a.ide.bottomOpen, a.ide.bottomTab = true, "term"
	for len(a.ide.tps) < n {
		a.newTerm()
	}
	samples := []struct {
		label, demo string
	}{
		{"", ""},
		{"", "~/Users/me/Projects/router-demo/src"},
		{"", "/opt/homebrew/bin/cargo test --workspace --all-features"},
		{"", "中文超长标题演示终端会话名"},
		{"deploy", ""},
	}
	for i := 0; i < n && i < len(samples); i++ {
		tp := a.ide.tps[i]
		tp.label = samples[i].label
		tp.demoTitle = samples[i].demo
	}
	a.ide.tpSel = min(a.ide.tpSel, n-1)
}
