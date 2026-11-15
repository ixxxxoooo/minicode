package demo

import (
	"fmt"
	"strings"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

func buildPressHugeReply() []Step {
	b := &stepBuf{}
	b.ev(core.EvReady{SessionID: "demo-press-huge-reply", Models: core.DefaultModels(), Model: "sonnet", Mode: core.ModeDefault})
	b.ev(core.EvTitleChanged{Title: "压力 · 超长回复"})
	u1 := "press-u1"
	b.ev(core.EvAppend{Item: &core.User{ID: u1, Text: "生成超长说明文档"}})
	b.ev(core.EvTurnStarted{})
	b.say("press-a1", genMarkdownBulk(50000, "手册"))
	b.endTurn(u1, 1, core.StopEndTurn)
	u2 := "press-u2"
	b.ev(core.EvAppend{Item: &core.User{ID: u2, Text: "再来一条无换行超长文本"}})
	b.ev(core.EvTurnStarted{})
	b.say("press-a2", genRunOn(200000))
	b.endTurn(u2, 2, core.StopEndTurn)
	return b.out()
}

func buildPressCmdOutput() []Step {
	b := &stepBuf{}
	b.ev(core.EvReady{SessionID: "demo-press-cmd-out", Models: core.DefaultModels(), Model: "sonnet", Mode: core.ModeDefault})
	b.ev(core.EvTitleChanged{Title: "压力 · 超长命令输出"})
	u := "press-u-cmd"
	b.ev(core.EvAppend{Item: &core.User{ID: u, Text: "跑一次全量集成测试并贴完整日志"}})
	b.ev(core.EvTurnStarted{})
	tc := core.NewToolCall("press-bash", "Bash", "integration test").WithInput(map[string]any{"command": "pytest -vv"})
	out := genANSILines(20000)
	b.toolDone(tc, 120000, &core.TerminalBlock{Command: "pytest -vv", Output: out, ExitCode: core.Int(0)})
	b.say("press-a-cmd", "集成测试完成，日志已附上。")
	b.endTurn(u, 3, core.StopEndTurn)
	return b.out()
}

func buildPressHugeDiff() []Step {
	b := &stepBuf{}
	b.ev(core.EvReady{SessionID: "demo-press-diff", Models: core.DefaultModels(), Model: "sonnet", Mode: core.ModeDefault})
	b.ev(core.EvTitleChanged{Title: "压力 · 超长 diff"})
	u := "press-u-diff"
	path := "src/generated/bulk_refactor.rs"
	b.ev(core.EvAppend{Item: &core.User{ID: u, Text: "批量重构并汇总改动"}})
	b.ev(core.EvTurnStarted{})
	old, neu := genDiffBody(3000, path)
	tc := core.NewToolCall("press-edit", "Edit", "编辑 "+path).WithInput(map[string]any{"file_path": path}).WithLocation(path)
	b.toolDone(tc, 5000, &core.DiffBlock{Path: path, Old: old, New: neu, Line: 1})
	files := make([]core.ChangedFile, 60)
	for i := range files {
		files[i] = core.ChangedFile{
			Path:    fmt.Sprintf("src/pkg%d/f%d.rs", i/6, i),
			Added:   40 + i%17,
			Removed: i % 9,
		}
	}
	b.ev(core.EvAppend{Item: &core.Changes{ID: "press-ch", Files: files}})
	b.say("press-a-diff", "单文件 3000 行 diff 与 60 文件改动汇总已生成。")
	b.endTurn(u, 4, core.StopEndTurn)
	return b.out()
}

func buildPressMassTools() []Step {
	const n = 800
	b := &stepBuf{}
	b.cap(2*n + 12)
	b.ev(core.EvReady{SessionID: "demo-press-tools", Models: core.DefaultModels(), Model: "sonnet", Mode: core.ModeDefault})
	b.ev(core.EvTitleChanged{Title: "压力 · 海量工具调用"})
	u := "press-u-tools"
	b.ev(core.EvAppend{Item: &core.User{ID: u, Text: "探索整个仓库结构"}})
	b.ev(core.EvTurnStarted{})
	names := []string{"Read", "Grep", "Edit"}
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("press-k-%d", i)
		name := names[i%3]
		path := fmt.Sprintf("src/tree/%d_%d.go", i/20, i%20)
		tc := core.NewToolCall(id, name, name+" "+path).WithInput(map[string]any{"file_path": path}).WithLocation(path)
		var c core.ToolContent = &core.TextBlock{Text: fmt.Sprintf("hit %d\n", i)}
		if name == "Edit" {
			o, ne := genDiffBody(3, path)
			c = &core.DiffBlock{Path: path, Old: o, New: ne, Line: i + 1}
		}
		b.toolDone(tc, 50+int64(i%40), c)
	}
	b.say("press-a-tools", fmt.Sprintf("完成 %d 次读/搜/改。", n))
	b.endTurn(u, 5, core.StopEndTurn)
	return b.out()
}

