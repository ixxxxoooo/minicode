package app

import (
	"strconv"

	"github.com/egoist/mygo"
	"github.com/ixxxxoooo/minicode/internal/core"
)

func (a *App) syncDockBadge() {
	if a.st == nil && len(a.live) == 0 {
		return
	}
	sum := a.brandSummarize()
	label := ""
	if sum.badge > 0 {
		if sum.badge > 99 {
			label = "99+"
		} else {
			label = strconv.Itoa(sum.badge)
		}
	}
	if label != a.brand.lastBadge {
		a.brand.lastBadge = label
		mygo.App.Dock.SetBadge(label)
		a11y := a.trayA11yLabel(sum)
		if a11y != a.brand.lastA11y {
			a.brand.lastA11y = a11y
			postA11yAnnouncement(a11y)
		}
	}
	a.syncDockProgress()
}

func (a *App) syncDockProgress() {
	if a.prefs == nil || !a.prefs.Bool(pDockProg, false) {
		dockSetProgress(0)
		return
	}
	if a.updSt.Phase == updDownload && a.updSt.Percent > 0 {
		dockSetProgress(float64(a.updSt.Percent) / 100)
		return
	}
	if a.st != nil && a.st.Phase == core.PhaseRunning {
		dockSetProgress(0.35) // 回合进行中:不确定进度时显示浅条
		return
	}
	dockSetProgress(0)
}
