package demo

import (
	"github.com/ixxxxoooo/minicode/internal/core"
)

// RunningFixBug 回放修复路由回合但不结束(PhaseRunning),供排队 / 引导截图。
func RunningFixBug() []Step {
	steps := FixBug(false)
	for i, s := range steps {
		if _, ok := s.Ev.(core.EvTurnEnded); ok {
			return steps[:i]
		}
	}
	return steps
}

// Questions3 三题向导:单选 + 多选 + 无选项(自由输入)。
func Questions3() []core.Question {
	return []core.Question{
		{Header: "斜杠", Question: "末尾斜杠要怎么处理?", Options: []core.QuestionOption{
			{Label: "忽略末尾斜杠", Description: "/users/ 与 /users 等价(推荐)"},
			{Label: "严格区分", Description: "保持现状"},
		}},
		{Header: "范围", Question: "这次改动要覆盖哪些模块?", MultiSelect: true, Options: []core.QuestionOption{
			{Label: "router", Description: "路径归一化"},
			{Label: "handlers", Description: "错误映射"},
			{Label: "middleware", Description: "日志与鉴权"},
		}},
		{Header: "补充", Question: "还有需要说明的约束吗?", Options: nil},
	}
}

func pendingQuestionBase(q []core.Question) []Step {
	return pendingBase(&core.QuestionRequest{ID: "q1", Questions: q})
}

// PendingQuestion3 停在三题待回答向导上(截图 ask2)。
func PendingQuestion3() []Step {
	return pendingQuestionBase(Questions3())
}

// AskDoneReplay 在会话流里展示三种已回答卡片:全答 / 含跳过 / 含自由输入。
func AskDoneReplay() []Step {
	base := []Step{
		{0, core.EvReady{SessionID: "demo-askdone", Models: core.DefaultModels(), Model: "sonnet", Mode: core.ModeDefault}},
		{0, core.EvTitleChanged{Title: "演示 · 问答已回答"}},
		{0, core.EvAppend{Item: &core.User{ID: "u0", Text: "演示三种回答卡片"}}},
		{0, core.EvTurnStarted{}},
	}
	add := func(ev core.Event) { base = append(base, Step{0, ev}) }

	add(core.EvAppend{Item: &core.QuestionRequest{
		ID: "qa-all", Questions: Questions(),
		Answered: true, Answers: [][]string{{"忽略末尾斜杠"}, {"加一个开关", "写进文档"}},
	}})
	add(core.EvAppend{Item: &core.Assistant{ID: "a1", Text: "第一种:全部作答。"}})

	add(core.EvAppend{Item: &core.QuestionRequest{
		ID: "qa-skip", Questions: Questions(),
		Answered: true, Answers: [][]string{{"严格区分"}, {}},
	}})
	add(core.EvAppend{Item: &core.Assistant{ID: "a2", Text: "第二种:第二题跳过。"}})

	add(core.EvAppend{Item: &core.QuestionRequest{
		ID: "qa-free", Questions: []core.Question{{
			Question: "部署目标?", Options: []core.QuestionOption{
				{Label: "Fly.io", Description: "就近边缘"},
				{Label: "其他", Description: "在下面说明"},
			},
		}},
		Answered: true, Answers: [][]string{{"自有 K8s 集群(华北)"}},
	}})
	add(core.EvAppend{Item: &core.Assistant{ID: "a3", Text: "第三种:自由输入 / 其它说明。"}})

	add(core.EvTurnEnded{Reason: core.StopEndTurn})
	return base
}
