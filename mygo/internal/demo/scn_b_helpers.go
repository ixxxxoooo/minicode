package demo

import (
	"fmt"
	"strings"

	"github.com/ixxxxoooo/minicode/internal/core"
)

func f64(v float64) *float64 { return &v }
func u64(v uint64) *uint64   { return &v }
func calls(n int) *int       { return &n }

// statsNoCost 第三方服务商:无费用字段。
func statsNoCost() *core.TurnStats {
	s := DefaultStats(7)
	s.CostUSD = nil
	s.Model = "openrouter/custom-model"
	return s
}

// statsNoCache 无缓存命中数据。
func statsNoCache() *core.TurnStats {
	s := DefaultStats(3)
	s.CacheRead, s.CacheWrite = 0, 0
	return s
}

// statsHuge 超大 token 与费用(演示大数字排版)。
func statsHuge() *core.TurnStats {
	cost := 124.56
	return &core.TurnStats{
		Model:      "claude-opus-4-6-20251115",
		Input:      8_900_000,
		Output:     2_100_000,
		Thinking:   640_000,
		CacheRead:  120_000_000,
		CacheWrite: 480_000,
		CostUSD:    f64(cost),
		ApiMs:      u64(4_500_000),
		TtftMs:     u64(890),
		Calls:      calls(88),
	}
}

// statsTiny 极短回合:几乎无 token。
func statsTiny() *core.TurnStats {
	return &core.TurnStats{Model: "claude-haiku-4-5", Input: 18, Output: 1, Calls: calls(1)}
}

func usagePct(pct float64, cost float64) core.Usage {
	used := uint64(float64(200_000) * pct / 100)
	return core.Usage{ContextUsed: used, ContextSize: 200_000, CostUSD: cost}
}

func longGoFence(lines int) string {
	var b strings.Builder
	b.WriteString("```go\npackage main\n\nimport \"fmt\"\n\n")
	for i := 1; i <= lines; i++ {
		fmt.Fprintf(&b, "func step%d() { fmt.Println(%d) }\n", i, i)
	}
	b.WriteString("```\n")
	return b.String()
}
