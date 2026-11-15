package core

import (
	"strings"
	"testing"
)

const t0 = int64(1_700_000_000_000)

func newState() *State {
	s := NewState()
	s.Now = func() int64 { return t0 }
	return s
}

func sub(id string, st SubagentStatus) EvAppend {
	return EvAppend{Item: NewSubagent(id, "Explore", "调研", st)}
}

func TestKindFromToolName(t *testing.T) {
	cases := map[string]ToolKind{"Bash": KindExecute, "mcp__github__list": KindMCP, "mcp__playwright__click": KindBrowser, "Agent": KindSubagent, "Read": KindRead, "Nope": KindOther}
	for name, want := range cases {
		if got := KindFromToolName(name); got != want {
			t.Errorf("%s: got %v want %v", name, got, want)
		}
	}
}

func TestToolPatch(t *testing.T) {
	tc := NewToolCall("1", "Bash", "运行测试")
	tc.Apply(ToolPatch{Status: Status(ToolCompleted), AppendContent: []ToolContent{&TextBlock{"ok"}}, DurationMs: 12})
	if !tc.Status.Finished() || len(tc.Content) != 1 || tc.DurationMs != 12 {
		t.Fatalf("%+v", tc)
	}
}

func TestLiveOutputReplacedThenYields(t *testing.T) {
	tc := NewToolCall("1", "Bash", "跑迁移").WithInput(map[string]any{"command": "sleep 9"})
	tc.Apply(ToolPatch{LiveOutput: Str("step 1\n")})
	tc.Apply(ToolPatch{LiveOutput: Str("step 1\nstep 2\n")})
	if len(tc.Content) != 1 {
		t.Fatalf("实时输出应整体替换: %d", len(tc.Content))
	}
	tb := tc.Content[0].(*TerminalBlock)
	if tb.Command != "sleep 9" || tb.Output != "step 1\nstep 2\n" || tb.ExitCode != nil {
		t.Fatalf("%+v", tb)
	}
	tc.Apply(ToolPatch{Status: Status(ToolCompleted), AppendContent: []ToolContent{&TerminalBlock{"sleep 9", "全部输出", Int(0)}}})
	if len(tc.Content) != 1 || tc.Content[0].(*TerminalBlock).ExitCode == nil {
		t.Fatal("最终结果应取代实时输出")
	}
}

func TestTextDeltasMerge(t *testing.T) {
	s := newState()
	for _, d := range []string{"你", "好"} {
		s.Apply(EvTextDelta{ID: "a1", Delta: d})
	}
	a, ok := s.Items[0].(*Assistant)
	if len(s.Items) != 1 || !ok || a.Text != "你好" || !a.Streaming {
		t.Fatalf("%+v", s.Items)
	}
}

func TestSubagentNesting(t *testing.T) {
	s := newState()
	s.Apply(sub("l1", SubRunning))
	s.Apply(EvAppend{Parent: "l1", Item: NewSubagent("l2", "x", "第 2 层", SubRunning)})
	s.Apply(EvAppend{Parent: "l2", Item: NewSubagent("l3", "x", "第 3 层", SubRunning)})
	s.Apply(EvAppend{Parent: "l3", Item: NewToolCall("t1", "Read", "读取")})
	s.Apply(EvToolUpdate{Parent: "l3", ID: "t1", Patch: ToolPatch{Status: Status(ToolCompleted)}})
	if len(s.Items) != 1 {
		t.Fatal("只有最外层在顶层")
	}
	l3 := s.Items[0].(*SubagentCard).Children[0].(*SubagentCard).Children[0].(*SubagentCard)
	if tc := l3.Children[0].(*ToolCall); tc.ID != "t1" || tc.Status != ToolCompleted {
		t.Fatalf("%+v", tc)
	}
	if s.RowOf("t1") != 0 {
		t.Fatal("RowOf 应指向顶层行 0")
	}
}

func TestMissingParentFallsBackToTopAndCounts(t *testing.T) {
	s := newState()
	before := countOf("session.parent_missing")
	s.Apply(EvAppend{Parent: "不存在", Item: NewToolCall("t", "Bash", "x")})
	if len(s.Items) != 1 || countOf("session.parent_missing") != before+1 {
		t.Fatal("找不到父卡片应落到顶层并计数")
	}
}

