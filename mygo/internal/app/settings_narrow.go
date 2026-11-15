package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

const setNavBreak = 900

func (a *App) settingsNarrow() bool { return a.winW > 0 && a.winW < setNavBreak }

// settingsTabPicker 窄窗时顶部分类下拉。
func (a *App) settingsTabPicker(c *ui.Context) {
	t := a.tk
	cur := a.setTabByID(a.setTab)
	head := kit.RowEl(c, false).Key("set-nav-drop").Gap(8).Padding(8, 10).Radius(t.RMd()).Background(t.Raised).Border(1, t.Bd2).Grow(1)
	if head.Clicked() {
		a.set.navDrop = !a.set.navDrop
	}
	head.Children(func() {
		kit.Ic(c, cur.icon, kit.S12)
		ui.Text(c, "分类: "+cur.title).Grow(1).FontWeight(600)
		kit.Ic(c, "chevD", kit.S12).TextColor(t.Tx3)
	})
	if !a.set.navDrop {
		return
	}
	ui.Column(c).Gap(1).Margin(4, 0, 0, 0).Padding(6).Radius(t.RMd()).Background(t.Raised).Border(1, t.Bd2).Children(func() {
		for _, tab := range settingsTabs {
			tab := tab
			on := tab.id == a.setTab
			row := kit.RowEl(c, on).Key("set-pick-"+tab.id).MinHeight(32).Padding(4, 8).Gap(8)
			row.OnClick(func() {
				a.setTab = tab.id
				a.set.navDrop = false
				a.set.mainSt.Y = 0
				a.set.lastTab = ""
			})
			row.Children(func() {
				kit.Ic(c, tab.icon, kit.S12)
				ui.Text(c, tab.title).Grow(1)
			})
		}
	})
	ui.Text(c, "").FontSize(theme.FsMicro) // 占位,避免后续元素重叠时少一行
}
