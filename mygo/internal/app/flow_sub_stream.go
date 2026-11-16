package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

func (a *App) subFollowBottom() bool {
	return !a.subNoFollow
}

func (a *App) subScrollBtn(c *ui.Context, items []core.Item) {
	if a.subFollowBottom() || a.subSt.MaxY-a.subSt.Y < 32 {
		return
	}
	fresh := flowTailMark(items) != a.subAwayLen
	a.toLatestBtn(c, "to-latest-sub", fresh, 10, func() {
		a.subNoFollow = false
		a.subAwayLen = flowTailMark(items)
		a.subSt.Y = 1e9
	})
}

func (a *App) subStatusTag(c *ui.Context, s *core.SubagentCard) {
	switch {
	case s.Active():
		kit.Tag(c, "运行中", kit.KInfo)
	case s.Status == core.SubFailed:
		kit.Tag(c, "失败", kit.KErr)
	case s.Status == core.SubStopped:
		kit.Tag(c, "已停止")
	case s.Background:
		kit.Tag(c, "后台", kit.KInfo)
	default:
		kit.Tag(c, "已完成", kit.KOk)
	}
}

// subTaskMsg 首条「任务」:复用用户消息卡样式。
func (a *App) subTaskMsg(c *ui.Context, s *core.SubagentCard) {
	text := strings.TrimSpace(s.Prompt)
	if text == "" {
		text = s.Description
	}
	u := &core.User{ID: "subtask:" + s.ID, Text: text}
	a.userMsg(c, u, -1)
}

func (a *App) subFlowItems(c *ui.Context, s *core.SubagentCard) {
	items := s.Children
	if len(items) == 0 && !s.Active() && strings.TrimSpace(s.Summary) == "" {
		t := a.tk
		ui.Column(c).Padding(24, 0).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "子 Agent 还没有产出").TextColor(t.Tx3)
		})
		return
	}
	a.flowCol(c, func() {
		a.subTaskMsg(c, s)
		for i := 0; i < len(items); {
			if isStepItem(items[i]) {
				j := i + 1
				for j < len(items) && isStepItem(items[j]) {
					j++
				}
				a.stepGroup(c, items[i:j])
				i = j
				continue
			}
			if flowTailVisible(items[i]) {
				a.tailItem(c, items[i])
			}
			i++
		}
		if summary := strings.TrimSpace(s.Summary); summary != "" && !s.Active() {
			a.subResultCard(c, summary)
		}
	})
}

func (a *App) subResultCard(c *ui.Context, summary string) {
	t := a.tk
	ui.Column(c).Gap(6).Margin(8, 0, 0, 0).Children(func() {
		ui.Text(c, "结果").FontSize(theme.FsCap).FontWeight(600).TextColor(t.Tx3)
		ui.Column(c).Padding(10, 12).Background(t.Raised).Radius(t.RMd()).Border(1, t.Bd1).Children(func() {
			ui.Text(c, summary).LineHeight(1.55)
		})
	})
}

func (a *App) subExportText(s *core.SubagentCard) string {
	var b strings.Builder
	if p := strings.TrimSpace(s.Prompt); p != "" {
		b.WriteString("## 任务\n")
		b.WriteString(p)
		b.WriteString("\n\n")
	}
	for _, it := range s.Children {
		switch v := it.(type) {
		case *core.Assistant:
			b.WriteString(v.Text)
			b.WriteString("\n\n")
		case *core.ToolCall:
			b.WriteString("- ")
			b.WriteString(v.Title)
			b.WriteString("\n")
		}
	}
	if sum := strings.TrimSpace(s.Summary); sum != "" {
		b.WriteString("\n## 结果\n")
		b.WriteString(sum)
	}
	return strings.TrimSpace(b.String())
}
