package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// assistant 是助手正文:已结束用 markdown.View,流式中用 markdown.Stream(逐词淡入)。
func (a *App) assistant(c *ui.Context, m *core.Assistant) {
	if strings.TrimSpace(m.Text) == "" {
		return
	}
	box := ui.Column(c).Key("asst-" + m.ID)
	box.Children(func() {
		if m.Streaming {
			markdown.Stream(c, m.Text, a.mdOptions())
		} else {
			markdown.View(c, m.Text, a.mdOptions())
		}
	})
}

// turnEnd 返回 items[i] 是不是某个回合的最后一项(下一项是用户消息或已到末尾),以及该回合的用户消息 id。
func turnEnd(items []core.Item, i int) (string, bool) {
	if i+1 < len(items) {
		if _, ok := items[i+1].(*core.User); !ok {
			return "", false
		}
	}
	for k := i; k >= 0; k-- {
		if u, ok := items[k].(*core.User); ok {
			return u.ID, true
		}
	}
	return "", false
}

// turnFooter 是助手回复下面的操作条:复制 / 重试 / 分叉,右侧是模型 · 用时 · 用量 · 缓存 · 费用。
func (a *App) turnFooter(c *ui.Context, uid string) {
	t := a.tk
	stats, ok := a.st.TurnStats[uid]
	if !ok {
		return
	}
	ui.Row(c).Key("foot-"+uid).Height(22).Margin(0, 0, 0, -4).Gap(2).AlignItems(ui.Center).TextColor(t.Tx4).FontSize(theme.FsCap).Children(func() {
		kit.Ib(c, "copy", kit.IbOpt{Small: true, Tip: "复制"})
		kit.Ib(c, "refresh", kit.IbOpt{Small: true, Tip: "重试"})
		kit.Ib(c, "fork", kit.IbOpt{Small: true, Tip: "从这里分叉"})
		var parts []string
		if l := stats.ModelLabel(); l != "" {
			parts = append(parts, l)
		}
		if d, ok := a.st.Durations[uid]; ok && d > 0 {
			parts = append(parts, core.FormatElapsed(int64(d)*1000))
		}
		parts = append(parts, strings.Split(stats.ShortText(), " · ")...)
		ui.Row(c).Margin(0, 0, 0, 6).Gap(6).AlignItems(ui.Center).Children(func() {
			for i, p := range parts {
				if i > 0 {
					ui.Text(c, "·")
				}
				ui.Text(c, p).FontFeatures("tnum").SingleLine()
			}
		})
	})
}

// subagentCard 是子 Agent 卡片(设计稿 .sub):标题行 + 统计行 + 当前动作。
func (a *App) subagentCard(c *ui.Context, s *core.SubagentCard) {
	t := a.tk
	run := s.Active()
	card := ui.Column(c).Key("sub-"+s.ID).Background(t.Raised).Radius(t.RMd()).Padding(7, 10).Gap(3).Margin(2, 0)
	for _, sh := range t.Shadow1() {
		card.Shadow(sh.X, sh.Y, sh.Blur, sh.Spread, sh.Color)
	}
	card.Children(func() {
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			if run {
				kit.Spin(c, t.ToolAgent)
			} else {
				kit.Ic(c, "bot", kit.S12).TextColor(t.ToolAgent)
			}
			ui.Text(c, s.AgentType).FontWeight(600).SingleLine()
			ui.Text(c, s.Description).TextColor(t.Tx3).SingleLine().Grow(1)
			if run {
				kit.Ib(c, "stop", kit.IbOpt{Small: true, Tip: "停止"})
			} else {
				kit.Ic(c, "check", kit.S12).TextColor(t.Ok)
			}
		})
		if s.Summary != "" {
			ui.Text(c, s.Summary).PaddingStart(22).FontSize(theme.FsCap).TextColor(t.Tx3)
		}
	})
}
