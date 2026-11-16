package demo

import (
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

func init() {
	Register(Scenario{
		ID: "resp-wait-slow", Title: "演示 · 慢响应状态行", Project: "status-lab",
		Build: buildRespWaitSlow,
	})
	Register(Scenario{
		ID: "resp-multi-timing", Title: "演示 · 多次调用耗时", Project: "status-lab",
		Build: buildRespMultiTiming,
	})
	Register(Scenario{
		ID: "compact-running", Title: "演示 · 压缩中", Project: "status-lab",
		Build: buildCompactRunning,
	})
	Register(Scenario{
		ID: "compact-done", Title: "演示 · 压缩完成", Project: "status-lab",
		Build: buildCompactDone,
	})
	Register(Scenario{
		ID: "compact-fail", Title: "演示 · 压缩失败", Project: "status-lab",
		Build: buildCompactFail,
	})
}

// FixedClock 为 true(cmd/shot 截图)时,演示阶段时间锚在设计稿的「今天」,与注入的时钟对齐;
// 否则锚在真实当前时间,开发版里等待行的已用时才是几秒而不是几十小时。
var FixedClock bool

func shotBase() time.Time {
	if FixedClock {
		return time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	}
	return time.Now()
}

func buildRespWaitSlow() []Step {
	t0 := shotBase().Add(-12 * time.Second)
	tw := shotBase().Add(-11 * time.Second)
	uid := "iu-slow"
	return []Step{
		{0, core.EvAppend{Item: &core.User{ID: uid, Text: "故意放慢,看等待行"}}},
		{0, core.EvTurnStarted{}},
		{0, core.EvUsage{Usage: core.Usage{ContextUsed: 52_000}}},
		{0, core.EvTurnPhase{Kind: core.StageConn, At: t0, TurnStart: t0}},
		{0, core.EvTurnPhase{Kind: core.StagePrep, At: t0.Add(600 * time.Millisecond), TurnStart: t0}},
		{0, core.EvTurnPhase{Kind: core.StageWait, Call: 1, At: tw, TurnStart: t0}},
	}
}

func buildRespMultiTiming() []Step {
	uid := "iu-mc"
	steps := []Step{{0, core.EvAppend{Item: &core.User{ID: uid, Text: "多次"}}}}
	steps = append(steps, multiCallSteps(uid)...)
	return steps
}

func multiCallSteps(uid string) []Step {
	b := &Backend{}
	return b.multiCallScene(uid)
}

func buildCompactRunning() []Step {
	t0 := shotBase().Add(-3*time.Second - 200*time.Millisecond)
	uid := "iu-cpr"
	return []Step{
		{0, core.EvAppend{Item: &core.User{ID: uid, Text: "/compact"}}},
		{0, core.EvTurnStarted{}},
		{0, core.EvUsage{Usage: core.Usage{ContextUsed: 25_500}}},
		{0, core.EvCompact{State: core.CompactRunning}},
		{0, core.EvTurnPhase{Kind: core.StageWait, Call: 1, At: t0, TurnStart: t0}},
	}
}

func buildCompactDone() []Step {
	uid := "iu-cpd"
	return []Step{
		{0, core.EvAppend{Item: &core.User{ID: uid, Text: "/compact"}}},
		{0, core.EvTurnStarted{}},
		{0, core.EvCompact{State: core.CompactDone, Pre: 25_500, Post: 5_600, Ms: 17_600}},
		{0, core.EvTurnEnded{Reason: core.StopEndTurn}},
	}
}

func buildCompactFail() []Step {
	b := &Backend{}
	uid := "iu-cpf"
	return append([]Step{
		{0, core.EvAppend{Item: &core.User{ID: uid, Text: "压缩失败"}}},
		{0, core.EvTurnStarted{}},
	}, b.compactFailScene()...)
}
