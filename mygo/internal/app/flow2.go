package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

// assistant 是助手正文:已结束用 markdown.View,流式中用 markdown.Stream(逐词淡入)。
func (a *App) assistant(c *ui.Context, m *core.Assistant) {
	if strings.TrimSpace(m.Text) == "" {
		return
	}
	box := ui.Column(c).Key("asst-" + m.ID)
	box.Children(func() {
		if m.Streaming {
			markdown.Stream(c, padCode(m.Text), a.mdOptions())
		} else {
			markdown.View(c, padCode(m.Text), a.mdOptions())
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

// subagentCard 是子 Agent 卡片(设计稿 .sub):标题行 + 统计行 + 当前动作。
func (a *App) subagentCard(c *ui.Context, s *core.SubagentCard) {
	t := a.tk
	run := s.Active()
	card := ui.ButtonBase(c).Key("sub-"+s.ID).Background(t.Raised).Radius(t.RMd()).Padding(7, 10).Margin(2, 0).
		Cursor(ui.CursorPointer).Transition(kit.Fade)
	for _, sh := range t.Shadow1() {
		card.Shadow(sh.X, sh.Y, sh.Blur, sh.Spread, sh.Color)
	}
	card.OnClick(func() { a.OpenWorkspaceTab(vSub, s.ID) })
	card.Children(func() {
		ui.Column(c).Gap(3).WidthPercent(100).Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				if run {
					kit.Spin(c, t.ToolAgent)
				} else {
					kit.Ic(c, "bot", kit.S12).TextColor(t.ToolAgent)
				}
				ui.Text(c, s.Description).FontWeight(600).SingleLine().Shrink(1)
				ui.Text(c, s.AgentType).TextColor(t.Tx3).SingleLine().Grow(1)
				if run {
					kit.Ib(c, "stop", kit.IbOpt{Small: true, Tip: "停止"}).OnClick(func() {
						if s.TaskID != "" {
							a.send(core.CmdStopTask{TaskID: s.TaskID})
							log.Info("停止子 Agent", "task", s.TaskID)
						}
					})
				} else {
					kit.Ic(c, "check", kit.S12).TextColor(t.Ok)
				}
			})
			// 统计行:用时 · 工具数 · token(设计稿 .sub 的第二行);疑似停滞时追加提示
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
			if s.Stalled(a.st.Now()) {
				meta = append(meta, "疑似停滞")
			}
			if len(meta) > 0 {
				mt := ui.Text(c, strings.Join(meta, "  ")).PaddingStart(22).FontSize(a.FlowCapFont()).TextColor(t.Tx3).FontFeatures("tnum")
				if a.chatTier() != tierNarrow { // 窄档允许折行(设计稿 .sub .stat flex-wrap)
					mt.SingleLine()
				}
			}
			if s.Summary != "" {
				ui.Text(c, s.Summary).PaddingStart(22).FontSize(a.FlowCapFont()).TextColor(t.Tx3)
			}
			a.subagentTimeline(c, s.Children)
		})
	})
}

// padCode 给行内代码两侧各补一个窄空格,让底色比文字宽出一点(设计稿的行内代码有 1×5 px 内边距,
// Markdown 组件没有这个选项)。围栏代码块里的反引号不处理;窄空格不会被 CommonMark 当作可裁掉的空白。
func padCode(src string) string {
	const thin = "\u2009"
	var b strings.Builder
	fenced := false
	for _, line := range strings.SplitAfter(src, "\n") {
		if strings.HasPrefix(strings.TrimLeft(line, " "), "```") {
			fenced = !fenced
			b.WriteString(line)
			continue
		}
		if fenced || !strings.Contains(line, "`") {
			b.WriteString(line)
			continue
		}
		in := false
		for _, r := range line {
			if r == '`' {
				if !in {
					b.WriteString("`" + thin)
				} else {
					b.WriteString(thin + "`")
				}
				in = !in
				continue
			}
			b.WriteRune(r)
		}
	}
	return b.String()
}