func buildPressDeepSub() []Step {
	b := &stepBuf{}
	b.ev(core.EvReady{SessionID: "demo-press-sub", Models: core.DefaultModels(), Model: "sonnet", Mode: core.ModeDefault})
	b.ev(core.EvTitleChanged{Title: "压力 · 深层嵌套子 Agent"})
	u := "press-u-sub"
	b.ev(core.EvAppend{Item: &core.User{ID: u, Text: "并行三路深度审查"}})
	b.ev(core.EvTurnStarted{})
	for br := 0; br < 3; br++ {
		root := fmt.Sprintf("press-sub-%d", br)
		b.ev(core.EvAppend{Item: core.NewSubagent(root, "review", fmt.Sprintf("分支 %d", br+1), core.SubRunning)})
		parent := root
		for layer := 1; layer <= 6; layer++ {
			for t := 0; t < 20; t++ {
				tid := fmt.Sprintf("%s-L%d-T%d", root, layer, t)
				path := fmt.Sprintf("src/branch%d/layer%d/f%d.rs", br, layer, t)
				tc := core.NewToolCall(tid, "Read", "读取 "+path).WithInput(map[string]any{"file_path": path}).WithLocation(path)
				b.ev(core.EvAppend{Parent: parent, Item: tc})
				b.ev(core.EvToolUpdate{Parent: parent, ID: tid, Patch: core.ToolPatch{
					Status:        core.Status(core.ToolCompleted),
					AppendContent: []core.ToolContent{&core.TextBlock{Text: fmt.Sprintf("layer %d tool %d\n", layer, t)}},
				}})
			}
			if layer < 6 {
				child := fmt.Sprintf("%s-L%d", root, layer+1)
				b.ev(core.EvAppend{Parent: parent, Item: core.NewSubagent(child, "explore", fmt.Sprintf("第 %d 层", layer+1), core.SubRunning)})
				parent = child
			}
		}
		b.ev(core.EvSubagentFinished{ID: root, Status: core.SubCompleted, Summary: "分支完成", ToolCount: 120, Tokens: 24000})
	}
	b.say("press-a-sub", "三路 6 层嵌套子 Agent 已结束。")
	b.endTurn(u, 6, core.StopEndTurn)
	return b.out()
}

