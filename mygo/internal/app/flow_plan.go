package app

// 计划:流式生成时自动打开计划标签。

func (a *App) onPlanUpdated(id string, streaming bool) {
	if !streaming || a.planOpened[id] {
		return
	}
	a.planOpened[id] = true
	a.OpenWorkspaceTab(vPlan, id)
	log.Info("流式计划:自动打开标签", "id", id)
}

func trunc(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
