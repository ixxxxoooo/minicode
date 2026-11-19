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
			// 后台只读盘,合并回主线程做(此前后台直接改列表:既会重复追加,又和界面读取抢数据)
			a.bg("重载MCP配置", func() func() {
				fresh := ds.ReadMCPConfig()
				return func() {
					if a.cu.disk == ds { // 期间换了项目就丢弃
						ds.ApplyMCPConfig(fresh)
					}
				}
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

// mcpFetchListBG 没有会话进程时,用 claude mcp list 给配置里的服务器补连接状态(后台跑,结果回主线程合并)。
// 它会逐个探活,实测要几十秒,所以同一时刻只跑一个,自动触发的 60 秒内不重复,手动刷新(force)不受限。
//
// 手动 / 保存配置后触发的(force)若撞上正在跑的那一次:不叠着起,记下 mcpListAgain,
// 当前这次跑完后立刻再测一遍——这样连续保存几台服务器,最后一次改动一定被检测到。
func (a *App) mcpFetchListBG(force bool) {
	ds := a.cuDisk()
	if ds == nil || a.cuDemo() {
		return
	}
	if a.mcpListBusy {
		if force {
			a.mcpListAgain = true
		}
		return
	}
	if !force && time.Since(a.mcpListAt) < 60*time.Second {
		return
	}
	a.mcpListBusy, a.mcpRefreshing, a.mcpRefreshAt = true, true, time.Now()
	a.bg("查询MCP连接状态", func() func() {
		st, err := ds.FetchMCPStatus() // 只跑子进程并解析,不碰 ds;结果在主线程里写
		return func() {
			a.mcpListBusy, a.mcpRefreshing, a.mcpListAt = false, false, time.Now()
			if err != nil {
				mcpLog.Warn("查询 MCP 状态失败", "err", err)
			} else if a.cu.disk == ds {
				ds.ApplyMCPStatus(st)
			}
			if a.mcpListAgain { // 跑的时候又有人改了配置:按最新配置再测一遍
				a.mcpListAgain = false
				a.mcpFetchListBG(true)
			}
		}
	})
}

func (a *App) mcpOnOpenPopup() {
	a.mcpDenyReload()
	a.mcpRefresh(mcpRefreshOpts{ReloadDisk: true, Runtime: a.mcpRuntimeReady()})
}

func (a *App) mcpOnOpenManage() {
	a.mcpRefresh(mcpRefreshOpts{Force: true, ReloadDisk: true, Runtime: a.mcpRuntimeReady()})
	if !a.mcpRuntimeReady() {
		a.mcpFetchListBG(false)
	}
}

func (a *App) mcpManualRefresh(startProc bool) {
	a.mcpToolsNet, a.mcpToolsNetErr = nil, nil // 手动刷新:直连读到的工具也重读
	a.mcpRefresh(mcpRefreshOpts{Force: true, ReloadDisk: true, Runtime: true, StartProc: startProc})
	if !startProc && !a.mcpRuntimeReady() {
		a.mcpFetchListBG(true)
	}
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
