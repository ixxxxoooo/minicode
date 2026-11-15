package demo

import (
	"github.com/ixxxxoooo/minicode/internal/core"
)

func init() {
	Register(Scenario{ID: "c-ask-single", Title: "待处理 · 提问单选单题", Label: "刚刚", Build: buildAskSingle})
	Register(Scenario{ID: "c-ask-multi", Title: "待处理 · 提问多选", Label: "刚刚", Build: buildAskMulti})
	Register(Scenario{ID: "c-ask-wizard", Title: "待处理 · 提问两三题向导", Label: "刚刚", Build: buildAskWizard})
	Register(Scenario{ID: "c-ask-longopt", Title: "待处理 · 提问长描述选项", Label: "刚刚", Build: buildAskLongOpt})
	Register(Scenario{ID: "c-ask-answered", Title: "待处理 · 提问已回答", Label: "刚刚", Build: buildAskAnswered})
	Register(Scenario{ID: "c-ask-pending", Title: "待处理 · 提问待回答", Label: "刚刚", Build: buildAskPending})
	Register(Scenario{ID: "c-ask-other", Title: "待处理 · 提问选了其他", Label: "刚刚", Build: buildAskOther})
}

func buildAskSingle() []Step {
	s := NewScript("c-ask-single", "待办 · 提问单选单题")
	uid := s.User("单选题")
	qid := s.ID("q")
	s.Add(ms(120), core.EvAppend{Item: &core.QuestionRequest{ID: qid, Questions: []core.Question{{
		Header: "策略", Question: "默认要不要忽略末尾斜杠?", Options: []core.QuestionOption{
			{Label: "忽略(推荐)", Description: "/users 与 /users/ 等价"},
			{Label: "严格区分", Description: "保留两个不同的路由"},
		},
	}}}})
	finishLab(s, uid, 1)
	return s.Steps()
}

func buildAskMulti() []Step {
	s := NewScript("c-ask-multi", "待办 · 提问多选")
	uid := s.User("多选题")
	qid := s.ID("q")
	s.Add(ms(120), core.EvAppend{Item: &core.QuestionRequest{ID: qid, Questions: []core.Question{{
		Header: "范围", Question: "这次改动要覆盖哪些模块?", MultiSelect: true,
		Options: []core.QuestionOption{
			{Label: "router", Description: "路径归一化"},
			{Label: "handlers", Description: "错误映射"},
			{Label: "middleware", Description: "日志与鉴权"},
		},
	}}}})
	finishLab(s, uid, 2)
	return s.Steps()
}

func buildAskWizard() []Step {
	s := NewScript("c-ask-wizard", "待办 · 提问两三题向导")
	uid := s.User("向导")
	qid := s.ID("q")
	s.Add(ms(120), core.EvAppend{Item: &core.QuestionRequest{ID: qid, Questions: Questions()}})
	finishLab(s, uid, 3)
	return s.Steps()
}

func buildAskLongOpt() []Step {
	s := NewScript("c-ask-longopt", "待办 · 提问长描述选项")
	uid := s.User("长描述")
	qid := s.ID("q")
	s.Add(ms(120), core.EvAppend{Item: &core.QuestionRequest{ID: qid, Questions: []core.Question{{
		Question: "选用哪种观测方案?",
		Options: []core.QuestionOption{
			{Label: "OpenTelemetry", Description: "需要 sidecar,可导出到 Grafana / Tempo,对现有 tracing 侵入较小,但本地开发要起 collector。"},
			{Label: "仅结构化日志", Description: "用 tracing subscriber 输出 JSON,便宜但缺乏分布式链路,适合先上线的 MVP。"},
		},
	}}}})
	finishLab(s, uid, 4)
	return s.Steps()
}

func buildAskAnswered() []Step {
	s := NewScript("c-ask-answered", "待办 · 提问已回答")
	uid := s.User("已回答态")
	qid := s.ID("q")
	s.Add(ms(120), core.EvAppend{Item: &core.QuestionRequest{
		ID: qid, Questions: Questions(),
		Answered: true, Answers: [][]string{{"忽略末尾斜杠"}, {"加一个开关", "写进文档"}},
	}})
	s.Say("已记录你的选择。")
	finishLab(s, uid, 5)
	return s.Steps()
}

func buildAskPending() []Step {
	s := NewScript("c-ask-pending", "待办 · 提问待回答")
	uid := s.User("待回答")
	qid := s.ID("q")
	s.Add(ms(120), core.EvAppend{Item: &core.QuestionRequest{ID: qid, Questions: Questions()}})
	return append(s.Steps(), Step{0, core.EvDuration{ID: uid, Secs: 1}})
}

func buildAskOther() []Step {
	s := NewScript("c-ask-other", "待办 · 提问选了其他")
	uid := s.User("其他")
	qid := s.ID("q")
	s.Add(ms(120), core.EvAppend{Item: &core.QuestionRequest{
		ID: qid, Questions: []core.Question{{
			Question: "部署目标?", Options: []core.QuestionOption{
				{Label: "Fly.io", Description: "就近边缘"},
				{Label: "AWS ECS", Description: "已有 VPC"},
				{Label: "其他", Description: "在下面说明"},
			},
		}},
		Answered: true, Answers: [][]string{{"其他"}},
	}})
	s.Say("收到,按自定义目标准备配置。")
	finishLab(s, uid, 6)
	return s.Steps()
}
