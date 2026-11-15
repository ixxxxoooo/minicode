package app

import (
	"github.com/ixxxxoooo/minicode/internal/automation"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/demo"
)

// 演示模式五页数据套切换(命令面板「演示:…」)。

func (a *App) demoSwitchCanvas(i int) {
	if i < 0 || i >= len(demo.CanvasSetNames()) {
		return
	}
	a.demoPgCanvas = i
	a.cvLibData, a.canv = nil, nil
	a.cv.verIx = 0
	a.toast = "已切换 · " + demo.CanvasSetNames()[i]
	log.Info("演示画布套", "set", i)
}

func (a *App) demoSwitchAuto(i int) {
	if i < 0 || i >= len(demo.AutomationsSetNames()) {
		return
	}
	a.demoPgAuto = i
	a.auto = automation.NewStore(demo.AutomationsFor(i, a.now()))
	a.au.sel, a.au.draftID = 0, 0
	if items := a.auto.List(); len(items) > 0 {
		if a.demoPgAuto == 0 && len(items) >= 2 {
			a.au.sel = 2
		} else {
			a.au.sel = items[0].ID
		}
		a.autoLoadDraft()
	}
	a.toast = "已切换 · " + demo.AutomationsSetNames()[i]
	log.Info("演示自动化套", "set", i)
}

func (a *App) demoSwitchBrowser(i int) {
	if i < 0 || i >= len(demo.BrowserSetNames()) {
		return
	}
	a.demoPgBrowser = i
	a.br = nil
	a.initBrowser()
	for _, u := range demo.BrowserDemoSeedHistory(i) {
		_ = a.brw().Go(u)
	}
	home := demo.BrowserDemoHome(i)
	_ = a.brw().Go(home)
	if i == 1 {
		a.brw().Fav[demo.BrowserHome] = true
		a.brw().Fav[home] = true
	}
	a.brs.st.Y, a.brs.lastURL = 0, ""
	a.toast = "已切换 · " + demo.BrowserSetNames()[i]
	log.Info("演示浏览器套", "set", i, "home", home)
}

func (a *App) demoSwitchCustomize(i int) {
	if i < 0 || i >= len(demo.CustomizeSetNames()) {
		return
	}
	a.demoPgCustomize = i
	a.cust = demo.CustomizeFor(i)
	a.cu.sel, a.cu.mk, a.cu.dOf = nil, nil, nil
	a.toast = "已切换 · " + demo.CustomizeSetNames()[i]
	log.Info("演示自定义套", "set", i)
}

func (a *App) demoSwitchUsage(i int) {
	if i < 0 || i >= len(demo.UsageSetNames()) {
		return
	}
	a.demoPgUsage = i
	if i == 1 {
		a.use.limitsState = core.LimitsIdle
	}
	a.toast = "已切换 · " + demo.UsageSetNames()[i]
	log.Info("演示用量套", "set", i)
}

// demoAutoSimBusy 让第一条自动化呈现「运行中」(界面态,非清单字段)。
func (a *App) demoAutoSimBusy() {
	if a.autoBusy == nil {
		a.autoBusy = map[int]bool{}
	}
	if items := a.auto.List(); len(items) > 0 {
		a.autoBusy[items[0].ID] = true
		a.toast = "演示:自动化任务 1 标记为运行中"
	}
}
