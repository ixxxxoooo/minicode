package app

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/customize"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

func (a *App) mcpChip(c *ui.Context, wide bool) ui.Element {
	n := a.mcpEnabledCount()
	lbl := itoa(n)
	if wide {
		lbl = "MCP " + lbl
	}
	ch := kit.Chip(c, lbl, kit.ChipOpt{Icon: "plug", Tip: "MCP 服务器", On: a.popup == "mcp", Line: true})
	ch.OnClick(func() {
		open := a.popup != "mcp"
		a.togglePop("mcp")
		if open && a.popup == "mcp" {
			a.mcpOnOpenPopup()
		}
	})
	return ch
}

func (a *App) mcpEnabledCount() int {
	rows := a.mcpRowsUnified()
	n := 0
	for _, r := range rows {
		if r.On {
			n++
		}
	}
	return n
}

func (a *App) mcpPopup(c *ui.Context, anchor ui.Element) {
	a.popOpen("mcp", func(open *bool) {
		ui.PopoverBase(c, anchor, open, func(panel ui.Element) {
			a.popPanel(panel, 300).Padding(0).Clip().AttachTo(anchor, ui.AnchorTopRight, ui.AnchorBottomRight).Margin(0, 0, 6, 0)
			rows := a.mcpRowsUnified()
			en, total := 0, len(rows)
			var tokSum uint64
			for _, r := range rows {
				if r.On {
					en++
				}
				tokSum += r.Tokens
			}
			ratio := float32(0)
			if a.st.Usage.ContextSize > 0 {
				ratio = float32(tokSum) / float32(a.st.Usage.ContextSize) * 100
			}
			t := a.tk
			ui.Row(c).Padding(8, 10).BorderWidth(0, 0, 1, 0).BorderColor(t.Bd1).Children(func() {
				ui.Column(c).Grow(1).Children(func() {
					ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
						ui.Text(c, fmt.Sprintf("已启用 %d / %d", en, total)).FontWeight(600).Grow(1)
						kit.Ib(c, "refresh", kit.IbOpt{Small: true, Tip: "刷新 MCP 状态"}).OnClick(func() {
							a.mcpManualRefresh(false)
							if !a.mcpRuntimeReady() {
								a.toast = "已重读配置;发送第一条消息后可查看连接状态"
							}
						})
					})
					if a.mcpRefreshing {
						ui.Text(c, "正在刷新…").FontSize(theme.FsCap).TextColor(t.Tx3)
					} else if tokSum > 0 || ratio > 0 {
						ui.Text(c, fmt.Sprintf("工具定义占上下文 %s · %.1f%%", fmtTok(tokSum), ratio)).FontSize(theme.FsCap).TextColor(t.Tx3)
					} else if !a.mcpRuntimeReady() {
						ui.Text(c, "会话未启动;已显示配置层服务器,点刷新重读配置").FontSize(theme.FsCap).TextColor(t.Tx4)
					} else if a.mcpRuntimeAt > 0 {
						ui.Text(c, "运行时状态更新于 "+relTime(a.now(), time.Unix(a.mcpRuntimeAt, 0))).FontSize(theme.FsCap).TextColor(t.Tx4)
					}
				})
			})
			ui.Column(c).Padding(4).Children(func() {
				if len(rows) == 0 {
					ui.Text(c, "没有配置 MCP 服务器").FontSize(theme.FsCap).TextColor(t.Tx3).Padding(8, 12)
				}
				for _, r := range rows {
					a.mcpRow(c, r)
				}
				kit.Sep(c)
				if !a.mcpRuntimeReady() && len(rows) > 0 {
					kit.Btn(c, "启动并查询连接", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(func() {
						a.mcpManualRefresh(true)
					})
				}
				if a.mi(c, "mcp-manage", "sliders", "管理 MCP…", "", false).Clicked() {
					a.popup = ""
					a.screen = ScreenCustomize
					a.cu.cat = customize.MCP
				}
			})
		})
	})
}

func (a *App) mcpRow(c *ui.Context, r McpRow) {
	t := a.tk
	key := "mcp-" + r.Name
	exp := a.mcpExp[key]
	row := ui.ButtonBase(c).Key(key).MinHeight(36).Padding(5, 8).Gap(6).Radius(t.RSm()).AlignItems(ui.Center).WidthPercent(100).Cursor(ui.CursorPointer)
	if row.Hovered() {
		row.Background(t.Fill1)
	}
	row.Children(func() {
		if kit.Ib(c, "chevR", kit.IbOpt{Small: true}).Clicked() {
			a.mcpExp[key] = !exp
		}
		chevRot := float32(0)
		if exp {
			chevRot = 90
		}
		kit.Ic(c, "chevR", kit.S12).TextColor(t.Tx4).Rotate(chevRot)
		a.mcpDot(c, r.Conn)
		ui.Column(c).Grow(1).MinWidth(0).Children(func() {
			ui.Text(c, r.Name).FontWeight(500).SingleLine()
			ui.Text(c, r.Detail).FontSize(theme.FsMicro).TextColor(t.Tx3).SingleLine()
		})
		if kit.Tog(c, r.On, key+"-tog") {
			a.toggleMcpServer(r.Name, !r.On)
		}
	})
	if exp && len(r.ToolsList) > 0 {
		for _, tl := range r.ToolsList {
			a.mcpToolRow(c, r.Name, tl)
		}
	}
}

func (a *App) mcpDot(c *ui.Context, conn core.McpConn) {
	t := a.tk
	col := t.Tx4
	switch conn {
	case core.McpConnected:
		col = t.Ok
	case core.McpFailed:
		col = t.Err
	case core.McpNeedsAuth, core.McpPending:
		col = t.Warn
	case core.McpDisabled:
		col = t.Tx4
	}
	ui.Box(c).Size(7, 7).Radius(4).Background(col).Shrink(0)
}

func mcpDetailFromItem(it *customize.Item) string {
	if it.Desc != "" {
		return it.Desc
	}
	if it.OnDemand {
		return fmt.Sprintf("%d 个工具 · 按需加载", len(it.Tools))
	}
	if it.Tok != "" {
		return it.Tok
	}
	return it.Cmd
}

func mcpConnFromItem(st customize.Status) core.McpConn {
	switch st {
	case customize.StOK:
		return core.McpConnected
	case customize.StWarn:
		return core.McpNeedsAuth
	case customize.StErr:
		return core.McpFailed
	}
	return core.McpUnknown
}

func (a *App) toggleMcpServer(name string, on bool) {
	if a.be == nil {
		a.toast = "当前没有活动后端"
		return
	}
	if a.be.Name() == "demo" {
		a.send(core.CmdMcpToggle{Name: name, Enabled: on})
		a.toast = "演示模式:MCP 开关已记录"
		return
	}
	if a.be.Name() == "claude" {
		a.send(core.CmdMcpToggle{Name: name, Enabled: on})
		log.Info("MCP 切换", "server", name, "on", on)
		return
	}
	a.toast = "该运行模式不支持 MCP 热切换"
}
