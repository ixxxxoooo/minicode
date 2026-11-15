package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

// 计划:流式生成时自动打开计划标签;选中文字批注 → 继续规划。

func (a *App) onPlanUpdated(id string, streaming bool) {
	if !streaming || a.planOpened[id] {
		return
	}
	a.planOpened[id] = true
	a.OpenWorkspaceTab(vPlan, id)
	log.Info("流式计划:自动打开标签", "id", id)
}

func (a *App) planSelectionBar(c *ui.Context, planID, sel string) {
	if strings.TrimSpace(sel) == "" {
		return
	}
	t := a.tk
	ui.Row(c).Gap(6).Padding(4, 0).AlignItems(ui.Center).Children(func() {
		ui.Text(c, "选中 "+itoa(runeLen(sel))+" 字").FontSize(11).TextColor(t.Tx3).Grow(1).SingleLine()
		kit.Btn(c, "批注并继续规划", kit.BtnOpt{Small: true, Variant: kit.BtnSoft}).OnClick(func() {
			a.send(core.CmdResolvePlan{ID: planID, Approve: false})
			a.draft = "关于计划这段:「" + trunc(sel, 120) + "」\n"
			a.toast = "已填入输入框,可补充后发送"
		})
	})
}

func runeLen(s string) int { return len([]rune(s)) }
func trunc(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
