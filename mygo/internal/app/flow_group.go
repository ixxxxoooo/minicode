package app

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// subagentTimeline 子 Agent 内嵌时间线:已答的提问画回答卡片,其余步骤行照常。
func (a *App) subagentTimeline(c *ui.Context, items []core.Item) {
	if len(items) == 0 {
		return
	}
	ui.Column(c).Gap(6).Padding(4, 0, 0, 22).Children(func() {
		for _, it := range items {
			switch v := it.(type) {
			case *core.QuestionRequest:
				if v.Answered {
					a.askEcho(c, v)
				}
			default:
				if isStepItem(it) {
					a.stepItem(c, it)
				}
			}
		}
	})
}

// 折叠工具分组的「最新一步」:分组折叠且运行中时,标题下方 22px 显示最后一个未结束的工具。

func lastOpenTool(g []*core.ToolCall) *core.ToolCall {
	for i := len(g) - 1; i >= 0; i-- {
		if g[i].Status == core.ToolRunning {
			return g[i]
		}
	}
	return nil
}

func (a *App) groupLatest(c *ui.Context, grpID string, tc *core.ToolCall) {
	if tc == nil {
		return
	}
	t := a.tk
	sp := a.presentTool(tc)
	prev := a.tickTool[grpID]
	if prev != "" && prev != tc.ID {
		a.tickAt[grpID] = time.Now()
	}
	a.tickTool[grpID] = tc.ID
	row := ui.Row(c).Key("tick-"+grpID).Height(22).Margin(0, 0, 2, theme.FlowInset).Gap(6).AlignItems(ui.Center).
		TextColor(t.Tx3).FontSize(theme.FsCap).Clip()
	off := float32(0)
	if at, ok := a.tickAt[grpID]; ok && kit.MotionOn() {
		p := float32(time.Since(at)) / 200
		if p < 1 {
			a.frameSoon()
			off = (1 - p) * 22
		} else {
			delete(a.tickAt, grpID)
		}
	}
	row.Translate(0, off).Opacity(1 - off/22)
	row.Children(func() {
		kit.Ic(c, sp.icon, kit.S12)
		kit.RunText(ui.Text(c, sp.verb+" "+sp.obj).SingleLine().Grow(1).MinWidth(0))
	})
}
