package app

import (
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

const ctxDebounce = 3 * time.Second

func (a *App) ctxRequestQuery(startProc bool) {
	if a.be == nil {
		return
	}
	if a.be.Name() == "demo" {
		a.ctxLoading = true
		a.send(core.CmdRefreshUsage{Context: true, Force: true})
		return
	}
	if a.be.Name() != "claude" {
		return
	}
	a.ctxLoading = true
	a.send(core.CmdRefreshUsage{Context: true, StartProc: startProc})
}

// ctxEnsureLoaded 点开上下文面板时按防抖查询 get_context_usage(先展示缓存)。
func (a *App) ctxEnsureLoaded() {
	now := a.now()
	if !a.ctxRefreshAt.IsZero() && now.Sub(a.ctxRefreshAt) < ctxDebounce {
		return
	}
	a.ctxRefreshAt = now
	a.ctxRequestQuery(false)
}

func (a *App) ctxManualRefresh() {
	a.ctxRefreshAt = a.now()
	a.ctxRequestQuery(false)
}

func (a *App) ctxStartAndQuery() {
	a.ctxRefreshAt = a.now()
	a.ctxRequestQuery(true)
}
