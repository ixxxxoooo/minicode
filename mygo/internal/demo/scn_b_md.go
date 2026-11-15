package demo

import "github.com/ixxxxoooo/minicode/internal/core"

func init() {
	Register(Scenario{
		ID:    "md",
		Title: "正文 · Markdown 全元素",
		Build: buildMd,
		Label: "昨天",
		Group: "昨天",
		Clock: "18:20",
	})
}

func buildMd() []Step {
	s := NewScript("md", "正文 · Markdown 全元素")
	uid := s.User("写一份发布说明,把 Markdown 能展示的结构都演示一遍。")
	s.Say(mdFullBody())
	cost := 0.038
	stats := &core.TurnStats{
		Model: "claude-sonnet-5-5", Input: 5200, Output: 4800,
		CacheRead: 88_000, CacheWrite: 1200,
		CostUSD: f64(cost), ApiMs: u64(12_400), TtftMs: u64(720), Calls: calls(4),
	}
	s.End(uid, 48, core.StopEndTurn, stats)
	s.Add(0, core.EvUsage{Usage: usagePct(22, 3.8)})
	return s.Steps()
}
