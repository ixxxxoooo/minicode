package demo

import (
	"fmt"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// Step 是脚本里的一步:等待 After 之后产出 Ev。
type Step struct {
	After time.Duration
	Ev    core.Event
}

const (
	fixTitle = "完整修复一个 Bug:路由大小写匹配"
	fixUser  = "路由匹配对大小写敏感:`/Users` 和 `/users` 应该命中同一个处理器。看一下 @{src/router.rs},修好并补上测试。"
	fixLead  = "问题出在路由表直接用原始路径做键:注册和匹配都没有归一化,所以 /Users 找不到 /users。"
	fixMD    = "### 已修复\n\n" +
		"- 新增 `normalize()`:去掉末尾斜杠并转小写,`add` 与 `match_path` 共用。\n" +
		"- 匹配不到时回退到 `fallback`,行为与原先一致。\n" +
		"- 补了 2 条测试,`cargo test router` 12 项全部通过。\n\n" +
		"如果希望保留大小写敏感的路由,可以给 `Router` 加一个开关,我可以接着做。"
)

const oldMatch = "    pub fn match_path(&self, path: &str) -> Option<&Handler> {\n        self.routes.get(path)\n    }"
const newMatch = "    pub fn match_path(&self, path: &str) -> Option<&Handler> {\n        let key = normalize(path);\n        self.routes.get(&key).or(self.fallback.as_ref())\n    }"
const testAdded = `use router_demo::router::Router;

#[test]
fn matches_ignoring_case() {
    let r = router();
    assert!(r.match_path("/Users").is_some());
    assert!(r.match_path("/users").is_some());
}

#[test]
fn matches_trailing_slash() {
    assert!(router().match_path("/users/").is_some());
}`

const cargoLive = "   Compiling router-demo v0.3.1\n     Running tests/router_test.rs\ntest matches_ignoring_case ... ok\n"
const cargoDone = "   Compiling router-demo v0.3.1\n     Running tests/router_test.rs\ntest matches_ignoring_case ... ok\ntest result: ok. 12 passed; 0 failed in 0.42s"

// FixBug 是设计稿里「修复路由大小写」那个完整回合的脚本:思考 → 探索 → 两处编辑 → 权限 → 跑测试 → 待办 → 回复。
// withPermission 为真时在运行测试前插入一次权限请求(停在那里等用户处理);静态回放传 false。
func FixBug(withPermission bool) []Step {
	ms := func(n int) time.Duration { return time.Duration(n) * time.Millisecond }
	var s []Step
	add := func(after time.Duration, ev core.Event) { s = append(s, Step{after, ev}) }

	add(0, core.EvReady{SessionID: "demo-fixbug", Models: core.DefaultModels(), Model: "sonnet", Mode: core.ModeDefault})
	add(0, core.EvTitleChanged{Title: fixTitle})
	add(0, core.EvAppend{Item: &core.User{ID: "u1", Text: fixUser, Files: []string{"src/router.rs"}}})
	add(ms(300), core.EvTurnStarted{})

	// 思考
	add(ms(200), core.EvAppend{Item: &core.Thinking{ID: "t1", Streaming: true}})
	add(ms(150), core.EvThinkingDelta{ID: "t1", Delta: "定位大小写敏感的来源。路由表用原始路径做键,注册与匹配两处都没有归一化。"})
	add(ms(300), core.EvThinkingDelta{ID: "t1", Delta: "最小改动是抽一个 normalize,两处共用,顺带处理末尾斜杠。"})
	add(ms(300), core.EvItemDone{ID: "t1"})
	add(0, core.EvDuration{ID: "t1", Secs: 4})

	// 计划卡(已批准)与一个仍在后台运行的子 Agent(设计稿 work 页的两张卡)
	approved := true
	add(ms(200), core.EvAppend{Item: &core.PlanApproval{ID: "fix-plan", Markdown: "## 路由大小写修复计划\n\n" + PlanMarkdown, Approved: &approved}})
	sub := core.NewSubagent("fix-sub", "src/handlers", "审查 handlers 的错误处理", core.SubBackground)
	sub.Summary = "搜索 to_lowercase"
	sub.TaskID, sub.Tokens, sub.ToolCount, sub.LastTool = "ag1", 6200, 12, "Grep"
	stampSub(sub, 48*time.Second)
	add(ms(150), core.EvAppend{Item: sub})

	// 探索:读、搜、读
	add(ms(200), core.EvAppend{Item: core.NewToolCall("r1", "Read", "读取 src/router.rs").WithInput(map[string]any{"file_path": "src/router.rs"}).WithLocation("src/router.rs")})
	add(ms(500), core.EvToolUpdate{ID: "r1", Patch: core.ToolPatch{Status: core.Status(core.ToolCompleted), AppendContent: []core.ToolContent{&core.TextBlock{Text: lines(86)}}}})
	add(ms(100), core.EvAppend{Item: core.NewToolCall("g1", "Grep", "搜索 match_path").WithInput(map[string]any{"pattern": "match_path", "path": "src/"})})
	add(ms(500), core.EvToolUpdate{ID: "g1", Patch: core.ToolPatch{Status: core.Status(core.ToolCompleted), AppendContent: []core.ToolContent{&core.TextBlock{Text: "Found 5 files\nsrc/router.rs\nsrc/server.rs\ntests/router_test.rs\ntests/case_test.rs\nsrc/lib.rs"}}}})
	add(ms(100), core.EvAppend{Item: core.NewToolCall("r2", "Read", "读取 tests/router_test.rs").WithInput(map[string]any{"file_path": "tests/router_test.rs"}).WithLocation("tests/router_test.rs")})
	add(ms(400), core.EvToolUpdate{ID: "r2", Patch: core.ToolPatch{Status: core.Status(core.ToolCompleted), AppendContent: []core.ToolContent{&core.TextBlock{Text: lines(64)}}}})

	// 两处编辑
	add(ms(200), core.EvAppend{Item: core.NewToolCall("e1", "Edit", "编辑 src/router.rs").WithInput(map[string]any{"file_path": "src/router.rs"}).WithLocation("src/router.rs")})
	add(ms(700), core.EvToolUpdate{ID: "e1", Patch: core.ToolPatch{Status: core.Status(core.ToolCompleted), AppendContent: []core.ToolContent{&core.DiffBlock{Path: "src/router.rs", Old: oldMatch, New: newMatch, Line: 41}}}})
	add(ms(100), core.EvAppend{Item: core.NewToolCall("e2", "Edit", "编辑 tests/router_test.rs").WithInput(map[string]any{"file_path": "tests/router_test.rs"}).WithLocation("tests/router_test.rs")})
	add(ms(500), core.EvToolUpdate{ID: "e2", Patch: core.ToolPatch{Status: core.Status(core.ToolCompleted), AppendContent: []core.ToolContent{&core.DiffBlock{Path: "tests/router_test.rs", Old: "use router_demo::router::Router;", New: testAdded, Line: 1}}}})

	// 跑测试(可选权限)
	bash := core.NewToolCall("b1", "Bash", "运行 cargo test router").WithInput(map[string]any{"command": "cargo test router"})
	if withPermission {
		add(ms(200), core.EvPromoteToPermission{Request: &core.PermissionRequest{ID: "b1", Tool: bash, Options: []core.PermissionOption{
			{ID: "allow", Label: "允许一次", Kind: core.AllowOnce},
			{ID: "always", Label: "本会话始终允许", Kind: core.AllowAlways},
			{ID: "deny", Label: "拒绝", Kind: core.RejectOnce},
		}}})
		add(ms(1200), core.EvPermissionResolved{ID: "b1", Option: "allow"})
	} else {
		add(ms(200), core.EvAppend{Item: bash})
	}
	add(ms(400), core.EvToolUpdate{ID: "b1", Patch: core.ToolPatch{LiveOutput: core.Str(cargoLive)}})
	add(ms(900), core.EvToolUpdate{ID: "b1", Patch: core.ToolPatch{Status: core.Status(core.ToolCompleted), DurationMs: 2400,
		AppendContent: []core.ToolContent{&core.TerminalBlock{Command: "cargo test router", Output: cargoDone, ExitCode: core.Int(0)}}}})

	// 待办
	add(ms(200), core.EvTodos{ID: "td1", Items: []core.TodoItem{
		{Content: "定位大小写敏感的来源", Status: core.TodoCompleted},
		{Content: "统一用 normalize 归一化路径", Status: core.TodoCompleted},
		{Content: "补充大小写与末尾斜杠的测试", Status: core.TodoCompleted},
	}})

	// 正文:先导语,再分块流式
	add(ms(300), core.EvAppend{Item: &core.Assistant{ID: "a1", Streaming: true}})
	add(0, core.EvTextDelta{ID: "a1", Delta: fixLead + "\n\n"})
	for i := 0; i < len(fixMD); {
		j := min(len(fixMD), i+24)
		for j < len(fixMD) && fixMD[j]&0xC0 == 0x80 { // 不切断 UTF-8 字符
			j++
		}
		add(ms(35), core.EvTextDelta{ID: "a1", Delta: fixMD[i:j]})
		i = j
	}
	add(ms(100), core.EvItemDone{ID: "a1"})

	// 回合收尾
	cost := 0.02
	apiMs, ttft, calls := uint64(61000), uint64(900), 6
	add(0, core.EvTurnStats{ID: "u1", Stats: core.TurnStats{Model: "claude-sonnet-5-5", Input: 44100, Output: 624, CacheRead: 400000, CacheWrite: 4000, CostUSD: &cost, ApiMs: &apiMs, TtftMs: &ttft, Calls: &calls}})
	add(0, core.EvDuration{ID: "u1", Secs: 73})
	// 实测响应耗时(冷启动 1.9 秒 + 首个响应 3.2 秒 + 整个回合 73 秒),脚注显示「用时 1 分 13 秒」,悬停看分段
	add(0, core.EvTurnTiming{ID: "u1", Timing: core.TurnTiming{StartupMs: 1900, FirstMs: 3200, TotalMs: 73000,
		Phases: []core.TimingPhase{
			{Kind: core.StageConn, Ms: 1900}, {Kind: core.StagePrep, Ms: 100}, {Kind: core.StageWait, Ms: 3200, Call: 1},
			{Kind: core.StageWrite, Ms: 1500, Call: 1}, {Kind: core.StageTool, Ms: 2400, Label: "Bash"}, {Kind: core.StageWait, Ms: 4200, Call: 2},
			{Kind: core.StageWrite, Ms: 28000, Call: 2}, {Kind: core.StageAsk, Ms: 18000}, {Kind: core.StageWait, Ms: 9000, Call: 3}, {Kind: core.StageWrite, Ms: 4700, Call: 3}},
		Calls: []core.CallInfo{{Cache: core.CacheTokens{Written: 26170, Uncached: 2}}, {Cache: core.CacheTokens{Read: 26180, Written: 17, Uncached: 4}}, {Cache: core.CacheTokens{Uncached: 30000}}}}})
	add(0, core.EvUsage{Usage: core.Usage{ContextUsed: 76000, ContextSize: 200000, CostUSD: 12.4}})
	add(0, core.EvTurnEnded{Reason: core.StopEndTurn})
	return s
}

func lines(n int) string {
	b := make([]byte, 0, n*6)
	for i := 1; i <= n; i++ {
		b = append(b, fmt.Sprintf("line %d\n", i)...)
	}
	return string(b)
}

// ReplayTasks 给静态截图补上设计稿任务坞各类胶囊(子代理 / 后台命令 / 监听 / 工作流 / 已结束)。
func ReplayTasks(st *core.State) {
	now := st.Now()
	a1 := core.NewBackgroundTask(now-48_000, "ag1", core.TaskAgent, "审查 handlers 的错误处理")
	a1.Summary = `搜索 "unwrap()"`
	sh := core.NewBackgroundTask(now-6*60_000, "sh1", core.TaskShell, "cargo watch -x test")
	sh.Summary = "等待文件变化…"
	sh.LastActivityMs = now - 3*60_000 // 演示停滞
	mon := core.NewBackgroundTask(now-120_000, "mon1", core.TaskMonitor, "监听 target/dev.log")
	mon.Summary = "已触发 2 次"
	wf := core.NewBackgroundTask(now-30_000, "wf1", core.TaskWorkflow, "发布流水线")
	wf.Summary = "构建 · 测试"
	a2 := core.NewBackgroundTask(now-80_000, "ag2", core.TaskAgent, "调研 axum 的路由匹配")
	a2.Status = core.TaskCompleted
	a2.EndedMs = now - 20_000
	a2.Summary = "总结:建议在中间件里归一化路径"
	done := core.NewBackgroundTask(now-200_000, "sh0", core.TaskShell, "npm run dev")
	done.Status = core.TaskStopped
	done.EndedMs = now - 90_000
	done.Summary = "用户停止"
	for _, t := range []*core.BackgroundTask{a1, sh, mon, wf, a2, done} {
		st.Apply(core.EvTaskStarted{Task: t})
	}
	st.Apply(core.EvTaskNotification{TaskID: "ag2", Status: core.TaskCompleted, Summary: a2.Summary})
	st.Apply(core.EvTaskNotification{TaskID: "sh0", Status: core.TaskStopped, Summary: done.Summary})
	st.Apply(core.EvMcpServers{Servers: demoMcpServers()})
	st.Apply(core.EvUsage{Usage: demoContextUsage()})
}
