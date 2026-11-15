package demo

import "github.com/ixxxxoooo/minicode/internal/core"

func init() {
	Register(Scenario{
		ID:    "turn-context",
		Title: "回合 · 上下文水位",
		Build: buildTurnContext,
		Label: "刚刚",
	})
}

func buildTurnContext() []Step {
	s := NewScript("turn-context", "回合 · 上下文水位")

	type step struct {
		prompt string
		reply  string
		pct    float64
		cost   float64
		secs   uint64
		seed   int
	}
	steps := []step{
		{"上下文约 5%", "开始积累上下文。", 5, 0.4, 3, 1},
		{"上下文约 76%", "对话变长,圆环应逼近警告区。", 76, 2.8, 6, 2},
		{"上下文约 95%", "接近上限,应有告急样式。", 95, 8.6, 9, 3},
		{"上下文 100% 已满", "已满后应提示压缩或新会话。", 100, 12.0, 4, 4},
		{"费用统计很大", "费用数字排版检查。", 88, 248.9, 5, 5},
	}
	for _, st := range steps {
		uid := s.User(st.prompt)
		s.Say(st.reply)
		s.End(uid, st.secs, core.StopEndTurn, DefaultStats(st.seed))
		s.Add(0, core.EvUsage{Usage: usagePct(st.pct, st.cost)})
	}
	return s.Steps()
}
