package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

// planBody 计划全文:Markdown 纯预览(标题 / 加粗 / 列表 / 代码按渲染显示,不可编辑)。
// 生成中整篇流式渲染;完成后按块(空行分段,围栏代码块不拆)渲染,每块旁有「批注」按钮
// (MyGo 无选区 API 时的设计稿方案)。
func (a *App) planBody(c *ui.Context, planID, md string, streaming bool) {
	if strings.TrimSpace(md) == "" {
		if streaming {
			ui.Text(c, "生成中…").TextColor(a.tk.Tx3)
		}
		return
	}
	if streaming {
		markdown.Stream(c, padCode(md), a.mdOptions())
		return
	}
	for i, blk := range splitPlanBlocks(md) {
		a.planPara(c, planID, i, blk)
	}
}

// splitPlanBlocks 按空行把 Markdown 切成块;围栏代码块(```)内部的空行不切。
func splitPlanBlocks(md string) []string {
	var out []string
	var cur []string
	fenced := false
	flush := func() {
		if len(cur) > 0 {
			out = append(out, strings.Join(cur, "\n"))
			cur = nil
		}
	}
	for _, ln := range strings.Split(strings.TrimSpace(md), "\n") {
		if strings.HasPrefix(strings.TrimLeft(ln, " "), "```") {
			fenced = !fenced
		}
		if !fenced && strings.TrimSpace(ln) == "" {
			flush()
			continue
		}
		cur = append(cur, ln)
	}
	flush()
	return out
}

func (a *App) planPara(c *ui.Context, planID string, ix int, text string) {
	row := ui.Row(c).Key("plan-p-"+itoa(ix)).Gap(8).AlignItems(ui.Start).Margin(0, 0, 4, 0)
	hov := row.Hovered() // 批注按钮只在悬停到这一段时出现,平时不占眼
	row.Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Children(func() {
			markdown.View(c, padCode(text), a.mdOptions())
		})
		btn := kit.Ib(c, "chat", kit.IbOpt{Small: true, Tip: "批注并继续规划"}).OnClick(func() {
			a.annotatePlan(planID, text)
		})
		if !hov {
			btn.Opacity(0)
		}
	})
}

func (a *App) annotatePlan(planID, para string) {
	para = strings.TrimSpace(para)
	if para == "" {
		return
	}
	log.Info("计划批注", "plan", planID, "len", len(para))
	a.send(core.CmdResolvePlan{ID: planID, Approve: false})
	a.draft = "关于计划这段:「" + trunc(para, 200) + "」\n"
	a.toast = "已填入输入框,可补充后发送"
}
