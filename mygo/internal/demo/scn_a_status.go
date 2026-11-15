package demo

import (
	"github.com/ixxxxoooo/minicode/internal/core"
)

func init() {
	Register(Scenario{
		ID: "tools-status", Title: "工具卡 · 状态矩阵", Project: "",
		Build: buildScnToolsStatus,
	})
	Register(Scenario{
		ID: "tools-think", Title: "工具卡 · 思考样式", Project: "",
		Build: buildScnToolsThink,
	})
}

func buildScnToolsStatus() []Step {
	s := NewScript("tools-status", "工具卡 · 状态矩阵")
	u := s.User("同一种 Bash / Read 工具,展示进行中、完成、失败、拒绝、中断、有耗时与无耗时。")

	base := func(id, title string) *core.ToolCall {
		return core.NewToolCall(id, "Bash", title).WithInput(map[string]any{"command": "echo " + id})
	}

	s.Add(ms(100), core.EvAppend{Item: base("run", "进行中")})
	s.Tool(base("ok", "已完成"), core.ToolCompleted, 850,
		&core.TerminalBlock{Command: "echo ok", Output: "ok\n", ExitCode: core.Int(0)})
	s.Tool(base("bad", "已失败"), core.ToolFailed, 1200,
		&core.TerminalBlock{Command: "echo bad", Output: "boom\n", ExitCode: core.Int(2)})

	deny := base("deny", "将被拒绝")
	s.Add(ms(100), core.EvAppend{Item: deny})
	s.Add(ms(200), core.EvPromoteToPermission{Request: &core.PermissionRequest{
		ID: "deny", Tool: deny, Options: []core.PermissionOption{
			{ID: "allow", Label: "允许一次", Kind: core.AllowOnce},
			{ID: "reject", Label: "拒绝", Kind: core.RejectOnce},
		},
	}})
	s.Add(ms(400), core.EvPermissionResolved{ID: "deny", Option: "reject"})

	s.Tool(core.NewToolCall("rd", "Read", "读取 src/lib.rs").
		WithInput(map[string]any{"file_path": "src/lib.rs"}).WithLocation("src/lib.rs"),
		core.ToolCompleted, 0, &core.TextBlock{Text: readSnippet(1, 6)})

	rdRun := core.NewToolCall("intr", "Read", "读取中将被中断").
		WithInput(map[string]any{"file_path": "src/router.rs"}).WithLocation("src/router.rs")
	s.Add(ms(100), core.EvAppend{Item: rdRun})
	s.Say("状态矩阵:进行中(run)、完成、失败、拒绝(deny)、无耗时与取消后的中断。")
	s.End(u, 8, core.StopCancelled, DefaultStats(1))
	return s.Steps()
}

func buildScnToolsThink() []Step {
	s := NewScript("tools-think", "工具卡 · 思考样式")
	u := s.User("演示思考:默认折叠、展开、流式、超长、多段与用时。")

	// 默认折叠(短)
	s.Think("先规划要展示的几种思考样式。", 2)

	// 第二段(多段思考)
	t2 := s.ID("t")
	s.Add(ms(200), core.EvAppend{Item: &core.Thinking{ID: t2, Text: "第二段:对照设计稿检查步骤行栅格与图标列。", Streaming: false}})
	s.Add(0, core.EvDuration{ID: t2, Secs: 3})

	// 超长思考(默认折叠,展开后限高滚动)
	longThink := "需要验证折叠时的标题截断、展开后的最大高度与滚动、以及流式尾部窗口。" +
		"设计稿要求思考正文与工具输出使用相同缩进(TEXT_INSET),图标列贴齐。" +
		"还要确认减弱动态效果时动画直达终态,以及多段思考在同一回合内垂直排列而不互相覆盖。"
	longThink += repeatPad(longThink, 800)
	tLong := s.ID("t")
	s.Add(ms(150), core.EvAppend{Item: &core.Thinking{ID: tLong, Text: longThink, Streaming: false}})
	s.Add(0, core.EvDuration{ID: tLong, Secs: 18})

	// 流式思考(进行中样式):先追加 streaming,不完整播放时可停在流式态;这里收尾展示完成态的流式过程
	tStream := s.ID("t")
	s.Add(ms(100), core.EvAppend{Item: &core.Thinking{ID: tStream, Streaming: true}})
	s.Add(ms(50), core.EvThinkingDelta{ID: tStream, Delta: "流式思考:正在组织"})
	s.Add(ms(50), core.EvThinkingDelta{ID: tStream, Delta: "关于工具卡状态矩阵的文案…"})
	s.Add(ms(80), core.EvItemDone{ID: tStream})
	s.Add(0, core.EvDuration{ID: tStream, Secs: 5})

	s.Say("思考行右侧应显示用时;流式结束后 `Streaming=false`。")
	s.End(u, 42, core.StopEndTurn, DefaultStats(11))
	return s.Steps()
}

func repeatPad(s string, n int) string {
	if len(s) >= n {
		return s
	}
	for len(s) < n {
		s += " 细节包括空态、错误态与可观测性字段。"
	}
	return s
}
