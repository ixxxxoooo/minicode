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
	if !startProc && !a.sessProcLive {
		// 会话进程没起来,后端不会回应(也不该为查询而启动进程),不能挂着「刷新中」
		a.ctxLoading = false
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
	a.ctxEnsureLimitsOnOpen()
}

// ctxEnsureLimitsOnOpen 官方账号点开圆环时后台刷新订阅限额(30s 内复用缓存)。
func (a *App) ctxEnsureLimitsOnOpen() {
	if !a.subscriptionLimitsEligible() {
		return
	}
	if a.be == nil || a.be.Name() == "demo" {
		if a.cuDemo() {
			a.send(core.CmdRefreshUsage{Limits: true, LimitsQuick: true})
		}
		return
	}
	if a.be.Name() != "claude" || !a.sessProcLive {
		return
	}
	u := a.st.Usage
	if u.LimitsState == core.LimitsChecking {
		return
	}
	if age := a.now().Unix() - u.LimitsAt; u.LimitsAt > 0 && age < core.LimitsPopoverTTLSecs {
		return
	}
	if u.LimitsState == core.LimitsIdle && u.FiveHourPct == nil && u.WeeklyPct == nil && u.LimitsAt == 0 {
		// 尚未取过:立刻查
	} else if u.LimitsAt > 0 && a.now().Unix()-u.LimitsAt < core.LimitsPopoverTTLSecs {
		return
	}
	a.send(core.CmdRefreshUsage{Limits: true, LimitsQuick: true})
}

func (a *App) ctxManualRefresh() {
	a.ctxRefreshAt = a.now()
	a.ctxRequestQuery(false)
}

func (a *App) ctxStartAndQuery() {
	a.ctxRefreshAt = a.now()
	a.ctxRequestQuery(true)
}

func (a *App) ctxRefreshLimitsManual() {
	if a.be == nil {
		a.toast = "还没有活动会话"
		return
	}
	if a.be.Name() == "claude" && !a.sessProcLive {
		a.toast = "会话进程尚未启动,发送消息后可查询"
		return
	}
	a.send(core.CmdRefreshUsage{Limits: true, Force: true})
}