func TestPromoteToPermission(t *testing.T) {
	s := newState()
	s.Apply(EvAppend{Item: NewToolCall("t1", "Bash", "ls")})
	s.Apply(EvPromoteToPermission{Request: &PermissionRequest{ID: "t1", Tool: NewToolCall("t1", "Bash", "ls")}})
	if _, ok := s.Items[0].(*PermissionRequest); !ok || len(s.Items) != 1 || !s.AwaitingUser() {
		t.Fatal("工具卡应被升级为权限请求")
	}
}

func TestReadyDoesNotDemoteRunningTurn(t *testing.T) {
	ready := EvReady{SessionID: "s", Model: "default"}
	s := newState()
	s.Apply(ready)
	if s.Phase != PhaseIdle {
		t.Fatal("启动中 → 空闲")
	}
	s = newState()
	s.Apply(EvTurnStarted{})
	s.Apply(ready)
	if s.Phase != PhaseRunning || s.SessionID != "s" {
		t.Fatal("握手应答晚到不能把进行中的回合打回空闲")
	}
}

func TestCancelMarksInterrupted(t *testing.T) {
	s := newState()
	s.Apply(EvTurnStarted{})
	s.Apply(EvAppend{Item: NewToolCall("t", "Bash", "x")})
	s.Apply(EvAppend{Item: &PermissionRequest{ID: "p", Tool: NewToolCall("p", "Edit", "y")}})
	s.Apply(EvTurnEnded{StopCancelled})
	if s.Items[0].(*ToolCall).Status != ToolInterrupted {
		t.Fatal("运行中的工具应被标为被中断")
	}
	if s.Items[1].(*PermissionRequest).Tool.Status != ToolInterrupted {
		t.Fatal("权限卡里等待中的工具也应被中断(旧版漏掉)")
	}
	if n, ok := s.Items[len(s.Items)-1].(*Notice); !ok || n.Text != InterruptedText {
		t.Fatal("末尾应有「已中断」")
	}
	n := len(s.Items)
	s.Apply(EvTurnEnded{StopCancelled})
	if len(s.Items) != n {
		t.Fatal("再次中断不重复追加")
	}
}

func TestCancelKeepsBackgroundSubagent(t *testing.T) {
	s := newState()
	s.Apply(sub("fg", SubRunning))
	s.Apply(sub("bg", SubRunning))
	bt := NewBackgroundTask(t0, "t-bg", TaskAgent, "后台")
	bt.ToolUseID = "bg"
	s.Apply(EvTaskStarted{bt})
	s.Apply(EvTurnEnded{StopCancelled})
	if s.Find("fg").(*SubagentCard).Status != SubStopped || s.Find("bg").(*SubagentCard).Status != SubBackground {
		t.Fatal("前台应停止、后台应继续")
	}
	if s.RunningTaskCount() != 1 {
		t.Fatal("后台任务仍在运行")
	}
}

func TestTaskSyncToCard(t *testing.T) {
	s := newState()
	s.Apply(sub("a1", SubRunning))
	bt := NewBackgroundTask(t0, "t1", TaskAgent, "调研")
	bt.ToolUseID = "a1"
	s.Apply(EvTaskStarted{bt})
	tok, uses := uint64(5200), 3
	s.Apply(EvTaskProgress{TaskID: "t1", Tokens: &tok, ToolUses: &uses, LastTool: "Grep", Summary: "正在搜索"})
	c := s.Items[0].(*SubagentCard)
	if c.Status != SubBackground || c.Tokens != 5200 || c.ToolCount != 3 || c.LastTool != "Grep" || c.Summary != "正在搜索" {
		t.Fatalf("%+v", c)
	}
	s.Apply(EvTaskNotification{TaskID: "t1", Status: TaskCompleted, Summary: "调研完成"})
	c = s.Items[0].(*SubagentCard)
	if c.Status != SubCompleted || c.EndedMs == 0 || c.Summary != "调研完成" || s.RunningTaskCount() != 0 {
		t.Fatalf("%+v", c)
	}
	for _, it := range s.Items {
		if _, ok := it.(*TaskNote); ok {
			t.Fatal("有卡片的子 Agent 不应再插通知行")
		}
	}
}

