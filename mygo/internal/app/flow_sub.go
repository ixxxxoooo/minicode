package app

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// subagentCard 是主会话流里的子 Agent「调用卡」:不内联展开过程,详情在工作区标签。
func (a *App) subagentCard(c *ui.Context, s *core.SubagentCard) {
	t := a.tk
	run := s.Active()
	card := ui.ButtonBase(c).Key("sub-"+s.ID).Background(t.Raised).Radius(t.RMd()).Padding(7, 10).Margin(2, 0).
		Cursor(ui.CursorPointer).Transition(kit.Fade)
	for _, sh := range t.Shadow1() {
		card.Shadow(sh.X, sh.Y, sh.Blur, sh.Spread, sh.Color)
	}
	card.OnClick(func() { a.openSubTab(s.ID) })
	card.Children(func() {
		ui.Column(c).Gap(4).WidthPercent(100).Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				a.subStatusIcon(c, s)
				ui.Text(c, s.Description).FontWeight(500).SingleLine().Shrink(1)
				kit.Tag(c, s.AgentType)
				if n := s.PendingPermissions(); n > 0 {
					kit.Tag(c, fmt.Sprintf("待确认 %d", n), kit.KWarn)
				}
				ui.Spacer(c).Grow(1)
				a.subCardActions(c, s, run)
			})
			a.subPromptBlock(c, s)
			a.subStatRow(c, s)
			if run {
				if step := s.SubLatestStep(); step != "" {
					a.subLatestTicker(c, s.ID, step)
				} else if s.Stalled(a.st.Now()) {
					ui.Text(c, "疑似停滞").PaddingStart(22).FontSize(a.FlowCapFont()).TextColor(t.Warn)
				}
			} else if strings.TrimSpace(s.Summary) != "" {
				ui.Text(c, core.ShortTitle(s.Summary)).PaddingStart(22).FontSize(a.FlowCapFont()).TextColor(t.Tx3).MaxLines(2)
			}
		})
	})
}

func (a *App) subStatusIcon(c *ui.Context, s *core.SubagentCard) {
	t := a.tk
	switch {
	case s.Active():
		kit.Spin(c, t.ToolAgent)
	case s.Status == core.SubFailed:
		kit.Ic(c, "errc", kit.S12).TextColor(t.Err)
	case s.Status == core.SubStopped:
		kit.Ic(c, "bot", kit.S12).TextColor(t.Tx4)
	default:
		kit.Ic(c, "check", kit.S12).TextColor(t.Ok)
	}
}

func (a *App) subCardActions(c *ui.Context, s *core.SubagentCard, run bool) {
	if run {
		kit.Btn(c, "查看过程", kit.BtnOpt{Small: true, Icon: "ext"}).OnClick(func() { a.openSubTab(s.ID) })
		kit.Ib(c, "stop", kit.IbOpt{Small: true, Tip: "停止"}).OnClick(func() {
			if s.TaskID != "" {
				a.send(core.CmdStopTask{TaskID: s.TaskID})
				log.Info("停止子 Agent", "task", s.TaskID, "card", s.ID)
			}
		})
	} else {
		kit.Btn(c, "查看过程", kit.BtnOpt{Variant: kit.BtnGhost, Small: true, Icon: "ext"}).OnClick(func() { a.openSubTab(s.ID) })
	}
}

func (a *App) openSubTab(id string) {
	log.Info("打开子 Agent 标签", "id", id)
	a.OpenWorkspaceTab(vSub, id)
	if a.findSub(id) != nil {
		a.subNoFollow = false
		a.subSt.Y = 1e9
	}
}

