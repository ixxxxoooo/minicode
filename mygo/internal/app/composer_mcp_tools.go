package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/customize"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

func (a *App) mcpToolRow(c *ui.Context, server string, tl core.McpToolUsage) {
	t := a.tk
	on := a.mcpToolEnabled(server, tl.Tool)
	key := "mcpt-" + server + "-" + tl.Tool
	ui.Row(c).Key(key).Padding(4, 8, 4, 28).Gap(6).MinHeight(28).AlignItems(ui.Center).Children(func() {
		ui.Text(c, tl.Tool).Grow(1).SingleLine().FontSize(theme.FsCap)
		if tl.Loaded {
			ui.Text(c, fmtTok(tl.Tokens)).TextColor(t.Tx3).FontFeatures("tnum").FontSize(theme.FsMicro)
		} else {
			ui.Text(c, "按需").TextColor(t.Tx4).FontSize(theme.FsMicro)
		}
		if kit.Tog(c, on, key+"-tog") {
			a.toggleMcpTool(server, tl.Tool, !on)
		}
	})
}

// mcpToolEnabled 读缓存(不在界面线程读盘);缓存在弹层打开与每次切换后由 mcpDenyReload 后台刷新。
func (a *App) mcpToolEnabled(server, tool string) bool {
	return !a.mcpDeny[customize.MCPToolKey(server, tool)]
}

// mcpDenyReload 后台重读本地层的工具禁用规则。
func (a *App) mcpDenyReload() {
	if a.proj == "" {
		a.mcpDeny = nil
		return
	}
	proj := a.proj
	a.bg("mcp-deny", func() func() {
		ds, err := customize.OpenDisk(proj)
		if err != nil {
			return func() {}
		}
		set := ds.MCPDenySet()
		return func() { a.mcpDeny = set }
	})
}

func (a *App) toggleMcpTool(server, tool string, on bool) {
	if a.proj == "" {
		a.toast = "需要打开项目才能写 MCP 工具开关"
		return
	}
	key := customize.MCPToolKey(server, tool)
	if a.mcpDeny == nil {
		a.mcpDeny = map[string]bool{}
	}
	a.mcpDeny[key] = !on // 先本地生效,开关立刻翻转
	proj := a.proj
	a.bg("mcp-tool-toggle", func() func() {
		ds, err := customize.OpenDisk(proj)
		if err != nil {
			return func() {
				a.mcpDeny[key] = on
				a.toast = "无法写入本地配置:" + err.Error()
			}
		}
		werr := ds.SetMCPTool(server, tool, on)
		return func() {
			if werr != nil {
				a.mcpDeny[key] = on // 写失败:还原
				log.Warn("MCP 工具开关失败", "err", werr, "server", server, "tool", tool)
				a.toast = "写入失败:" + werr.Error()
				return
			}
			log.Info("MCP 工具开关", "server", server, "tool", tool, "on", on)
			a.mcpManualRefresh(a.mcpRuntimeReady())
		}
	})
}
