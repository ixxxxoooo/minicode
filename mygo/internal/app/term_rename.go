package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// termRen 是终端标签行内重命名状态(只存在内存)。
type termRen struct {
	on  bool // 零值 = 没在重命名(ix 零值是 0,不能单靠 ix 判断,否则第 0 个终端永远处于重命名)
	ix  int
	buf string
}

func (a *App) termDefaultName(i int) string {
	name := shellName()
	if i > 0 {
		name += " " + itoa(i+1)
	}
	return name
}

func (a *App) termDisplayName(i int) string {
	if i >= 0 && i < len(a.ide.tps) {
		if s := a.ide.tps[i].label; s != "" {
			return s
		}
	}
	return a.termDefaultName(i)
}

// termRenaming 判断第 i 个终端标签是否正在行内重命名。
func (a *App) termRenaming(i int) bool { return a.termRen.on && a.termRen.ix == i }

func (a *App) beginTermRename(i int) {
	if i < 0 || i >= len(a.ide.tps) {
		return
	}
	a.termRen = termRen{on: true, ix: i, buf: a.ide.tps[i].label}
	log.Info("终端标签重命名", "index", i)
}

func (a *App) commitTermRename() {
	r := a.termRen
	if !r.on {
		return
	}
	if r.ix < 0 || r.ix >= len(a.ide.tps) {
		a.termRen = termRen{}
		return
	}
	name := strings.TrimSpace(r.buf)
	if name == "" || name == a.termDefaultName(r.ix) {
		a.ide.tps[r.ix].label = ""
	} else {
		a.ide.tps[r.ix].label = name
	}
	a.termRen = termRen{}
}

func (a *App) termChip(c *ui.Context, i int) {
	t := a.tk
	name := a.termDisplayName(i)
	ren := a.termRenaming(i)
	if ren {
		ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
			kit.Ic(c, "terminal", kit.S12).TextColor(t.Tx3)
			in := ui.TextInputBase(c.Key("term-ren-"+itoa(i)), &a.termRen.buf).Width(120).FontSize(theme.FsCap).
				AutoFocus().OnSubmit(a.commitTermRename)
			in.OnShortcut(0, ui.KeyEscape, func() { a.termRen = termRen{} })
		})
		return
	}
	on := a.ide.termSel == "" && a.ide.tpSel == i
	ch := kit.Chip(c, name, kit.ChipOpt{Icon: "terminal", On: on})
	if ch.Clicked() {
		a.ide.termSel, a.ide.tpSel = "", i
		a.ide.tpc().focusIt = true
	}
	if ch.DoubleClicked() {
		a.beginTermRename(i)
	}
	if len(a.ide.tps) > 1 { // 中键关闭终端标签(至少保留一个)
		a.onMiddleClick(ch, "term-"+itoa(i), func() { a.askCloseTermTab(i) })
	}
	ch.ContextMenu(func(m *ui.Menu) {
		if menuItem(m, "重命名…").Chosen() {
			a.beginTermRename(i)
		}
		if menuItem(m, "关闭终端").Disabled(len(a.ide.tps) < 2).Tooltip(map[bool]string{true: "至少保留一个终端"}[len(a.ide.tps) < 2]).Chosen() {
			a.askCloseTermTab(i)
		}
	})
	if on && len(a.ide.tps) > 1 {
		kit.Ib(c, "x", kit.IbOpt{Small: true, Tip: "关闭这个终端"}).OnClick(func() { a.askCloseTermTab(i) })
	}
}
