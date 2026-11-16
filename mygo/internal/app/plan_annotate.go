package app

import (
	"github.com/ixxxxoooo/minicode/internal/theme"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

// planBody 计划全文:按段落渲染,每段旁有「批注」按钮(MyGo 无选区 API 时的设计稿方案)。
func (a *App) planBody(c *ui.Context, planID, md string, streaming bool) {
	paras := splitPlanParas(md)
	if len(paras) == 0 {
		if streaming && strings.TrimSpace(md) == "" {
			ui.Text(c, "生成中…").TextColor(a.tk.Tx3)
		}
		return
	}
	for i, p := range paras {
		if strings.TrimSpace(p) == "" {
			continue
		}
		a.planPara(c, planID, i, p, streaming)
	}
}

func splitPlanParas(md string) []string {
	md = strings.TrimSpace(md)
	if md == "" {
		return nil
	}
	return strings.Split(md, "\n")
}

func (a *App) planPara(c *ui.Context, planID string, ix int, text string, streaming bool) {
	t := a.tk
	ui.Row(c).Key("plan-p-"+itoa(ix)).Gap(8).AlignItems(ui.Start).Margin(0, 0, 6, 0).Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Children(func() {
			ui.Text(c, text).FontSize(theme.Px(13)).LineHeight(1.55).TextColor(t.Tx1)
		})
		if !streaming {
			kit.Ib(c, "chat", kit.IbOpt{Small: true, Tip: "批注并继续规划"}).OnClick(func() {
				a.annotatePlan(planID, text)
			})
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
