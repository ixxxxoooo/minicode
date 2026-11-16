package app

import (
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/diag"
)

var mcpLog = diag.L("mcp")

// mcpRefreshOpts 控制一次 MCP 刷新行为。
type mcpRefreshOpts struct {
	Force      bool
	Runtime    bool // 向会话发 mcp_status
	StartProc  bool // 启动进程再查
	ReloadDisk bool // 重读配置
}

func (a *App) mcpRefresh(opts mcpRefreshOpts) {
	if a == nil {
		return
	}
	if !opts.Force && !a.mcpRefreshAt.IsZero() && time.Since(a.mcpRefreshAt) < mcpDebounce {
		return
	}
	a.mcpRefreshAt = time.Now()
	if opts.ReloadDisk {
		a.cuEnsureDisk()
		if ds := a.cuDisk(); ds != nil {
			a.bg("重载MCP配置", func() func() {
				ds.ReloadMCPConfig()
				return func() {}
			})
		}
	}
	runtime := opts.Runtime
	if runtime && !a.mcpRuntimeReady() && !opts.StartProc {
		runtime = false
	}
	// 只有真的会向 Claude 查运行时状态才进入「刷新中」(没有进程时后端不会回事件,否则会一直转)
	a.mcpRefreshing = runtime
	if a.be == nil {
		a.mcpRefreshing = false
		return
	}
	if a.be.Name() == "demo" {
		a.mcpRefreshing = true
		a.send(core.CmdRefreshMcp{Runtime: true, Force: true})
		return
	}
	if a.be.Name() != "claude" {
		a.mcpRefreshing = false
		return
	}
	a.send(core.CmdRefreshMcp{Runtime: runtime, StartProc: opts.StartProc, Force: true})
	mcpLog.Info("刷新MCP", "runtime", runtime, "startProc", opts.StartProc)
}

func (a *App) mcpOnOpenPopup() {
	a.mcpDenyReload()
	a.mcpRefresh(mcpRefreshOpts{ReloadDisk: true, Runtime: a.mcpRuntimeReady()})
}

func (a *App) mcpOnOpenManage() {
	a.mcpRefresh(mcpRefreshOpts{Force: true, ReloadDisk: true, Runtime: a.mcpRuntimeReady()})
}

func (a *App) mcpManualRefresh(startProc bool) {
	a.mcpRefresh(mcpRefreshOpts{Force: true, ReloadDisk: true, Runtime: true, StartProc: startProc})
}

func (a *App) mcpAfterTurnEnd() {
	if a.mcpTurnRefDone || a.be == nil {
		return
	}
	a.mcpTurnRefDone = true
	a.mcpRefresh(mcpRefreshOpts{ReloadDisk: true, Runtime: a.mcpRuntimeReady()})
}

func (a *App) onMcpServers() {
	a.mcpRefreshing = false
	a.mcpRuntimeAt = a.now().Unix()
	// 对账:后端报告的状态与刚点的开关一致就不再需要覆盖;不一致的留到超时(后端可能还没处理完)
	for n, ov := range a.mcpOn {
		if time.Since(ov.at) > mcpOverrideTTL {
			delete(a.mcpOn, n)
			continue
		}
		for _, sv := range a.st.Mcp {
			if sv.Name == n && (sv.Conn == core.McpDisabled) == !ov.on {
				delete(a.mcpOn, n)
			}
		}
	}
}
