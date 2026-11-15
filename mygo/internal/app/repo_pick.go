package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// folderPicker 是多根工作区里「选一个文件夹」的下拉胶囊(图谱的仓库下拉、搜索的范围 chip 共用)。
// current < 0 表示「全部」(label 是它的文案;onlyRepos 且没有 all 文案时必须选一个);
// 选中某项回调 onPick(序号),选「全部」回调 onPick(-1)。
func (a *App) folderPicker(c *ui.Context, key string, current int, all string, onlyRepos bool, onPick func(i int)) {
	t := a.tk
	label := all
	on := current >= 0 && current < len(a.repos)
	if on {
		label = a.repos[current].name
	}
	b := ui.ButtonBase(c).Key("fp-" + key).Height(22).PaddingX(8).Gap(5).Radius(11).AlignItems(ui.Center).Cursor(ui.CursorPointer).
		FontSize(theme.FsCap).Shrink(0).Transition(kit.Fade)
	switch {
	case on:
		b.Background(t.AcSoft).TextColor(t.AcTx)
	case b.Hovered():
		b.Background(t.Fill2).TextColor(t.Tx2)
	default:
		b.Background(t.Fill1).TextColor(t.Tx3)
	}
	b.OnClick(func() { a.togglePop("fp-" + key) })
	b.Children(func() {
		kit.Ic(c, "folder", kit.S12)
		ui.Text(c, label).SingleLine()
		kit.Ic(c, "chevD", kit.S12)
	})
	a.popOpen("fp-"+key, func(open *bool) {
		ui.PopoverBase(c, b, open, func(panel ui.Element) {
			a.popPanel(panel, 240).AttachTo(b, ui.AnchorTopLeft, ui.AnchorBottomLeft).Margin(4, 0, 0, 0)
			if all != "" && a.mi(c, key+"-all", "folder", strings.TrimPrefix(all, "范围:"), "", current < 0).Clicked() {
				a.popup = ""
				onPick(-1)
			}
			for i, e := range a.repos {
				if (onlyRepos && e.repo == nil) || (!onlyRepos && e.kind != "") { // 搜索范围只按顶层文件夹
					continue
				}
				br, _, _, _ := a.repoSummary(i)
				if a.mi(c, key+"-"+itoa(i), "folder", e.name, br, i == current).Clicked() {
					a.popup = ""
					onPick(i)
				}
			}
		})
	})
}

// repoCount 工作区里真正是 Git 仓库的文件夹数。
func (a *App) repoCount() int {
	n := 0
	for _, e := range a.repos {
		if e.repo != nil {
			n++
		}
	}
	return n
}
