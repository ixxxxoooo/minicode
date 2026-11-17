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

// ctxQueryAll 一条命令同时查「上下文分类明细」(get_context_usage)与「官方账号限额」(get_usage,仅官方账号)。
// 会话进程还没起来(打开的是历史会话、惰性启动)就顺带启动它:这是用户主动点开面板 / 点刷新,
// 要的就是精确值,不该只给一句「估算,点刷新」再让人点「启动并查询」。force 为真时限额越过缓存。
func (a *App) ctxQueryAll(force bool) {
	a.ctxLoading = true
	a.send(core.CmdRefreshUsage{
		Context: true, Limits: a.subscriptionLimitsEligible(),
		Force: force, LimitsQuick: !force, StartProc: true,
	})
}

// ctxEnsureLoaded 点开上下文面板时主动查询一次(防抖;先展示缓存 / 历史估算,结果回来后自动更新)。
func (a *App) ctxEnsureLoaded() {
	now := a.now()
	if !a.ctxRefreshAt.IsZero() && now.Sub(a.ctxRefreshAt) < ctxDebounce {
		return
	}
	a.ctxRefreshAt = now
	if a.be != nil && a.be.Name() == "claude" {
		a.ctxQueryAll(false)
		return
	}
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
	if a.be != nil && a.be.Name() == "claude" {
		a.ctxQueryAll(false)
		return
	}
	a.ctxRequestQuery(false)
}

func (a *App) ctxStartAndQuery() {
	a.ctxRefreshAt = a.now()
	if a.be != nil && a.be.Name() == "claude" {
		a.ctxQueryAll(false)
		return
	}
	a.ctxRequestQuery(true)
}

func (a *App) ctxRefreshLimitsManual() {
	if a.be == nil {
		a.toast = "还没有活动会话"
		return
	}
	if a.be.Name() == "claude" {
		a.ctxQueryAll(true) // 进程没起来就顺带启动;限额越过缓存
		return
	}
	a.send(core.CmdRefreshUsage{Limits: true, Force: true})
}