func buildPressStreamFlood() []Step {
	const (
		tick  = 15 * time.Millisecond
		total = 4000 // ≈60s
		chunk = 48
	)
	b := &stepBuf{}
	b.cap(total*4 + 16)
	b.ev(core.EvReady{SessionID: "demo-press-flood", Models: core.DefaultModels(), Model: "sonnet", Mode: core.ModeDefault})
	b.ev(core.EvTitleChanged{Title: "压力 · 流式洪水"})
	u := "press-u-flood"
	b.ev(core.EvAppend{Item: &core.User{ID: u, Text: "持续流式输出直到正常收尾"}})
	b.ev(core.EvTurnStarted{})
	thinkID, sayID, bashID := "press-think", "press-say", "press-bash"
	b.steps = append(b.steps, Step{tick, core.EvAppend{Item: &core.Thinking{ID: thinkID, Streaming: true}}})
	b.steps = append(b.steps, Step{tick, core.EvAppend{Item: &core.Assistant{ID: sayID, Streaming: true}}})
	bash := core.NewToolCall(bashID, "Bash", "build").WithInput(map[string]any{"command": "cargo build"})
	b.steps = append(b.steps, Step{tick, core.EvAppend{Item: bash}})
	phaseText := genRunOn(8000)
	var live strings.Builder
	live.Grow(total * chunk)
	for i := 0; i < total; i++ {
		off := (i * chunk) % len(phaseText)
		end := min(off+chunk, len(phaseText))
		delta := phaseText[off:end]
		live.WriteString(delta)
		tail := live.String()
		if len(tail) > 8192 {
			tail = tail[len(tail)-8192:]
		}
		liveStr := tail
		b.steps = append(b.steps,
			Step{tick, core.EvThinkingDelta{ID: thinkID, Delta: delta}},
			Step{0, core.EvTextDelta{ID: sayID, Delta: delta}},
			Step{0, core.EvToolUpdate{ID: bashID, Patch: core.ToolPatch{LiveOutput: core.Str(liveStr)}}},
		)
	}
	b.steps = append(b.steps,
		Step{tick, core.EvItemDone{ID: thinkID}},
		Step{0, core.EvItemDone{ID: sayID}},
		Step{0, core.EvToolUpdate{ID: bashID, Patch: core.ToolPatch{
			Status: core.Status(core.ToolCompleted), DurationMs: 60000,
			AppendContent: []core.ToolContent{&core.TerminalBlock{Command: "cargo build", Output: "finished\n", ExitCode: core.Int(0)}},
		}}},
		Step{0, core.EvTurnStats{ID: u, Stats: *DefaultStats(99)}},
		Step{0, core.EvDuration{ID: u, Secs: 62}},
		Step{0, core.EvTurnEnded{Reason: core.StopEndTurn}},
	)
	return b.out()
}

func buildPressHugeUser() []Step {
	b := &stepBuf{}
	b.ev(core.EvReady{SessionID: "demo-press-user", Models: core.DefaultModels(), Model: "sonnet", Mode: core.ModeDefault})
	b.ev(core.EvTitleChanged{Title: "压力 · 超长用户消息"})
	files := make([]string, 200)
	for i := range files {
		files[i] = fmt.Sprintf("pkg/service%d/handler.go", i)
	}
	u := "press-u-big"
	b.ev(core.EvAppend{Item: &core.User{ID: u, Text: genMarkdownBulk(30000, "需求说明"), Files: files}})
	b.ev(core.EvTurnStarted{})
	b.say("press-a-user", "收到，将按 200 个引用分批处理。")
	b.endTurn(u, 7, core.StopEndTurn)
	return b.out()
}

func init() {
	regs := []Scenario{
		{ID: "press-huge-reply", Title: "压力 · 超长回复", Build: buildPressHugeReply, Live: false},
		{ID: "press-cmd-output", Title: "压力 · 超长命令输出", Build: buildPressCmdOutput, Live: false},
		{ID: "press-huge-diff", Title: "压力 · 超长 diff", Build: buildPressHugeDiff, Live: false},
		{ID: "press-mass-tools", Title: "压力 · 海量工具调用", Build: buildPressMassTools, Live: false},
		{ID: "press-deep-sub", Title: "压力 · 深层嵌套子 Agent", Build: buildPressDeepSub, Live: false},
		{ID: "press-stream-flood", Title: "压力 · 流式洪水", Build: buildPressStreamFlood, Live: true},
		{ID: "press-huge-user", Title: "压力 · 超长用户消息", Build: buildPressHugeUser, Live: false},
	}
	for _, s := range regs {
		Register(s)
	}
}
