package demo

import (
	"fmt"
	"strings"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// Send 处理界面命令:演示后端也能「真的」互动——发消息会得到回复,
// 含「权限 / 提问 / 计划」字样的提示会停在对应的待处理卡片上,处理后接着播放。
func (b *Backend) Send(cmd core.Command) {
	log.Debug("命令", "cmd", cmd)
	switch c := cmd.(type) {
	case core.CmdPrompt:
		b.prompt(c)
	case core.CmdCancel:
		b.mu.Lock()
		b.gen++ // 让正在播放的协程退出
		sink := b.sink
		b.pending = nil
		b.mu.Unlock()
		if sink != nil {
			sink(core.EvTurnEnded{Reason: core.StopCancelled})
		}
	case core.CmdResolvePermission:
		b.resume(c.ID, core.EvPermissionResolved{ID: c.ID, Option: c.Option})
	case core.CmdAnswerQuestion:
		b.resume(c.ID, core.EvQuestionAnswered{ID: c.ID, Answers: c.Answers})
	case core.CmdResolvePlan:
		if c.Mode != nil && c.Approve {
			b.mu.Lock()
			sink := b.sink
			b.mu.Unlock()
			if sink != nil {
				sink(core.EvModeChanged{Mode: *c.Mode})
			}
		}
		b.resume(c.ID, core.EvPlanResolved{ID: c.ID, Approved: c.Approve})
	case core.CmdRefreshMcp:
		b.mu.Lock()
		sink := b.sink
		b.mu.Unlock()
		if sink != nil && c.Runtime {
			sink(core.EvMcpServers{Servers: demoMcpServers()})
		}
	case core.CmdRefreshUsage:
		u := demoContextUsage()
		if !c.Context && !c.Limits {
			break
		}
		if c.Context {
			u.ContextQueriedAt = time.Now().Unix()
			u.ContextEstimated = false
		}
		b.mu.Lock()
		sink := b.sink
		b.mu.Unlock()
		if sink != nil && c.Context {
			sink(core.EvUsage{Usage: u})
		}
	case core.CmdMcpToggle:
		// 演示:只记日志,状态在下次 Refresh 时仍来自脚本
		log.Info("演示 MCP 切换", "name", c.Name, "on", c.Enabled)
	case core.CmdEditUser:
		b.mu.Lock()
		sink := b.sink
		b.mu.Unlock()
		if sink != nil && c.Text != "" {
			sink(core.EvAppend{Item: &core.User{ID: b.nextID("iu"), Text: c.Text}})
			b.prompt(core.CmdPrompt{Text: c.Text})
		}
	case core.CmdCompact:
		b.mu.Lock()
		sink := b.sink
		b.mu.Unlock()
		if sink != nil {
			sink(core.EvAppend{Item: &core.Notice{Level: core.NoticeInfo, Text: "正在压缩上下文…"}})
		}
		go func() {
			time.Sleep(900 * time.Millisecond)
			b.mu.Lock()
			s := b.sink
			b.mu.Unlock()
			if s == nil {
				return
			}
			u := demoContextUsage()
			u.ContextUsed = u.ContextUsed * 6 / 10
			s(core.EvUsage{Usage: u})
			s(core.EvAppend{Item: &core.Notice{Level: core.NoticeInfo, Text: "已压缩上下文"}})
		}()
	case core.CmdRestartTask:
		b.mu.Lock()
		sink := b.sink
		b.mu.Unlock()
		if sink != nil && c.TaskID != "" {
			now := time.Now().UnixMilli()
			t := core.NewBackgroundTask(now, c.TaskID, core.TaskShell, "演示:重启的任务")
			t.Summary = "重新启动…"
			sink(core.EvTaskStarted{Task: t})
			log.Info("演示重启任务", "task", c.TaskID)
		}
	case core.CmdStopTask:
		b.mu.Lock()
		sink := b.sink
		b.mu.Unlock()
		if sink != nil {
			st := core.TaskStopped
			sink(core.EvTaskUpdated{TaskID: c.TaskID, Status: &st, Description: "演示:已停止"})
			sink(core.EvTaskNotification{TaskID: c.TaskID, Status: core.TaskStopped, Summary: "演示:任务已手动停止"})
			log.Info("演示停止任务", "task", c.TaskID)
		}
	case core.CmdBackgroundCurrent:
		b.mu.Lock()
		sink := b.sink
		b.mu.Unlock()
		if sink != nil {
			now := time.Now().UnixMilli()
			id := "demo-bg-" + c.ToolUseID
			if id == "demo-bg-" {
				id = "demo-bg-all"
			}
			t := core.NewBackgroundTask(now, id, core.TaskShell, "演示:转入后台的命令")
			t.ToolUseID, t.Backgrounded, t.Summary = c.ToolUseID, true, "运行中(后台)…"
			sink(core.EvTaskStarted{Task: t})
			if c.ToolUseID != "" {
				sink(core.EvSubagentStatus{ID: c.ToolUseID, Status: core.SubBackground})
			}
			log.Info("演示转入后台", "tool", c.ToolUseID, "task", id)
		}
	}
}

// resume 用户处理了待办卡片:先送出处理结果,再播放挂起的后续剧情。
func (b *Backend) resume(id string, ev core.Event) {
	b.mu.Lock()
	next := b.pending[id]
	delete(b.pending, id)
	sink := b.sink
	b.mu.Unlock()
	if sink == nil {
		return
	}
	sink(ev)
	if next != nil {
		b.play(next())
	}
}

func (b *Backend) later(id string, f func() []Step) {
	b.mu.Lock()
	if b.pending == nil {
		b.pending = map[string]func() []Step{}
	}
	b.pending[id] = f
	b.mu.Unlock()
}

func (b *Backend) nextID(prefix string) string {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.seq++
	return prefix + string(rune('0'+b.seq%10)) + string(rune('a'+b.seq/10%26))
}

func (b *Backend) prompt(p core.CmdPrompt) {
	uid := b.nextID("iu")
	user := core.EvAppend{Item: &core.User{ID: uid, Text: p.Text, Images: p.Images, Files: p.Files}}
	if p.Steer { // 引导:塞进正在进行的回合,不重开回合
		b.mu.Lock()
		sink := b.sink
		b.mu.Unlock()
		if sink != nil {
			sink(user)
		}
		return
	}
	ms := func(n int) time.Duration { return time.Duration(n) * time.Millisecond }
	head := []Step{{0, user}, {ms(250), core.EvTurnStarted{}}}
	text := strings.ToLower(p.Text)
	switch {
	case strings.Contains(text, "权限"):
		head = append(head, b.permissionScene(uid)...)
	case strings.Contains(text, "多问") || strings.Contains(text, "三个问题"):
		head = append(head, b.questionScene3(uid)...)
	case strings.Contains(text, "提问"):
		head = append(head, b.questionScene(uid)...)
	case strings.Contains(text, "引导"):
		head = append(head, b.steerDemoScene(uid)...)
	case strings.Contains(text, "计划"):
		head = append(head, b.planScene(uid)...)
	case strings.Contains(text, "子代理") || strings.Contains(text, "子 agent"):
		head = append(head, b.subagentScene(uid)...)
	case strings.Contains(text, "任务"):
		head = append(head, b.bgTaskScene(uid)...)
	case strings.Contains(text, "失败"):
		head = append(head, b.failToolScene(uid)...)
	case strings.Contains(text, "长输出"):
		head = append(head, b.longOutputScene(uid)...)
	case strings.Contains(text, "图片"):
		head = append(head, b.imageScene(uid)...)
	case strings.Contains(text, "待办"):
		head = append(head, b.todoScene(uid)...)
	default:
		head = append(head, b.replyScene(uid, "收到。这是演示后端的回复:真实模式下这里会是 Claude 的回答。")...)
	}
	b.play(head)
}

// 收尾:一段正文 + 回合结束。
func (b *Backend) replyScene(uid, text string) []Step {
	ms := func(n int) time.Duration { return time.Duration(n) * time.Millisecond }
	aid := b.nextID("ia")
	s := []Step{{ms(300), core.EvAppend{Item: &core.Assistant{ID: aid, Streaming: true}}}}
	for _, r := range chunks(text, 6) {
		s = append(s, Step{ms(40), core.EvTextDelta{ID: aid, Delta: r}})
	}
	s = append(s, Step{ms(80), core.EvItemDone{ID: aid}},
		Step{0, core.EvDuration{ID: uid, Secs: 3}},
		Step{0, core.EvTurnEnded{Reason: core.StopEndTurn}})
	return s
}

func (b *Backend) permissionScene(uid string) []Step {
	ms := func(n int) time.Duration { return time.Duration(n) * time.Millisecond }
	id := b.nextID("ib")
	bash := core.NewToolCall(id, "Bash", "运行 cargo test router").WithInput(map[string]any{"command": "cargo test router"})
	b.later(id, func() []Step {
		s := []Step{
			{ms(300), core.EvToolUpdate{ID: id, Patch: core.ToolPatch{LiveOutput: core.Str(cargoLive)}}},
			{ms(600), core.EvToolUpdate{ID: id, Patch: core.ToolPatch{Status: core.Status(core.ToolCompleted), DurationMs: 1200,
				AppendContent: []core.ToolContent{&core.TerminalBlock{Command: "cargo test router", Output: cargoDone, ExitCode: core.Int(0)}}}}},
		}
		return append(s, b.replyScene(uid, "测试全部通过。")...)
	})
	return []Step{{ms(200), core.EvPromoteToPermission{Request: &core.PermissionRequest{ID: id, Tool: bash, Options: PermissionOptions()}}}}
}

// PermissionOptions 是 Bash 权限卡的标准三个选项。
func PermissionOptions() []core.PermissionOption {
	return permOptsAll()
}

// questionScene3 三题向导(单选 + 多选 + 自由输入)。
func (b *Backend) questionScene3(uid string) []Step {
	ms := func(n int) time.Duration { return time.Duration(n) * time.Millisecond }
	id := b.nextID("iq")
	b.later(id, func() []Step { return b.replyScene(uid, "收到,三题都记下了。") })
	return []Step{{ms(200), core.EvAppend{Item: &core.QuestionRequest{ID: id, Questions: Questions3()}}}}
}

// steerDemoScene 较长回合(~8s 内多步工具),便于演示 ⌘⏎ 引导与 ⏎ 排队。
func (b *Backend) steerDemoScene(uid string) []Step {
	ms := func(n int) time.Duration { return time.Duration(n) * time.Millisecond }
	var s []Step
	s = append(s,
		Step{ms(400), core.EvAppend{Item: &core.Thinking{ID: "sg1", Streaming: true}}},
		Step{ms(1200), core.EvThinkingDelta{ID: "sg1", Delta: "先读路由与测试,再分批改。"}},
		Step{ms(800), core.EvItemDone{ID: "sg1"}},
	)
	r1 := core.NewToolCall(b.nextID("sr"), "Read", "读取 src/router.rs").WithInput(map[string]any{"file_path": "src/router.rs"}).WithLocation("src/router.rs")
	s = append(s,
		Step{ms(300), core.EvAppend{Item: r1}},
		Step{ms(2000), core.EvToolUpdate{ID: r1.ID, Patch: core.ToolPatch{Status: core.Status(core.ToolCompleted), DurationMs: 800}}},
	)
	g1 := core.NewToolCall(b.nextID("sg"), "Grep", "搜索 match_path").WithInput(map[string]any{"pattern": "match_path", "path": "src/"})
	s = append(s,
		Step{ms(400), core.EvAppend{Item: g1}},
		Step{ms(2200), core.EvToolUpdate{ID: g1.ID, Patch: core.ToolPatch{Status: core.Status(core.ToolCompleted), DurationMs: 600}}},
	)
	r2 := core.NewToolCall(b.nextID("sr2"), "Read", "读取 tests/router_test.rs").WithInput(map[string]any{"file_path": "tests/router_test.rs"}).WithLocation("tests/router_test.rs")
	s = append(s,
		Step{ms(500), core.EvAppend{Item: r2}},
		Step{ms(1800), core.EvToolUpdate{ID: r2.ID, Patch: core.ToolPatch{Status: core.Status(core.ToolCompleted), DurationMs: 700}}},
	)
	// 故意不收尾,回合保持进行中
	s = append(s, Step{ms(300), core.EvAppend{Item: &core.Assistant{ID: b.nextID("sa"), Streaming: true}}})
	return s
}

// Questions 是演示用的两题向导(设计稿 F.ask 的内容)。
func Questions() []core.Question {
	return []core.Question{
		{Header: "斜杠", Question: "末尾斜杠要怎么处理?", Options: []core.QuestionOption{
			{Label: "忽略末尾斜杠", Description: "/users/ 与 /users 等价(推荐)"},
			{Label: "严格区分", Description: "保持现状"},
		}},
		{Header: "大小写", Question: "需要保留大小写敏感的开关吗?", MultiSelect: true, Options: []core.QuestionOption{
			{Label: "加一个开关", Description: "Router::case_sensitive(bool)"},
			{Label: "写进文档", Description: "只在 README 里说明"},
			{Label: "都不要", Description: ""},
		}},
	}
}

func (b *Backend) questionScene(uid string) []Step {
	ms := func(n int) time.Duration { return time.Duration(n) * time.Millisecond }
	id := b.nextID("iq")
	b.later(id, func() []Step { return b.replyScene(uid, "好的,按你的选择来改。") })
	return []Step{{ms(200), core.EvAppend{Item: &core.QuestionRequest{ID: id, Questions: Questions()}}}}
}

// PlanMarkdown 是演示计划正文。
const PlanMarkdown = "1. 抽出 `normalize`,注册与匹配共用\n2. 匹配失败时回退到 fallback\n3. 补 2 条测试并跑通"

func (b *Backend) subagentScene(uid string) []Step {
	ms := func(n int) time.Duration { return time.Duration(n) * time.Millisecond }
	id := b.nextID("is")
	card := core.NewSubagent(id, "code-reviewer", "互动触发的子 Agent", core.SubRunning)
	s := []Step{
		{ms(200), core.EvAppend{Item: card}},
		{ms(400), core.EvAppend{Parent: id, Item: core.NewToolCall(b.nextID("it"), "Read", "读取 main.rs").WithInput(map[string]any{"file_path": "src/main.rs"})}},
	}
	return append(s, b.replyScene(uid, "子 Agent 已派出,可点卡片查看嵌套时间线。")...)
}

func (b *Backend) bgTaskScene(uid string) []Step {
	ms := func(n int) time.Duration { return time.Duration(n) * time.Millisecond }
	now := time.Now().UnixMilli()
	t := core.NewBackgroundTask(now, "ix-task", core.TaskShell, "互动:watch 测试")
	s := []Step{{ms(200), core.EvTaskStarted{Task: t}}}
	return append(s, b.replyScene(uid, "已在任务坞添加一条后台命令,可点胶囊查看。")...)
}

func (b *Backend) failToolScene(uid string) []Step {
	ms := func(n int) time.Duration { return time.Duration(n) * time.Millisecond }
	tid := b.nextID("if")
	tc := core.NewToolCall(tid, "Bash", "运行 false").WithInput(map[string]any{"command": "false"})
	s := []Step{
		{ms(200), core.EvAppend{Item: tc}},
		{ms(500), core.EvToolUpdate{ID: tid, Patch: core.ToolPatch{Status: core.Status(core.ToolFailed), DurationMs: 200,
			AppendContent: []core.ToolContent{&core.TerminalBlock{Command: "false", Output: "exit code 1", ExitCode: core.Int(1)}}}}},
	}
	return append(s, b.replyScene(uid, "命令失败,工具卡应显示错误态。")...)
}

func (b *Backend) longOutputScene(uid string) []Step {
	ms := func(n int) time.Duration { return time.Duration(n) * time.Millisecond }
	tid := b.nextID("il")
	var out strings.Builder
	for i := 1; i <= 40; i++ {
		out.WriteString(fmt.Sprintf("log line %d: 很长的构建输出示例…\n", i))
	}
	tc := core.NewToolCall(tid, "Bash", "运行 verbose 构建").WithInput(map[string]any{"command": "cargo build -vv"})
	s := []Step{{ms(200), core.EvAppend{Item: tc}},
		{ms(600), core.EvToolUpdate{ID: tid, Patch: core.ToolPatch{Status: core.Status(core.ToolCompleted), DurationMs: 3000,
			AppendContent: []core.ToolContent{&core.TerminalBlock{Command: "cargo build -vv", Output: out.String(), ExitCode: core.Int(0)}}}}},
	}
	return append(s, b.replyScene(uid, "长输出应出现限高与「展开」。")...)
}

func (b *Backend) imageScene(uid string) []Step {
	ms := func(n int) time.Duration { return time.Duration(n) * time.Millisecond }
	aid := b.nextID("ii")
	return []Step{
		{ms(300), core.EvAppend{Item: &core.Assistant{ID: aid, Streaming: true}}},
		{ms(80), core.EvTextDelta{ID: aid, Delta: "附带图片时,会话流会显示占位说明。\n\n![示意](演示资源)\n\n也可在发送时带上 Images 字段。"}},
		{ms(80), core.EvItemDone{ID: aid}},
		{0, core.EvDuration{ID: uid, Secs: 2}},
		{0, core.EvTurnEnded{Reason: core.StopEndTurn}},
	}
}

func (b *Backend) todoScene(uid string) []Step {
	ms := func(n int) time.Duration { return time.Duration(n) * time.Millisecond }
	tid := b.nextID("io")
	s := []Step{{ms(200), core.EvTodos{ID: tid, Items: []core.TodoItem{
		{Content: "互动项 1", Status: core.TodoCompleted},
		{Content: "互动项 2", Status: core.TodoInProgress, ActiveForm: "处理互动项 2"},
		{Content: "互动项 3", Status: core.TodoPending},
	}}}}
	return append(s, b.replyScene(uid, "待办已更新。")...)
}

func (b *Backend) planScene(uid string) []Step {
	ms := func(n int) time.Duration { return time.Duration(n) * time.Millisecond }
	id := b.nextID("ip")
	b.later(id, func() []Step { return b.replyScene(uid, "开始按计划实施。") })
	s := []Step{{ms(200), core.EvAppend{Item: &core.PlanApproval{ID: id, Streaming: true}}}}
	acc := ""
	for _, r := range chunks(PlanMarkdown, 10) {
		acc += r
		s = append(s, Step{ms(60), core.EvPlanUpdated{ID: id, Markdown: acc, Streaming: true}})
	}
	return append(s, Step{ms(100), core.EvPlanUpdated{ID: id, Markdown: PlanMarkdown, Streaming: false}})
}

// chunks 按字符(不切断 UTF-8)把文本切成每块 n 个字符。
func chunks(s string, n int) []string {
	var out []string
	r := []rune(s)
	for i := 0; i < len(r); i += n {
		out = append(out, string(r[i:min(len(r), i+n)]))
	}
	return out
}

// 静态回放用的待处理场景(截图与测试):每个都停在等待用户的卡片上。

// PendingPermission 停在 Bash 权限卡上。
func PendingPermission() []Step {
	var s []Step
	for _, st := range FixBug(true) {
		if _, ok := st.Ev.(core.EvPermissionResolved); ok {
			break
		}
		s = append(s, st)
	}
	return s
}

func pendingBase(item core.Item) []Step {
	return []Step{
		{0, core.EvReady{SessionID: "demo-pending", Models: core.DefaultModels(), Model: "sonnet", Mode: core.ModeDefault}},
		{0, core.EvTitleChanged{Title: "完整修复一个 Bug:路由大小写匹配"}},
		{0, core.EvAppend{Item: &core.User{ID: "u1", Text: fixUser, Files: []string{"src/router.rs"}}}},
		{0, core.EvTurnStarted{}},
		{0, core.EvAppend{Item: item}},
	}
}

// PendingQuestion 停在三题向导上(截图 / 测试默认待办)。
func PendingQuestion() []Step {
	return pendingQuestionBase(Questions3())
}

// PendingPlan 停在待批准的计划上。
func PendingPlan() []Step {
	return pendingBase(&core.PlanApproval{ID: "p1", Markdown: PlanMarkdown})
}