func (a *App) subStatRow(c *ui.Context, s *core.SubagentCard) {
	t := a.tk
	var meta []string
	if el, ok := s.ElapsedMs(a.st.Now()); ok {
		meta = append(meta, core.FormatElapsed(el))
	}
	if n := s.Tools(); n > 0 {
		meta = append(meta, itoa(n)+" 个工具")
	}
	if s.Tokens > 0 {
		meta = append(meta, core.FormatTokens(s.Tokens))
	}
	if s.Background {
		meta = append(meta, "后台")
	}
	if len(meta) == 0 {
		return
	}
	mt := ui.Text(c, strings.Join(meta, " · ")).PaddingStart(22).FontSize(a.FlowCapFont()).TextColor(t.Tx3).FontFeatures("tnum")
	if a.chatTier() != tierNarrow {
		mt.SingleLine()
	}
}

func (a *App) subPromptBlock(c *ui.Context, s *core.SubagentCard) {
	prompt := strings.TrimSpace(s.Prompt)
	if prompt == "" {
		return
	}
	t := a.tk
	foldKey := "subprompt:" + s.ID
	expanded := a.folds[foldKey]
	long := len([]rune(prompt)) > 120 || strings.Count(prompt, "\n") >= 2
	ui.Column(c).Gap(4).PaddingStart(22).Children(func() {
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "任务").FontSize(theme.FsCap).FontWeight(500).TextColor(t.Tx3).Shrink(0)
			ui.Spacer(c).Grow(1)
			kit.Ib(c, "copy", kit.IbOpt{Small: true, Tip: "复制任务"}).OnClick(func() { c.WriteClipboard(prompt) })
			if long {
				kit.Btn(c, map[bool]string{true: "收起", false: "展开"}[expanded], kit.BtnOpt{Variant: kit.BtnGhost, Small: true}).OnClick(func() {
					a.folds[foldKey] = !expanded
				})
			}
		})
		if expanded {
			kit.VScroll(c).MaxHeight(160).Children(func() {
				ui.Text(c, prompt).TextColor(t.Tx2).LineHeight(1.5).Selectable()
			})
		} else {
			ui.Text(c, prompt).TextColor(t.Tx2).LineHeight(1.5).MaxLines(2)
		}
	})
}

func (a *App) subLatestTicker(c *ui.Context, id, step string) {
	t := a.tk
	row := ui.Row(c).Key("sub-tick-"+id).Height(theme.Px(22)).Margin(0, 0, 0, theme.FlowInset).Gap(6).AlignItems(ui.Center).
		TextColor(t.Tx3).FontSize(theme.FsCap).Clip()
	row.Children(func() {
		kit.Spin(c, t.ToolAgent)
		kit.RunText(ui.Text(c, step).SingleLine().Grow(1).MinWidth(0))
	})
}

// compactSubRow 用于「并行 N 个」组内:不占满一张大卡,只留一行摘要 + 最新一步。
func (a *App) compactSubRow(c *ui.Context, s *core.SubagentCard) {
	t := a.tk
	run := s.Active()
	row := ui.ButtonBase(c).Key("subrow-"+s.ID).Background(t.Raised).Radius(t.RSm()).Padding(6, 8).Margin(0, 0, 4, 0).
		Cursor(ui.CursorPointer).Transition(kit.Fade)
	row.OnClick(func() { a.openSubTab(s.ID) })
	row.Children(func() {
		ui.Column(c).Gap(3).WidthPercent(100).Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				a.subStatusIcon(c, s)
				ui.Text(c, s.Description).FontWeight(500).SingleLine().Shrink(1)
				if n := s.PendingPermissions(); n > 0 {
					kit.Tag(c, fmt.Sprintf("待确认 %d", n), kit.KWarn)
				}
				ui.Spacer(c).Grow(1)
				if run {
					kit.Ib(c, "stop", kit.IbOpt{Small: true, Tip: "停止"}).OnClick(func() {
						if s.TaskID != "" {
							a.send(core.CmdStopTask{TaskID: s.TaskID})
						}
					})
				}
			})
			if run {
				if step := s.SubLatestStep(); step != "" {
					a.subLatestTicker(c, s.ID, step)
				}
			}
		})
	})
}
