package app

import (
	"path/filepath"
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

// termBaseName 是没有重命名 / 终端标题时的基础名:启动目录名,没有目录就用 shell 名。
func (a *App) termBaseName(i int) string {
	if i >= 0 && i < len(a.ide.tps) && a.ide.tps[i].cwd != "" {
		return filepath.Base(a.ide.tps[i].cwd)
	}
	if a.proj != "" {
		return filepath.Base(a.proj)
	}
	return shellName()
}

// termDefaultName 是第 i 个终端的默认名:基础名与更靠前的标签重名时追加序号(「项目 2」),
// 竖向列表里同项目开多个终端也分得清。
func (a *App) termDefaultName(i int) string {
	name := a.termBaseName(i)
	for j := 0; j < i && j < len(a.ide.tps); j++ {
		if a.termTabRawTitle(j) == name {
			return name + " " + itoa(i+1)
		}
	}
	return name
}

func (a *App) termDisplayName(i int) string { return a.termTabRawTitle(i) }

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

func (a *App) termChipSized(c *ui.Context, i int, tabW float32) {
	t := a.tk
	raw := a.termTabRawTitle(i)
	textW := max(tabW-termTabPadX-8, 16)
	show := termTabTitle(raw, textW)
	ren := a.termRenaming(i)
	if ren {
		ui.Row(c).Gap(4).AlignItems(ui.Center).MaxWidth(tabW).Children(func() {
			kit.Ic(c, "terminal", kit.S12).TextColor(t.Tx3)
			in := ui.TextInputBase(c.Key("term-ren-"+itoa(i)), &a.termRen.buf).Grow(1).MinWidth(0).FontSize(theme.FsCap).
				AutoFocus().OnSubmit(a.commitTermRename)
			in.OnShortcut(0, ui.KeyEscape, func() { a.termRen = termRen{} })
		})
		return
	}
	on := a.ide.termSel == "" && a.ide.tpSel == i
	chip := ui.ButtonBase(c).Key("term-chip-" + itoa(i)).Height(theme.Px(22)).PaddingX(7).Radius(t.RSm()).Gap(4).
		AlignItems(ui.Center).FontSize(theme.Px(11.5)).Cursor(ui.CursorPointer).Shrink(0).MaxWidth(tabW).Tooltip(raw)
	col := t.Tx2
	if on {
		chip.Background(t.Fill2)
		col = t.Tx1
	} else if chip.Hovered() {
		chip.Background(t.Fill1)
		col = t.Tx1
	}
	chip.TextColor(col)
	chip.Children(func() {
		kit.Ic(c, "terminal", kit.S12)
		ui.Text(c, show).SingleLine().MinWidth(0)
	})
	if chip.Clicked() {
		a.ide.termSel, a.ide.tpSel = "", i
		a.ide.tpc().focusIt = true
	}
	if chip.DoubleClicked() {
		a.beginTermRename(i)
	}
	if len(a.ide.tps) > 1 {
		a.onMiddleClick(chip, "term-"+itoa(i), func() { a.askCloseTermTab(i) })
	}
	kit.CtxMenu(chip, func(m *ui.Menu) { a.termTabContextMenu(m, i) })
	if w := chip.Bounds().W; w > 0 {
		if a.ide.termTabW == nil {
			a.ide.termTabW = map[int]float32{}
		}
		a.ide.termTabW[i] = w
	}
	if on && len(a.ide.tps) > 1 {
		kit.Ib(c, "x", kit.IbOpt{Small: true, Tip: "关闭这个终端"}).OnClick(func() { a.askCloseTermTab(i) })
	}
}
