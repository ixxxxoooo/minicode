package demo

import (
	"github.com/ixxxxoooo/minicode/internal/core"
)

// 演示:分阶段耗时(设计稿 resp-time 页)。时刻留空,由归约时按墙钟记。
// 触发词:默认回复(短等待)、「慢」(冷启动 + 11 秒等待模型 + 收尾 4 秒)、「多次」(两次模型调用 + 工具 + 缓存命中)。

func stage(k core.StageKind, call int) core.Event {
	return core.EvTurnPhase{Kind: k, Call: call}
}

func phase(k core.StageKind, msN int64, call int, label string) core.TimingPhase {
	return core.TimingPhase{Kind: k, Ms: msN, Call: call, Label: label}
}

func sumMs(p []core.TimingPhase) int64 {
	var n int64
	for _, x := range p {
		n += x.Ms
	}
	return n
}

func timingEv(uid string, startup int64, phases []core.TimingPhase, calls []core.CallInfo) core.Event {
	first := int64(0)
	for _, p := range phases {
		if p.Kind == core.StageWrite || p.Kind == core.StageThink {
			break
		}
		first += p.Ms
	}
	return core.EvTurnTiming{ID: uid, Timing: core.TurnTiming{StartupMs: startup, FirstMs: first, TotalMs: sumMs(phases), Phases: phases, Calls: calls}}
}

// cacheCalls 演示用缓存:第一次写入、第二次命中。
func cacheCalls(n int) []core.CallInfo {
	out := []core.CallInfo{{Cache: core.CacheTokens{Written: 26170, Uncached: 2}}}
	for i := 1; i < n; i++ {
		out = append(out, core.CallInfo{Cache: core.CacheTokens{Read: 26180, Written: 17, Uncached: 4}})
	}
	return out
}

// shortWaitScene 默认回复:准备 → 等待模型 1.4 秒 → 生成,脚注带分段条。
func (b *Backend) shortWaitScene(uid, text string) []Step {
	s := []Step{{0, stage(core.StagePrep, 0)}, {ms(120), stage(core.StageWait, 1)}, {ms(1400), stage(core.StageWrite, 1)}}
	s = append(s, b.replyScene(uid, text)...)
	// 回合结束前补上分段耗时
	n := len(s) - 1
	phases := []core.TimingPhase{phase(core.StagePrep, 120, 0, ""), phase(core.StageWait, 1400, 1, ""), phase(core.StageWrite, 900, 1, "")}
	s = append(s[:n:n], Step{0, timingEv(uid, 0, phases, cacheCalls(1)[:1])}, s[n])
	return s
}

// slowWaitScene 冷启动 0.6 秒 + 等待模型 11 秒(琥珀 + 慢因提示)+ 生成 + 收尾 4 秒(收尾行)。
func (b *Backend) slowWaitScene(uid string) []Step {
	s := []Step{{0, stage(core.StageConn, 0)}, {ms(600), stage(core.StagePrep, 0)}, {ms(100), stage(core.StageWait, 1)}, {ms(11000), stage(core.StageWrite, 1)}}
	aid := b.nextID("ia")
	s = append(s, Step{ms(60), core.EvAppend{Item: &core.Assistant{ID: aid, Streaming: true}}})
	for _, r := range chunks("这是一次故意放慢的演示:等待模型用了 11 秒,等待行会转琥珀并给出原因;最后一段出完后还有 4 秒才结束,会出现「收尾中」。", 6) {
		s = append(s, Step{ms(40), core.EvTextDelta{ID: aid, Delta: r}})
	}
	s = append(s, Step{ms(80), core.EvItemDone{ID: aid}}, Step{0, stage(core.StageFin, 1)})
	phases := []core.TimingPhase{
		phase(core.StageConn, 600, 0, ""), phase(core.StagePrep, 100, 0, ""), phase(core.StageWait, 11000, 1, ""),
		phase(core.StageWrite, 1400, 1, ""), phase(core.StageFin, 4000, 1, ""),
	}
	s = append(s, Step{ms(4000), core.EvDuration{ID: uid, Secs: 17}}, Step{0, timingEv(uid, 600, phases, cacheCalls(1))}, Step{0, core.EvTurnEnded{Reason: core.StopEndTurn}})
	return s
}

// multiCallScene 两次模型调用:等待 → 生成 → 工具 → 再次等待(缓存命中)→ 生成。
func (b *Backend) multiCallScene(uid string) []Step {
	tid := b.nextID("if")
	tc := core.NewToolCall(tid, "Bash", "运行 echo hi").WithInput(map[string]any{"command": "echo hi"})
	s := []Step{
		{0, stage(core.StagePrep, 0)}, {ms(100), stage(core.StageWait, 1)}, {ms(2100), stage(core.StageWrite, 1)},
		{ms(500), core.EvAppend{Item: tc}}, {0, stage(core.StageTool, 1)},
		{ms(800), core.EvToolUpdate{ID: tid, Patch: core.ToolPatch{Status: core.Status(core.ToolCompleted), DurationMs: 800,
			AppendContent: []core.ToolContent{&core.TerminalBlock{Command: "echo hi", Output: "hi", ExitCode: core.Int(0)}}}}},
		{0, stage(core.StageWait, 2)}, {ms(3900), stage(core.StageWrite, 2)},
	}
	phases := []core.TimingPhase{
		phase(core.StagePrep, 100, 0, ""), phase(core.StageWait, 2100, 1, ""), phase(core.StageWrite, 500, 1, ""),
		phase(core.StageTool, 800, 0, "Bash"), phase(core.StageWait, 3900, 2, ""), phase(core.StageWrite, 900, 2, ""),
	}
	r := b.replyScene(uid, "两次模型调用:第一次写入缓存,第二次命中缓存;点脚注的「用时」看瀑布。")
	n := len(r) - 1
	s = append(s, r[:n]...)
	return append(s, Step{0, timingEv(uid, 0, phases, cacheCalls(2))}, r[n])
}
