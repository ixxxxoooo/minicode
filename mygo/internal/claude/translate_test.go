package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

func reduce(t *testing.T, tr *Translator, lines ...string) *core.State {
	t.Helper()
	st := core.NewState()
	for _, l := range lines {
		v := parseLine(l)
		if v == nil {
			t.Fatalf("测试数据不是合法 JSON:%s", l)
		}
		var evs []core.Event
		switch str(v, "type") {
		case "control_request":
			evs, _ = tr.ControlRequest(v)
		default:
			evs = tr.Handle(v)
		}
		for _, e := range evs {
			st.Apply(e)
		}
	}
	return st
}

func TestPartialJSONString(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{`{"plan":"第一`, "第一", true},
		{`{"plan": "a\nb\"c"`, "a\nb\"c", true},
		{`{"plan":"半截转义\`, "半截转义", true},
		{`{"plan":"\u4f60\u597d`, "你好", true},
		{`{"plan":"\u4f6`, "", true},
		{`{"plan":"\ud83d\ude00x`, "😀x", true},
		{`{"plan":"\ud83d`, "", true},
		{`{"other":"x"`, "", false},
		{`{"plan":12`, "", false},
	}
	for _, c := range cases {
		got, ok := partialJSONString(c.in, "plan")
		if got != c.want || ok != c.ok {
			t.Errorf("%q → (%q,%v),期望 (%q,%v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestOptionsArgs(t *testing.T) {
	mode, eff := core.ModePlan, core.EffortHigh
	o := Options{Cwd: "/w", SessionID: "s1", Model: "opus", Mode: &mode, Effort: &eff, AddDirs: []string{"/x"}}
	a := strings.Join(o.Args(), " ")
	for _, want := range []string{"--input-format stream-json", "--permission-prompt-tool=stdio", "--allow-dangerously-skip-permissions",
		"--session-id s1", "--model opus", "--permission-mode plan", "--effort high", "--add-dir /x", "--thinking-display summarized"} {
		if !strings.Contains(a, want) {
			t.Errorf("参数缺少 %q:%s", want, a)
		}
	}
	if strings.Contains(a, "--resume") {
		t.Errorf("新会话不应带 --resume:%s", a)
	}
	// 恢复 + 分叉
	o = Options{Resume: "old", Fork: true, SessionID: "new"}
	a = strings.Join(o.Args(), " ")
	if !strings.Contains(a, "--resume old") || !strings.Contains(a, "--fork-session") || !strings.Contains(a, "--session-id new") {
		t.Errorf("分叉参数不对:%s", a)
	}
	if o.EffectiveSessionID() != "new" || (Options{Resume: "old"}).EffectiveSessionID() != "old" {
		t.Error("有效会话 id 推断不对")
	}
	// default 模型不传 --model
	if strings.Contains(strings.Join(Options{Model: "default"}.Args(), " "), "--model") {
		t.Error("default 不应传 --model")
	}
	env := Options{ExtraEnv: []core.EnvVar{{Key: "MINICODE_LAUNCH_AUTOCOMPACT", Value: "auto"}}}
	if !strings.Contains(strings.Join(env.Args(), " "), "--autocompact auto") {
		t.Error("MINICODE_LAUNCH_AUTOCOMPACT 应生成 --autocompact")
	}
}

func TestUserMessageAndResponses(t *testing.T) {
	m := UserMessage("你好", nil, true, "u-1")
	if m["priority"] != "next" || m["uuid"] != "u-1" {
		t.Fatalf("引导消息应带 priority:next 与 uuid:%v", m)
	}
	if _, has := UserMessage("x", nil, false, "u")["priority"]; has {
		t.Error("普通消息不应带 priority")
	}
	// 图片读不到时退化成文字说明
	c := sub(UserMessage("x", []string{"/不存在.png"}, false, "u"), "message")["content"].([]any)
	if len(c) != 2 || !strings.Contains(str(asObj(c[0]), "text"), "图片读取失败") {
		t.Errorf("图片读取失败应退化成文字:%v", c)
	}

	p := &Pending{RequestID: "r1", ToolUseID: "t1", ToolName: "Bash", Input: obj{"command": "ls -la"}}
	allow := sub(sub(p.PermissionResponse("allow_once"), "response"), "response")
	if allow["behavior"] != "allow" || allow["toolUseID"] != "t1" {
		t.Errorf("允许应答不对:%v", allow)
	}
	always := sub(sub(p.PermissionResponse("allow_always"), "response"), "response")
	rule := asObj(asObj(asObj(always["updatedPermissions"].([]any)[0])["rules"].([]any)[0]))
	if rule["toolName"] != "Bash" || rule["ruleContent"] != "ls:*" {
		t.Errorf("总是允许应按命令首词生成规则:%v", rule)
	}
	deny := sub(sub(p.PermissionResponse("reject"), "response"), "response")
	if deny["behavior"] != "deny" {
		t.Errorf("拒绝应答不对:%v", deny)
	}

	q := &Pending{RequestID: "r2", Kind: PendQuestion, Input: obj{"questions": []any{obj{"question": "选哪个?"}}}}
	ans := sub(sub(sub(q.QuestionResponse([][]string{{"A", "B"}}), "response"), "response"), "updatedInput")["answers"].(obj)
	if ans["选哪个?"] != "A, B" {
		t.Errorf("提问答案应按问题写回:%v", ans)
	}

	plan := &Pending{RequestID: "r3", Kind: PendPlan, Input: obj{"plan": "x"}}
	mode := core.ModeAcceptEdits
	perms := sub(sub(plan.PlanResponse(true, &mode), "response"), "response")["updatedPermissions"].([]any)
	if asObj(perms[0])["mode"] != "acceptEdits" {
		t.Errorf("批准计划应切到所选模式:%v", perms)
	}
	if sub(sub(plan.PlanResponse(false, nil), "response"), "response")["behavior"] != "deny" {
		t.Error("不批准应拒绝")
	}
}

func TestBuildTool(t *testing.T) {
	tl := BuildTool("1", "Read", obj{"file_path": "/w/src/a.rs"}, "/w")
	if tl.Title != "读取 src/a.rs" || len(tl.Locations) != 1 {
		t.Errorf("读取标题不对:%q", tl.Title)
	}
	tl = BuildTool("2", "Edit", obj{"file_path": "/w/a.rs", "old_string": "x", "new_string": "y"}, "/w")
	if d, ok := tl.Content[0].(*core.DiffBlock); !ok || d.Old != "x" || d.New != "y" {
		t.Errorf("编辑应带 Diff:%#v", tl.Content)
	}
	if got := BuildTool("3", "mcp__github__get_pr", obj{}, "").Title; got != "github · get_pr" {
		t.Errorf("MCP 标题不对:%q", got)
	}
	if got := ShortPath("/a/b/c/d/e", "/zzz"); got != "…/c/d/e" {
		t.Errorf("长路径应保留末三段:%q", got)
	}
	if got := BuildTool("4", "Bash", obj{"command": strings.Repeat("长", 100)}, "").Title; len([]rune(got)) != 81 {
		t.Errorf("命令标题应截到 80 字加省略号:%d", len([]rune(got)))
	}
}

func TestStreamingDedupe(t *testing.T) {
	tr := NewTranslator("/w")
	st := reduce(t, tr,
		`{"type":"stream_event","event":{"type":"message_start","message":{"id":"m1"}}}`,
		`{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"thinking"}}}`,
		`{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"想"}}}`,
		`{"type":"stream_event","event":{"type":"content_block_stop","index":0}}`,
		`{"type":"stream_event","event":{"type":"content_block_start","index":1,"content_block":{"type":"text"}}}`,
		`{"type":"stream_event","event":{"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"答"}}}`,
		`{"type":"stream_event","event":{"type":"content_block_stop","index":1}}`,
		`{"type":"assistant","message":{"id":"m1","content":[{"type":"thinking","thinking":"想"},{"type":"text","text":"答"}]}}`,
	)
	if len(st.Items) != 2 {
		t.Fatalf("流式与完整消息应去重成 2 条:%d", len(st.Items))
	}
	// 没流式过的完整消息直接追加
	st = reduce(t, NewTranslator("/w"), `{"type":"assistant","message":{"id":"m2","content":[{"type":"text","text":"直接"}]}}`)
	if len(st.Items) != 1 {
		t.Fatalf("未流式的完整消息应追加:%d", len(st.Items))
	}
}

func TestResultStatsAndStopReasons(t *testing.T) {
	tr := NewTranslator("/w")
	tr.Interrupted = true
	var got []core.Event
	for _, e := range tr.Handle(parseLine(`{"type":"result","is_error":false,"total_cost_usd":1.5,"num_turns":2,"duration_api_ms":900,"ttft_ms":120,
		"usage":{"input_tokens":100,"output_tokens":50,"cache_read_input_tokens":900,"cache_creation_input_tokens":10,"output_tokens_details":{"thinking_tokens":7}},
		"model_usage_since_last_result":{"claude-sonnet-5-5":{"costUSD":0.25,"outputTokens":50,"canonicalModel":"claude-sonnet-5-5"}},
		"modelUsage":{"claude-sonnet-5-5":{"contextWindow":200000}}}`)) {
		got = append(got, e)
	}
	var stats *core.TurnStats
	var reason core.StopReason = -1
	var usage *core.Usage
	for _, e := range got {
		switch x := e.(type) {
		case core.EvTurnStats:
			stats = &x.Stats
		case core.EvTurnEnded:
			reason = x.Reason
		case core.EvUsage:
			usage = &x.Usage
		}
	}
	if stats == nil || stats.Input != 100 || stats.CacheRead != 900 || stats.Thinking != 7 || stats.CostUSD == nil || *stats.CostUSD != 0.25 || stats.Model != "claude-sonnet-5-5" {
		t.Fatalf("回合统计不对(费用应取单回合而不是会话累计):%+v", stats)
	}
	if *stats.Calls != 2 || *stats.TtftMs != 120 {
		t.Errorf("轮数与首字延迟不对:%+v", stats)
	}
	if reason != core.StopCancelled {
		t.Errorf("请求过中断应以已取消结束:%v", reason)
	}
	if usage == nil || usage.ContextSize != 200000 || usage.CostUSD != 1.5 {
		t.Errorf("用量不对:%+v", usage)
	}
	if tr.Interrupted {
		t.Error("回合结束后应清掉中断标记")
	}
	// 错误回合
	evs := NewTranslator("/w").Handle(parseLine(`{"type":"result","is_error":true,"result":"额度用完","usage":{}}`))
	last := evs[len(evs)-1].(core.EvTurnEnded)
	if last.Reason != core.StopError {
		t.Errorf("错误回合原因不对:%v", last.Reason)
	}
}

func TestAPIErrorShownOnce(t *testing.T) {
	tr := NewTranslator("/w")
	st := reduce(t, tr,
		`{"type":"assistant","isApiErrorMessage":true,"message":{"id":"e","model":"<synthetic>","content":[{"type":"text","text":"API Error: 529"}]}}`,
		`{"type":"result","is_error":true,"result":"API Error: 529","usage":{}}`)
	n := 0
	for _, it := range st.Items {
		switch it.(type) {
		case *core.Notice, *core.Assistant:
			n++
		}
	}
	if n != 1 {
		t.Fatalf("API 错误只应显示一份:%d 条", n)
	}
}

func TestPlanStreamingThenPermission(t *testing.T) {
	tr := NewTranslator("/w")
	st := reduce(t, tr,
		`{"type":"stream_event","api_message_id":"m","event":{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"p1","name":"ExitPlanMode"}}}`,
		`{"type":"stream_event","api_message_id":"m","event":{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"plan\":\"第一步"}}}`,
	)
	pa, ok := st.Items[len(st.Items)-1].(*core.PlanApproval)
	if !ok || !pa.Streaming || pa.Markdown != "第一步" {
		t.Fatalf("计划应流式出现:%#v", st.Items)
	}
	for _, e := range tr.Handle(parseLine(`{"type":"stream_event","api_message_id":"m","event":{"type":"content_block_stop","index":0}}`)) {
		st.Apply(e)
	}
	evs, _ := tr.ControlRequest(parseLine(`{"type":"control_request","request_id":"r","request":{"subtype":"can_use_tool","tool_name":"ExitPlanMode","tool_use_id":"p1","input":{"plan":"第一步\n第二步"}}}`))
	for _, e := range evs {
		st.Apply(e)
	}
	plans := 0
	for _, it := range st.Items {
		if p, ok := it.(*core.PlanApproval); ok {
			plans++
			if p.Streaming || p.Markdown != "第一步\n第二步" {
				t.Errorf("权限请求到达时应定稿:%#v", p)
			}
		}
	}
	if plans != 1 {
		t.Fatalf("定稿不应重复追加:%d 张", plans)
	}
	// 另一个计划生成到一半回合就结束:按未批准收尾
	tr2 := NewTranslator("/w")
	st2 := reduce(t, tr2,
		`{"type":"stream_event","api_message_id":"m","event":{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"p2","name":"ExitPlanMode"}}}`,
		`{"type":"result","is_error":false,"usage":{}}`)
	p2 := st2.Items[len(st2.Items)-1].(*core.PlanApproval)
	if p2.Approved == nil || *p2.Approved {
		t.Errorf("回合结束时未定稿的计划应标成未批准:%#v", p2)
	}
}

func TestSubagentAndBackgroundTask(t *testing.T) {
	tr := NewTranslator("/w")
	st := reduce(t, tr,
		`{"type":"assistant","message":{"id":"a","content":[{"type":"tool_use","id":"ag1","name":"Agent","input":{"description":"调研","subagent_type":"explore","run_in_background":true}}]}}`,
		`{"type":"assistant","parent_tool_use_id":"ag1","message":{"id":"b","content":[{"type":"tool_use","id":"r1","name":"Read","input":{"file_path":"/w/x"}}]}}`,
		`{"type":"system","subtype":"task_started","task_id":"tk1","tool_use_id":"ag1","task_type":"local_agent","description":"调研"}`,
	)
	card := st.Items[0].(*core.SubagentCard)
	if card.Status != core.SubBackground || len(card.Children) != 1 {
		t.Fatalf("后台子 Agent 应在后台且带嵌套的工具:%#v", card)
	}
	if !tr.TaskIsLive("tk1") {
		t.Error("本进程见过的任务应算在运行")
	}
	// 启动结果不是完成
	for _, e := range tr.Handle(parseLine(`{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"ag1","content":"Async agent launched"}]},"tool_use_result":{"isAsync":true,"agentId":"tk1"}}`)) {
		st.Apply(e)
	}
	if card.Status != core.SubBackground {
		t.Errorf("后台启动不应被当作完成:%v", card.Status)
	}
	for _, e := range tr.Handle(parseLine(`{"type":"system","subtype":"task_notification","task_id":"tk1","status":"completed","summary":"完成了"}`)) {
		st.Apply(e)
	}
	if tr.TaskIsLive("tk1") {
		t.Error("终态通知后任务不再算在运行")
	}
	// 前台子 Agent 结束带统计
	tr = NewTranslator("/w")
	st = reduce(t, tr,
		`{"type":"assistant","message":{"id":"a","content":[{"type":"tool_use","id":"ag2","name":"Task","input":{"description":"查","subagent_type":"general-purpose"}}]}}`,
		`{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"ag2","content":"结果文本"}]},"tool_use_result":{"totalTokens":1200,"totalToolUseCount":3,"totalDurationMs":4000}}`)
	c2 := st.Items[0].(*core.SubagentCard)
	if c2.Status != core.SubCompleted || c2.Tokens != 1200 || c2.ToolCount != 3 || c2.Summary != "结果文本" {
		t.Errorf("前台子 Agent 结束应带统计与摘要:%#v", c2)
	}
}

func TestQuestionAndTodos(t *testing.T) {
	tr := NewTranslator("/w")
	st := reduce(t, tr,
		`{"type":"assistant","message":{"id":"a","content":[{"type":"tool_use","id":"td","name":"TodoWrite","input":{"todos":[{"content":"写","status":"in_progress","activeForm":"写代码"},{"content":"测","status":"pending"}]}}]}}`,
		`{"type":"control_request","request_id":"q","request":{"subtype":"can_use_tool","tool_name":"AskUserQuestion","tool_use_id":"qq","input":{"questions":[{"header":"库","question":"用哪个?","multiSelect":false,"options":[{"label":"A","description":"a"},{"label":"B"}]}]}}}`)
	td := st.Items[0].(*core.Todos)
	if len(td.Items) != 2 || td.Items[0].Status != core.TodoInProgress || td.Items[0].ActiveForm != "写代码" {
		t.Errorf("待办解析不对:%#v", td)
	}
	q := st.Items[1].(*core.QuestionRequest)
	if len(q.Questions) != 1 || len(q.Questions[0].Options) != 2 || q.Questions[0].Options[0].Description != "a" {
		t.Errorf("提问解析不对:%#v", q)
	}
	if got := tr.TakePending("qq"); got == nil || got.Kind != PendQuestion {
		t.Error("提问应挂起等待应答")
	}
	// 不支持的控制请求立即回复
	_, reply := tr.ControlRequest(parseLine(`{"type":"control_request","request_id":"x","request":{"subtype":"whatever"}}`))
	if sub(reply, "response")["subtype"] != "error" {
		t.Errorf("不支持的请求应立即回错误:%v", reply)
	}
	// 撤销请求
	reduce(t, tr, `{"type":"control_request","request_id":"c","request":{"subtype":"can_use_tool","tool_name":"Bash","tool_use_id":"b1","input":{"command":"ls"}}}`)
	evs := tr.CancelRequest("c")
	if len(evs) != 1 {
		t.Fatalf("撤销请求应产出一个事件:%v", evs)
	}
	if e, ok := evs[0].(core.EvPermissionResolved); !ok || e.Option != "reject" {
		t.Errorf("撤销应标成拒绝:%v", evs[0])
	}
}

func TestBashResults(t *testing.T) {
	tr := NewTranslator("/w")
	st := reduce(t, tr,
		`{"type":"assistant","message":{"id":"a","content":[{"type":"tool_use","id":"b1","name":"Bash","input":{"command":"false"}}]}}`,
		`{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"b1","is_error":true,"content":"Exit code 2\nboom"}]}}`)
	tl := st.Items[0].(*core.ToolCall)
	tb := tl.Content[len(tl.Content)-1].(*core.TerminalBlock)
	if tl.Status != core.ToolFailed || tb.ExitCode == nil || *tb.ExitCode != 2 {
		t.Errorf("失败应解析出退出码 2:%v %#v", tl.Status, tb)
	}
	// 后台 Bash:不当作终端输出,关联任务
	tr = NewTranslator("/w")
	st = reduce(t, tr,
		`{"type":"assistant","message":{"id":"a","content":[{"type":"tool_use","id":"b2","name":"Bash","input":{"command":"sleep 99","run_in_background":true}}]}}`,
		`{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"b2","content":"Command running in background with ID: bg9. Output is being written to: /tmp/x/bg9.output."}]},"tool_use_result":{"backgroundTaskId":"bg9"}}`)
	tl = st.Items[0].(*core.ToolCall)
	if tl.TaskID != "bg9" || len(tl.Content) != 0 {
		t.Errorf("后台 Bash 应关联任务且没有终端输出:%#v", tl)
	}
	if got := backgroundOutputFile("Output is being written to: /tmp/x/bg9.output."); got != "/tmp/x/bg9.output" {
		t.Errorf("输出文件解析不对:%q", got)
	}
}

func TestReplayLines(t *testing.T) {
	lines := []string{
		`{"type":"ai-title","aiTitle":"修复路由"}`,
		`{"type":"user","uuid":"u-1","timestamp":"2026-10-09T10:00:00.000Z","message":{"content":"帮我修复"}}`,
		`{"type":"user","isMeta":true,"timestamp":"2026-10-09T10:00:01.000Z","message":{"content":"<command-name>x</command-name>"}}`,
		`{"type":"assistant","timestamp":"2026-10-09T10:00:05.000Z","message":{"id":"m1","model":"claude-sonnet-5-5","usage":{"input_tokens":10,"output_tokens":5},"content":[{"type":"thinking","thinking":"想想"},{"type":"text","text":"好的"}]}}`,
		`{"type":"assistant","timestamp":"2026-10-09T10:00:09.000Z","message":{"id":"m2","model":"claude-sonnet-5-5","usage":{"input_tokens":20,"output_tokens":8},"content":[{"type":"tool_use","id":"t1","name":"Read","input":{"file_path":"/w/a"}}]}}`,
		`{"type":"user","timestamp":"2026-10-09T10:00:10.000Z","message":{"content":[{"type":"tool_result","tool_use_id":"t1","content":"内容"}]}}`,
		`{"type":"user","isSidechain":true,"message":{"content":"子链路不显示"}}`,
		`{"type":"user","timestamp":"2026-10-09T10:00:11.000Z","message":{"content":"<task-notification><task-id>x</task-id><status>completed</status></task-notification>"}}`,
		`这一行不是 JSON`,
	}
	evs := ReplayLines(lines, "/w")
	if t0, ok := evs[0].(core.EvTitleChanged); !ok || t0.Title != "修复路由" {
		t.Fatalf("标题应排在最前:%v", evs[0])
	}
	st := core.NewState()
	for _, e := range evs {
		st.Apply(e)
	}
	var users, assistants, thinks, tools int
	for _, it := range st.Items {
		switch it.(type) {
		case *core.User:
			users++
		case *core.Assistant:
			assistants++
		case *core.Thinking:
			thinks++
		case *core.ToolCall:
			tools++
		}
	}
	if users != 1 || assistants != 1 || thinks != 1 || tools != 1 {
		t.Fatalf("回放条目数不对(元信息 / 侧链 / 任务通知不应显示成用户消息):u=%d a=%d th=%d tool=%d", users, assistants, thinks, tools)
	}
	var dur, stats int
	var calls int
	for _, e := range evs {
		switch x := e.(type) {
		case core.EvDuration:
			dur++
		case core.EvTurnStats:
			stats++
			calls = *x.Stats.Calls
			if x.Stats.Input != 30 || x.Stats.Output != 13 {
				t.Errorf("回合用量应按消息汇总:%+v", x.Stats)
			}
		}
	}
	if dur != 2 || stats != 1 || calls != 2 {
		t.Errorf("应有思考与回合两个用时、一份回合统计(2 次调用):dur=%d stats=%d calls=%d", dur, stats, calls)
	}
	if _, ok := evs[len(evs)-1].(core.EvTasksChanged); !ok {
		t.Errorf("末尾应追加空的存活快照:%T", evs[len(evs)-1])
	}
}

func TestSessionIndex(t *testing.T) {
	dir := t.TempDir()
	proj := filepath.Join(dir, EncodeProjectDir("/Users/x/my-app"))
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(proj, "s1.jsonl")
	write := func(s string, appendMode bool) {
		flag := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
		if appendMode {
			flag = os.O_CREATE | os.O_WRONLY | os.O_APPEND
		}
		fh, err := os.OpenFile(f, flag, 0o644)
		if err != nil {
			t.Fatal(err)
		}
		fh.WriteString(s)
		fh.Close()
	}
	write(`{"type":"user","cwd":"/Users/x/my-app","message":{"content":"第一个问题"}}`+"\n", false)
	os.WriteFile(filepath.Join(proj, "empty.jsonl"), []byte(`{"type":"summary"}`+"\n"), 0o644)
	now := time.Now()
	ix := BuildIndex(dir, now)
	if len(ix.Projects) != 1 || ix.Projects[0].Name != "my-app" || ix.Projects[0].Path != "/Users/x/my-app" {
		t.Fatalf("项目应取自会话内的 cwd:%+v", ix.Projects)
	}
	if ss := ix.Projects[0].Sessions; len(ss) != 1 || ss[0].Title != "第一个问题" || ss[0].Label != "刚刚" || ss[0].Group != "今天" {
		t.Fatalf("会话标题 / 时间不对(没有用户输入的会话不展示):%+v", ss)
	}
	if loc, ok := ix.Locate("s1"); !ok || loc.Cwd != "/Users/x/my-app" || !strings.HasSuffix(loc.Path, "s1.jsonl") {
		t.Errorf("按 id 反查不对:%+v", loc)
	}
	// 增量:追加 ai-title 后标题变成 ai-title,半行不吸收
	write(`{"type":"ai-title","aiTitle":"AI 起的标题"}`+"\n"+`{"type":"ai-ti`, true)
	ix = BuildIndex(dir, now)
	if got := ix.Projects[0].Sessions[0].Title; got != "AI 起的标题" {
		t.Errorf("增量扫描应读到 ai-title:%q", got)
	}
	if !SessionOnDisk(dir, "/Users/x/my-app", "s1") || SessionOnDisk(dir, "/Users/x/my-app", "nope") {
		t.Error("会话落盘检测不对")
	}
	if err := DeleteSession(dir, "/Users/x/my-app", "../evil"); err == nil {
		t.Error("不合法的会话 id 应拒绝")
	}
	if err := DeleteSession(dir, "/Users/x/my-app", "s1"); err != nil || SessionOnDisk(dir, "/Users/x/my-app", "s1") {
		t.Errorf("删除会话失败:%v", err)
	}
}

func TestRelTime(t *testing.T) {
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.Local)
	cases := []struct {
		t   time.Time
		rel string
		grp string
		clk string
	}{
		{now.Add(-10 * time.Second), "刚刚", "今天", "刚刚"},
		{now.Add(-5 * time.Minute), "5 分钟", "今天", "5 分钟"},
		{now.Add(-2 * time.Hour), "2 小时", "今天", "2 小时"},
		{time.Date(2026, 11, 14, 18, 20, 0, 0, time.Local), "昨天", "昨天", "18:20"},
		{time.Date(2026, 11, 11, 9, 0, 0, 0, time.Local), "周三", "更早", "周三"},
		{time.Date(2026, 10, 28, 9, 0, 0, 0, time.Local), "10月28日", "更早", "10月28日"},
	}
	for _, c := range cases {
		g, k := TimeLabels(c.t, now)
		if r := RelTime(c.t, now); r != c.rel || g != c.grp || k != c.clk {
			t.Errorf("%v → (%s,%s,%s),期望 (%s,%s,%s)", c.t, r, g, k, c.rel, c.grp, c.clk)
		}
	}
}

func TestRewindParse(t *testing.T) {
	o := ParseRewind("u1", true, true, obj{"canRewind": true, "filesChanged": []any{"/a", "/b"}, "insertions": float64(1), "deletions": float64(2)}, "")
	if !o.CanRewind || !o.HasChanges() || len(o.Files) != 2 || o.Insertions != 1 || o.Deletions != 2 {
		t.Errorf("预览应答解析不对:%+v", o)
	}
	o = ParseRewind("u1", true, true, obj{"canRewind": false, "error": "No file checkpoint found for this message."}, "")
	if o.CanRewind || !strings.Contains(o.Reason(), "检查点") {
		t.Errorf("没有检查点应给出人话:%+v", o)
	}
	o = ParseRewind("u1", false, false, nil, "Failed to rewind")
	if o.Error != "Failed to rewind" || o.DryRun {
		t.Errorf("错误应答:%+v", o)
	}
}

func TestParseExitCode(t *testing.T) {
	for in, want := range map[string]int{"Exit code 2\nx": 2, "Exit code -1": -1, "Exit code abc": 1, "random": 1, "Exit code 127": 127} {
		if got := parseExitCode(in); got != want {
			t.Errorf("%q → %d,期望 %d", in, got, want)
		}
	}
}

// 随消息发出的 IDE 上下文块在回放历史时不能显示成用户输入。
func TestReplayDropsIDEContext(t *testing.T) {
	lines := []string{
		`{"type":"user","uuid":"u-1","timestamp":"2026-10-09T10:00:00.000Z","message":{"content":[{"type":"text","text":"<ide-context>\n当前文件:a.go\n</ide-context>"},{"type":"text","text":"帮我看看"}]}}`,
	}
	st := core.NewState()
	for _, e := range ReplayLines(lines, "/w") {
		st.Apply(e)
	}
	if len(st.Items) != 1 {
		t.Fatalf("应只有一条用户消息:%d", len(st.Items))
	}
	if u, ok := st.Items[0].(*core.User); !ok || u.Text != "帮我看看" {
		t.Fatalf("用户消息不应带上下文:%#v", st.Items[0])
	}
}

func TestUserMessageCtxBlock(t *testing.T) {
	m := UserMessageCtx("问题", nil, false, "u", "文件 a.go")
	blocks := m["message"].(obj)["content"].([]any)
	if len(blocks) != 2 || !strings.HasPrefix(blocks[0].(obj)["text"].(string), IDEContextTag) {
		t.Fatalf("上下文应是正文前的独立文本块:%#v", blocks)
	}
	if m2 := UserMessage("问题", nil, false, "u"); len(m2["message"].(obj)["content"].([]any)) != 1 {
		t.Fatal("无上下文时只有正文块")
	}
}
