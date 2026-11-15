package app

import (
	"fmt"
	"sort"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/customize"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// mcpView 合并会话里的 mcp_status 与自定义页配置。
type mcpView struct {
	Name, Detail string
	Conn         core.McpConn
	On           bool
	Tools        int
	Tokens       uint64
	OnDemand     bool
	Expand       bool
	ToolsList    []core.McpToolUsage
}

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
			a.send(core.CmdRefreshMcp{})
		}
	})
	return ch
}

func (a *App) mcpEnabledCount() int {
	rows := a.mcpRows()
	n := 0
	for _, r := range rows {
		if r.On {
			n++
		}
	}
	if n == 0 && len(a.st.Mcp) > 0 {
		for _, s := range a.st.Mcp {
			if s.Conn != core.McpDisabled {
				n++
			}
		}
	}
	if n == 0 {
		return len(rows) // 演示:显示配置里启用的数量
	}
	return n
}

func (a *App) mcpPopup(c *ui.Context, anchor ui.Element) {
	a.popOpen("mcp", func(open *bool) {
		ui.PopoverBase(c, anchor, open, func(panel ui.Element) {
			a.popPanel(panel, 300).Padding(0).Clip().AttachTo(anchor, ui.AnchorTopRight, ui.AnchorBottomRight).Margin(0, 0, 6, 0)
			rows := a.mcpRows()
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
					ui.Text(c, fmt.Sprintf("已启用 %d / %d", en, total)).FontWeight(600)
					if tokSum > 0 || ratio > 0 {
						ui.Text(c, fmt.Sprintf("工具定义占上下文 %s · %.1f%%", fmtTok(tokSum), ratio)).FontSize(theme.FsCap).TextColor(t.Tx3)
					} else if a.be != nil && a.be.Name() == "claude" && len(a.st.Mcp) == 0 {
						ui.Text(c, "发送消息后才会连接 MCP;点刷新获取状态").FontSize(theme.FsCap).TextColor(t.Tx4)
					}
				})
			})
			ui.Column(c).Padding(4).Children(func() {
				for _, r := range rows {
					a.mcpRow(c, r)
				}
				kit.Sep(c)
				if a.mi(c, "mcp-manage", "sliders", "管理 MCP…", "", false).Clicked() {
					a.popup = ""
					a.screen = ScreenCustomize
					a.cu.cat = customize.MCP
				}
			})
		})
	})
}

func (a *App) mcpRow(c *ui.Context, r mcpView) {
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

func (a *App) mcpRows() []mcpView {
	st := a.cuStore()
	byName := map[string]mcpView{}
	if st != nil {
		for _, it := range st.List(customize.MCP, customize.All, "") {
			mv := mcpView{Name: it.Name, On: it.On, OnDemand: it.OnDemand, Tools: len(it.Tools)}
			mv.Detail = mcpDetailFromItem(it)
			mv.Conn = mcpConnFromItem(it.Status)
			byName[it.Name] = mv
		}
	}
	for _, s := range a.st.Mcp {
		mv, ok := byName[s.Name]
		if !ok {
			mv = mcpView{Name: s.Name, On: s.Conn != core.McpDisabled, Tools: len(s.Tools)}
		}
		mv.Conn = s.Conn
		if s.Error != "" {
			mv.Detail = s.Error
		} else if mv.Detail == "" {
			mv.Detail = fmt.Sprintf("%d 个工具", len(s.Tools))
		}
		if len(s.Tools) > 0 {
			mv.Tools = len(s.Tools)
		}
		byName[s.Name] = mv
	}
	for _, tl := range a.st.Usage.McpTools {
		mv := byName[tl.Server]
		mv.Tokens += tl.Tokens
		mv.ToolsList = append(mv.ToolsList, tl)
		byName[tl.Server] = mv
	}
	var names []string
	for n := range byName {
		names = append(names, n)
	}
	sort.Strings(names)
	out := make([]mcpView, 0, len(names))
	for _, n := range names {
		out = append(out, byName[n])
	}
	return out
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
