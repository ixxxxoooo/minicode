package demo

import (
	"github.com/ixxxxoooo/minicode/internal/core"
)

func init() {
	Register(Scenario{ID: "c-plan-stream", Title: "计划 · 流式生成中", Label: "刚刚", Live: true, Build: buildPlanStreaming})
	Register(Scenario{ID: "c-plan-pending", Title: "计划 · 待批准", Label: "刚刚", Build: buildPlanPending})
	Register(Scenario{ID: "c-plan-approved", Title: "计划 · 已批准", Label: "刚刚", Build: buildPlanApproved})
	Register(Scenario{ID: "c-plan-rejected", Title: "计划 · 已拒绝继续规划", Label: "刚刚", Build: buildPlanRejected})
	Register(Scenario{ID: "c-plan-long", Title: "计划 · 很长含代码块", Label: "刚刚", Build: buildPlanLong})
	Register(Scenario{ID: "c-plan-todos", Title: "计划 · 与待办联动", Label: "刚刚", Build: buildPlanTodos})
}

func buildPlanStreaming() []Step {
	s := NewScript("c-plan-stream", "计划 · 流式生成中")
	uid := s.User("流式生成计划")
	pid := s.ID("p")
	s.Add(ms(150), core.EvAppend{Item: &core.PlanApproval{ID: pid, Streaming: true}})
	acc := ""
	for _, chunk := range chunks(PlanMarkdown, 8) {
		acc += chunk
		s.Add(ms(70), core.EvPlanUpdated{ID: pid, Markdown: acc, Streaming: true})
	}
	// 不收尾 Streaming=false,保持生成中外观
	s.Add(ms(500), core.EvTurnStats{ID: uid, Stats: *DefaultStats(7)})
	return s.Steps()
}

func buildPlanPending() []Step {
	s := NewScript("c-plan-pending", "计划 · 待批准")
	uid := s.User("待批准")
	pid := s.ID("p")
	s.Add(ms(120), core.EvAppend{Item: &core.PlanApproval{ID: pid, Markdown: PlanMarkdown}})
	return append(s.Steps(), Step{0, core.EvDuration{ID: uid, Secs: 1}})
}

func buildPlanApproved() []Step {
	s := NewScript("c-plan-approved", "计划 · 已批准")
	uid := s.User("已批准")
	pid := s.ID("p")
	tr := true
	s.Add(ms(120), core.EvAppend{Item: &core.PlanApproval{ID: pid, Markdown: PlanMarkdown, Approved: &tr}})
	s.Add(0, core.EvModeChanged{Mode: core.ModeAcceptEdits})
	s.Say("计划已批准,进入实施。")
	finishLab(s, uid, 8)
	return s.Steps()
}

func buildPlanRejected() []Step {
	s := NewScript("c-plan-rejected", "计划 · 已拒绝继续规划")
	uid := s.User("继续规划")
	pid := s.ID("p")
	fl := false
	s.Add(ms(120), core.EvAppend{Item: &core.PlanApproval{ID: pid, Markdown: PlanMarkdown, Approved: &fl}})
	s.Say("好的,我会把方案再细化一轮。")
	finishLab(s, uid, 9)
	return s.Steps()
}

func buildPlanLong() []Step {
	s := NewScript("c-plan-long", "计划 · 很长含代码块")
	uid := s.User("长计划")
	pid := s.ID("p")
	s.Add(ms(120), core.EvAppend{Item: &core.PlanApproval{ID: pid, Markdown: longPlanMD()}})
	finishLab(s, uid, 10)
	return s.Steps()
}

func buildPlanTodos() []Step {
	s := NewScript("c-plan-todos", "计划 · 与待办联动")
	uid := s.User("计划+待办")
	pid := s.ID("p")
	s.Add(ms(120), core.EvAppend{Item: &core.PlanApproval{ID: pid, Markdown: PlanMarkdown}})
	s.Add(ms(200), core.EvTodos{ID: s.ID("td"), Items: []core.TodoItem{
		{Content: "抽出 normalize", Status: core.TodoInProgress, ActiveForm: "实现 normalize"},
		{Content: "补测试", Status: core.TodoPending},
		{Content: "更新文档", Status: core.TodoPending},
	}})
	s.Say("计划待批准,待办已同步。")
	finishLab(s, uid, 11)
	return s.Steps()
}
