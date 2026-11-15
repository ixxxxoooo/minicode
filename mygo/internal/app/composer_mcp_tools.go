package app

import (
	"time"

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

func (a *App) mcpToolEnabled(server, tool string) bool {
	if a.proj == "" {
		return true
	}
	ds, err := customize.OpenDisk(a.proj)
	if err != nil {
		return true
	}
	return !ds.MCPToolDenied(server, tool)
}

func (a *App) toggleMcpTool(server, tool string, on bool) {
	if a.proj == "" {
		a.toast = "需要打开项目才能写 MCP 工具开关"
		return
	}
	a.bg("mcp-tool-toggle", func() func() {
		ds, err := customize.OpenDisk(a.proj)
		if err != nil {
			return func() { a.toast = "无法写入本地配置:" + err.Error() }
		}
		werr := ds.SetMCPTool(server, tool, on)
		return func() {
			if werr != nil {
				log.Warn("MCP 工具开关失败", "err", werr, "server", server, "tool", tool)
				a.toast = "写入失败:" + werr.Error()
				return
			}
			log.Info("MCP 工具开关", "server", server, "tool", tool, "on", on)
			a.send(core.CmdRefreshMcp{})
			delay := time.AfterFunc(1500*time.Millisecond, func() {
				if a.win != nil {
					a.win.Update(func() { a.send(core.CmdRefreshMcp{}) })
				}
			})
			_ = delay
		}
	})
}
