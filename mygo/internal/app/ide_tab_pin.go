package app

import (
	"github.com/egoist/mygo/ui"

	"github.com/ixxxxoooo/minicode/internal/diag"
)

var tabPinLog = diag.L("ide-tab")

// setTabPinned 固定或取消固定标签:固定在左缘,并取消预览状态。
func (a *App) setTabPinned(tb *ideTab, pin bool) {
	if tb == nil || tb.virt != "" {
		return
	}
	tb.pinned = pin
	if pin {
		tb.preview = false
	}
	a.ide.tabs = reorderPinnedTabs(a.ide.tabs)
	if a.ide.splitDir != 0 {
		a.ide.tabs2 = reorderPinnedTabs(a.ide.tabs2)
	}
	tabPinLog.Info("标签固定", "path", tb.path, "pin", pin)
	a.toast = map[bool]string{true: "已固定标签", false: "已取消固定"}[pin]
}

func reorderPinnedTabs(tabs []*ideTab) []*ideTab {
	pin, rest := []*ideTab{}, []*ideTab{}
	for _, t := range tabs {
		if t != nil && t.pinned {
			pin = append(pin, t)
		} else {
			rest = append(rest, t)
		}
	}
	return append(pin, rest...)
}

// pinTabMenu 在标签右键菜单里写「固定标签 / 取消固定」。
func (a *App) pinTabMenu(m *ui.Menu, tb *ideTab) {
	if !isFileTab(tb) || tb.diff {
		return
	}
	label := map[bool]string{true: "取消固定标签", false: "固定标签"}[tb.pinned]
	if menuItem(m, label).Chosen() {
		a.setTabPinned(tb, !tb.pinned)
	}
}