func TestShellNoteDedup(t *testing.T) {
	s := newState()
	s.Apply(EvTaskStarted{NewBackgroundTask(t0, "sh1", TaskShell, "构建")})
	done := EvTaskNotification{TaskID: "sh1", Status: TaskCompleted, Summary: "构建完成"}
	s.Apply(done)
	s.Apply(done)
	n := 0
	for _, it := range s.Items {
		if _, ok := it.(*TaskNote); ok {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("重复通知应去重,实际 %d 条", n)
	}
}

func TestToolCardLinksTask(t *testing.T) {
	s := newState()
	s.Apply(EvAppend{Item: NewToolCall("b1", "Bash", "启动服务")})
	bt := NewBackgroundTask(t0, "sh1", TaskShell, "启动服务")
	bt.ToolUseID = "b1"
	s.Apply(EvTaskStarted{bt})
	if s.Items[0].(*ToolCall).TaskID != "sh1" {
		t.Fatal("工具卡应关联任务")
	}
}

func TestSnapshotReplacesAndExcludesAmbient(t *testing.T) {
	s := newState()
	s.Apply(EvTaskStarted{NewBackgroundTask(t0, "a", TaskShell, "A")})
	s.Apply(EvTaskStarted{NewBackgroundTask(t0, "b", TaskShell, "B")})
	if s.RunningTaskCount() != 2 {
		t.Fatal()
	}
	amb := NewBackgroundTask(t0, "amb", TaskOther, "内部")
	amb.Ambient = true
	own := NewBackgroundTask(t0, "own", TaskShell, "子代理自己的命令")
	own.OwnedBySub = true
	s.Apply(EvTasksChanged{[]*BackgroundTask{NewBackgroundTask(t0, "b", TaskShell, "B"), amb, own}})
	if s.RunningTaskCount() != 1 || s.Task("a").Status != TaskStopped {
		t.Fatal("快照里没有 a:应视为已结束;环境与私有任务不计数")
	}
}

func TestResumeKeepsOutput(t *testing.T) {
	s := newState()
	s.Apply(EvTaskStarted{NewBackgroundTask(t0, "t", TaskAgent, "第一次")})
	s.Apply(EvTaskOutput{TaskID: "t", Text: "hello"})
	s.Apply(EvTaskNotification{TaskID: "t", Status: TaskCompleted, Summary: "完"})
	s.Apply(EvTaskStarted{NewBackgroundTask(t0, "t", TaskAgent, "第二次")})
	if len(s.Tasks) != 1 || s.Tasks[0].Status != TaskRunning || s.Tasks[0].Output != "hello" {
		t.Fatal("恢复运行应复用任务编号并保留输出")
	}
}

func TestOutputCapKeepsRuneBoundary(t *testing.T) {
	bt := NewBackgroundTask(t0, "x", TaskShell, "")
	bt.PushOutput(t0, strings.Repeat("中", OutputCap)) // 每个 3 字节,必然超限
	if len(bt.Output) > OutputCap || !strings.HasPrefix(bt.Output, "中") {
		t.Fatalf("截断应落在字符边界: len=%d", len(bt.Output))
	}
}

func TestDenyMarksToolDenied(t *testing.T) {
	s := newState()
	s.Apply(EvAppend{Item: &PermissionRequest{ID: "p", Tool: NewToolCall("p", "Bash", "rm"), Options: []PermissionOption{{"allow_once", "允许", AllowOnce}, {"reject", "拒绝", RejectOnce}}}})
	s.Apply(EvPermissionResolved{ID: "p", Option: "reject"})
	if s.Items[0].(*PermissionRequest).Tool.Status != ToolDenied {
		t.Fatal("拒绝应把工具标为被拒绝")
	}
}

func TestAwaitingUser(t *testing.T) {
	s := newState()
	s.Apply(EvAppend{Item: &PlanApproval{ID: "p", Markdown: "# x"}})
	if !s.AwaitingUser() {
		t.Fatal()
	}
	s.Apply(EvPlanResolved{ID: "p", Approved: true})
	if s.AwaitingUser() {
		t.Fatal()
	}
}

func TestNoticeIDsUnique(t *testing.T) {
	s := newState()
	s.Apply(EvNotice{Text: "a"})
	s.Apply(EvAppend{Item: &User{ID: "u"}})
	s.Apply(EvNotice{Text: "b"})
	ids := map[string]bool{}
	for _, it := range s.Items {
		if ids[it.ItemID()] {
			t.Fatalf("id 重复: %s", it.ItemID())
		}
		ids[it.ItemID()] = true
	}
}

func TestFormatters(t *testing.T) {
	eq := func(got, want string) {
		t.Helper()
		if got != want {
			t.Errorf("got %q want %q", got, want)
		}
	}
	eq(FormatElapsed(12_000), "12 秒")
	eq(FormatElapsed(185_000), "3 分 05 秒")
	eq(FormatElapsed(3_720_000), "1 小时 02 分")
	eq(FormatTokens(820), "820")
	eq(FormatTokens(8200), "8.2k")
	eq(FormatTokens(1_300_000), "1.3M")
	eq(FormatTokens(999_960), "1.0M") // 旧版会得到 1000.0k
	eq(TidyLog("a\r\nb\rc\n"), "a\nc\n")
}

func TestStatusMapping(t *testing.T) {
	if TaskStatusFromCLI("killed") != TaskStopped || TaskStatusFromCLI("completed") != TaskCompleted || TaskKindFromType("local_agent") != TaskAgent {
		t.Fatal()
	}
}

func TestModeAndEffort(t *testing.T) {
	m := ModeDefault
	for range AllModes {
		m = m.Next()
	}
	if m != ModeDefault {
		t.Fatal("模式应循环回起点")
	}
	for _, x := range AllModes {
		if got, ok := ParseMode(x.CLI()); !ok || got != x {
			t.Fatal(x)
		}
	}
	if _, ok := ParseMode("nope"); ok {
		t.Fatal()
	}
}

func TestTurnStatsText(t *testing.T) {
	cost, api, ttft, calls := 0.0044, uint64(4233), uint64(2916), 2
	s := TurnStats{Model: "claude-haiku-4-5-20251001", Input: 4, Output: 230, Thinking: 149, CacheRead: 35_613, CacheWrite: 19_732, CostUSD: &cost, ApiMs: &api, TtftMs: &ttft, Calls: &calls}
	if s.ModelLabel() != "Haiku 4.5" {
		t.Fatal(s.ModelLabel())
	}
	label := func(m string) string { return TurnStats{Model: m}.ModelLabel() }
	for in, want := range map[string]string{"claude-sonnet-5-5": "Sonnet 5.5", "claude-3-5-sonnet-20241022": "Sonnet 3.5", "claude-opus-4": "Opus 4", "glm-4.6": "glm-4.6", "claude-sonnet-4-5[1m]": "claude-sonnet-4-5[1m]", "<synthetic>": ""} {
		if got := label(in); got != want {
			t.Errorf("%s → %q want %q", in, got, want)
		}
	}
	if s.ShortText() != "↑ 55.3k ↓ 230 · 缓存 64% · $0.004" {
		t.Fatal(s.ShortText())
	}
	d := s.DetailText()
	if !strings.Contains(d, "其中思考 149") || !strings.Contains(d, "首字 2.9s") || !strings.Contains(d, "2 次调用") {
		t.Fatal(d)
	}
	if (TurnStats{Input: 120, Output: 40}).ShortText() != "↑ 120 ↓ 40" || CostText(0.123) != "$0.12" || !(TurnStats{}).Empty() {
		t.Fatal()
	}
}

func TestTurnStatsAttachToLastUser(t *testing.T) {
	s := newState()
	s.Apply(EvAppend{Item: &User{ID: "u1", Text: "hi"}})
	s.Apply(EvTurnStats{Stats: TurnStats{Input: 5, Output: 7}})
	s.Apply(EvTurnStats{Stats: TurnStats{}}) // 空统计不记录
	if len(s.TurnStats) != 1 || s.TurnStats["u1"].Output != 7 {
		t.Fatal()
	}
}

func TestRewindFlow(t *testing.T) {
	s := newState()
	s.Phase = PhaseIdle
	s.Apply(EvAppend{Item: &User{ID: "u1", Text: "把 README 改一下"}})
	s.Apply(EvUserUUID{ID: "u1", UUID: "uuid-1"})
	if u, err := s.RewindBegin("u1"); err != nil || u != "uuid-1" {
		t.Fatal(err)
	}
	s.Apply(EvRewind{RewindOutcome{UUID: "uuid-1", DryRun: true, CanRewind: true, Files: []string{"/p/a", "/p/b"}, Insertions: 1, Deletions: 2}})
	if s.Rewind == nil || s.Rewind.Stage != RewindReady {
		t.Fatal("应进入待确认")
	}
	if u, ok := s.RewindConfirm(); !ok || u != "uuid-1" {
		t.Fatal()
	}
	s.Apply(EvRewind{RewindOutcome{UUID: "uuid-1", CanRewind: true}})
	n, ok := s.Items[len(s.Items)-1].(*Notice)
	if s.Rewind != nil || !ok || !strings.Contains(n.Text, "2 个文件") || !strings.Contains(n.Text, "把 README 改一下") {
		t.Fatalf("%+v", s.Items[len(s.Items)-1])
	}
}

func TestRewindGuards(t *testing.T) {
	s := newState()
	s.Phase = PhaseIdle
	s.Apply(EvAppend{Item: &User{ID: "u1"}})
	s.Apply(EvUserUUID{ID: "u1", UUID: "uuid-1"})
	if _, err := s.RewindBegin("未知"); err == nil {
		t.Fatal()
	}
	s.Phase = PhaseRunning
	if _, err := s.RewindBegin("u1"); err == nil || !strings.Contains(err.Error(), "回合进行中") {
		t.Fatal()
	}
	s.Phase = PhaseIdle
	_, _ = s.RewindBegin("u1")
	s.Apply(EvRewind{RewindOutcome{UUID: "别的", DryRun: true, CanRewind: true}})
	if s.Rewind.Stage != RewindChecking {
		t.Fatal("别的 uuid 的结果不应影响当前确认")
	}
	s.Rewind = &RewindPrompt{Stage: RewindReady, ItemID: "u1", Outcome: RewindOutcome{UUID: "uuid-1", DryRun: true, CanRewind: true}}
	if _, ok := s.RewindConfirm(); ok {
		t.Fatal("没有改动不能确认")
	}
	s.Rewind = &RewindPrompt{Stage: RewindApplying, ItemID: "u1", UUID: "uuid-1", Files: 1}
	s.Apply(EvRewind{RewindFailed("uuid-1", false, "No file checkpoint found for this message.")})
	if n, ok := s.Items[len(s.Items)-1].(*Notice); !ok || n.Level != NoticeWarning {
		t.Fatal("失败应留警告")
	}
}

func TestMcp(t *testing.T) {
	if McpNormName("claude.ai Claude Docs") != "claude_ai_Claude_Docs" {
		t.Fatal(McpNormName("claude.ai Claude Docs"))
	}
	for in, want := range map[string]McpConn{"connected": McpConnected, "failed": McpFailed, "needs-auth": McpNeedsAuth, "pending": McpPending, "disabled": McpDisabled, "?": McpUnknown} {
		if ParseMcpConn(in) != want {
			t.Errorf("%s", in)
		}
	}
}

func TestSubagentStall(t *testing.T) {
	c := NewSubagent("a", "x", "y", SubRunning)
	c.StampStart(t0)
	if c.Stalled(t0+StallMs-1) || !c.Stalled(t0+StallMs) {
		t.Fatal("超过阈值才算停滞")
	}
	if e, ok := c.ElapsedMs(t0 + 5000); !ok || e != 5000 {
		t.Fatal()
	}
}

func TestUnknownEventCounted(t *testing.T) {
	s := newState()
	before := countOf("session.unknown_event")
	s.Apply(nil)
	if countOf("session.unknown_event") != before+1 {
		t.Fatal("未知事件应计数")
	}
}
