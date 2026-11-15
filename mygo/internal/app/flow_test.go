package app

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/backend"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/demo"
)

func scenarioSteps(id string) []demo.Step {
	for _, s := range demo.Scenarios() {
		if s.ID == id && s.Build != nil {
			return s.Build()
		}
	}
	return nil
}

// sessionApp 一次性回放演示场景(不走 demo.Attach,避免测试里触发 Dock 主线程死锁)。
func sessionApp(t *testing.T, sessID string) (*App, *ui.Tester) {
	t.Helper()
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	a.SetProjects(demo.Projects(), "scenario-lab")
	a.curSess = sessID
	steps := scenarioSteps(sessID)
	if steps == nil {
		t.Fatalf("未知演示场景 %q", sessID)
	}
	demo.Replay(a.st, steps)
	tt := ui.NewTester(a.View, 1280, 800)
	tt.Frame()
	return a, tt
}

func replayApp(t *testing.T, steps []demo.Step) (*App, *ui.Tester) {
	t.Helper()
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	a.SetProjects(demo.Projects(), "fixbug")
	fb := &fakeBE{}
	a.Attach(fb)
	demo.Replay(a.st, steps)
	tt := ui.NewTester(a.View, 1280, 800)
	tt.Frame()
	return a, tt
}

func TestFlowGapsTaskNoteAndFold(t *testing.T) {
	a, tt := sessionApp(t, "flow-gaps-b4")
	if !tt.HasText("cargo watch 已结束") {
		t.Fatal("应渲染 TaskNote 通知行")
	}
	if !tt.HasText("并行 2 个") {
		t.Fatal("应出现并行子 Agent 汇总行")
	}
	if !tt.HasText("已处理") {
		t.Fatal("成功回合应收成「已处理」折叠")
	}
	if !tt.HasText("cargo build") {
		t.Fatal("失败回合应保持步骤展开(cargo build)")
	}
	if !tt.HasText("你拒绝了") {
		t.Fatal("权限拒绝应有专用提示行")
	}
	if !a.turnFoldEnabled() {
		t.Fatal("回合折叠偏好应默认开启")
	}
}

func TestAskDoneEcho(t *testing.T) {
	_, tt := replayApp(t, demo.AskDoneReplay())
	if !tt.HasText("已回答 2 个问题") {
		t.Fatal("应显示回答卡片标题")
	}
	if !tt.HasText("忽略末尾斜杠") {
		t.Fatal("回答胶囊应显示选项文字")
	}
	if !tt.HasText("已跳过") {
		t.Fatal("跳过题应显示灰色已跳过")
	}
}

func TestDockPendingBadge(t *testing.T) {
	_, tt := sessionApp(t, "b4-dock-queue")
	if !tt.HasText("1 / 3") {
		t.Fatal("多张审批应显示 1 / 3")
	}
}

func TestQueueStopAndSteer(t *testing.T) {
	a, tt, _ := pendingApp(t, demo.FixBug(false))
	a.st.Apply(core.EvTurnStarted{})
	tt.Frame()
	a.submitKey("第二条排队", false)
	a.submitKey("第三条排队", false)
	if a.q.len() != 2 {
		t.Fatalf("应有两条排队,实际 %d", a.q.len())
	}
	// 悬停动作:改为引导(通过 take + steerSend 等价验证 UI 文案存在)
	tt.Frame()
	if !tt.HasText("待发送") {
		t.Fatal("排队卡应可见")
	}
	if !a.q.stopAndSend(0) || a.q.preempt != "第二条排队" {
		t.Fatalf("停止并发送应登记 preempt,得 %q", a.q.preempt)
	}
	a.st.Apply(core.EvTurnEnded{Reason: core.StopCancelled})
	a.afterEventForTest(core.EvTurnEnded{Reason: core.StopCancelled})
	// 回合结束后 preempt 会立刻 flush,队列应已清空
	if a.q.preempt != "" || a.q.len() != 1 {
		t.Fatalf("中断结束后应发出 preempt 并保留第二条排队,preempt=%q len=%d", a.q.preempt, a.q.len())
	}
}

func TestQueuePausedResumeWithdraw(t *testing.T) {
	a, tt, _ := pendingApp(t, demo.FixBug(false))
	a.st.Apply(core.EvTurnStarted{})
	a.submitKey("README 示例", false)
	a.afterEventForTest(core.EvTurnEnded{Reason: core.StopError})
	tt.Frame()
	if !tt.HasText("排队已暂停") {
		t.Fatal("中断应暂停排队")
	}
	if err := tt.Click("撤回"); err != nil {
		t.Fatal(err)
	}
	if a.q.len() != 0 {
		t.Fatal("撤回应清空队列")
	}
}

func TestQuestionEscSkip(t *testing.T) {
	a, tt, _ := pendingApp(t, demo.PendingQuestion())
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if a.qa["q1"].cur != 1 {
		t.Fatalf("Esc 应跳过第 1 题并进入第 2 题,cur=%d", a.qa["q1"].cur)
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if a.qa["q1"].cur != 2 {
		t.Fatalf("再 Esc 应进入第 3 题,cur=%d", a.qa["q1"].cur)
	}
}

func TestFlowScrollToLatest(t *testing.T) {
	a, tt := replayApp(t, demo.FixBug(false))
	a.PrepFlowScrollAway(true)
	if a.flowSt.MaxY < 48 {
		t.Skip("流内容过短,无法滚离底部")
	}
	a.ApplyFlowScrollAway()
	tt.Frame()
	if !tt.HasText("回到最新") {
		t.Fatal("滚离底部应显示回到最新")
	}
	if err := tt.Click("回到最新"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !a.flowFollowBottom() {
		t.Fatal("点击后应恢复贴底跟随")
	}
}

func TestCtxStateSixKinds(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	check := func(want string, fn func()) {
		fn()
		if got := a.ctxState(); got != want {
			t.Fatalf("期望 ctxState=%q,实际 %q", want, got)
		}
	}
	check("ok", func() {
		a.ctxCompacting, a.ctxLoading = false, false
		a.st.Usage = core.Usage{ContextSize: 200_000, ContextUsed: 10_000}
	})
	check("warn", func() {
		a.st.Usage = core.Usage{ContextSize: 200_000, ContextUsed: 180_000}
	})
	check("busy", func() { a.ctxCompacting = true })
	check("load", func() {
		a.ctxCompacting = false
		a.ctxLoading = true
	})
	check("none", func() {
		a.ctxLoading = false
		a.st.Usage = core.Usage{LimitsState: core.LimitsUnavailable}
	})
	check("err", func() {
		a.st.Usage = core.Usage{ContextSize: 200_000, ContextUsed: 50_000, ContextDetailErr: "分类明细加载失败"}
	})
}

func TestCtxPanelErrUI(t *testing.T) {
	a, tt := replayApp(t, demo.FixBug(false))
	a.st.Usage = core.Usage{ContextSize: 200_000, ContextUsed: 80_000, ContextDetailErr: "无法读取分类明细"}
	a.ctxLoading = false
	tt.Frame()
	if !tt.HasText("40%") {
		t.Fatal("应有上下文百分比")
	}
	if err := tt.Click("40%"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText("无法读取分类明细") {
		t.Fatal("err 态应展示错误文案")
	}
	if a.ctxState() != "err" {
		t.Fatalf("ctxState 应为 err,实际 %s", a.ctxState())
	}
}

func TestNoticeErrorRetry(t *testing.T) {
	a, tt, fb := pendingApp(t, demo.FixBug(false))
	a.st.Apply(core.EvTurnStarted{})
	a.st.Apply(core.EvAppend{Item: &core.Notice{ID: "n1", Level: core.NoticeError, Text: "请求失败 · 529 服务过载"}})
	tt.Frame()
	if !tt.HasText("重试") {
		t.Fatal("错误提示卡应有重试按钮")
	}
	a.st.Apply(core.EvTurnEnded{Reason: core.StopError})
	tt.Frame()
	if err := tt.Click("重试"); err != nil {
		t.Fatal(err)
	}
	if !isPrompt(fb.last(), a.userText(a.lastUserID()), false) {
		t.Fatalf("重试应重发最后用户消息:%v", fb.last())
	}
}

func TestTurnFoldLogic(t *testing.T) {
	steps := []core.Item{
		core.NewToolCall("r", "Read", "").WithInput(map[string]any{"file_path": "a.go"}),
		core.NewToolCall("f", "Bash", "").WithInput(map[string]any{"command": "go test"}),
	}
	if !turnFoldedItem(steps[0]) {
		t.Fatal("工具行应可折叠")
	}
	failSteps := []core.Item{
		core.NewToolCall("b", "Bash", "").WithInput(map[string]any{"command": "x"}),
	}
	failSteps[0].(*core.ToolCall).Status = core.ToolFailed
	if !turnMustStayOpen(failSteps) {
		t.Fatal("失败工具应保持展开")
	}
}

func TestSteerSyncHeuristic(t *testing.T) {
	q := &promptQueue{}
	now := time.Now()
	items := []core.Item{
		&core.User{ID: "u0", Text: "hello"},
	}
	q.addSteer("steer me", "steer me", 1, now.Add(-time.Second))
	q.syncSteers(items, true, now) // 尚无回显,保留
	if len(q.steers) != 1 {
		t.Fatal("未回显时不应收掉引导")
	}
	items = append(items, &core.User{ID: "u1", Text: "steer me"})
	items = append(items, &core.Assistant{ID: "a1", Text: "ok"})
	q.syncSteers(items, true, now.Add(2*steerDwell))
	if len(q.steers) != 0 {
		t.Fatal("新步骤出现后引导应视为已接收")
	}
}

func TestDockDraftRestore(t *testing.T) {
	a, _, _ := pendingApp(t, demo.PendingPermission())
	if len(a.pendingQueue()) == 0 {
		t.Fatal("应有待处理权限卡")
	}
	a.draft = "处理完继续写这段"
	a.dockStash = false
	a.dockDraft = ""
	a.syncDockComposer()
	if !a.dockStash || a.dockDraft != "处理完继续写这段" {
		t.Fatalf("草稿应暂存 stash=%v draft=%q", a.dockStash, a.dockDraft)
	}
	a.st.Items = nil
	a.syncDockComposer()
	if a.draft != "处理完继续写这段" || a.dockStash {
		t.Fatal("处理完应恢复草稿")
	}
}

type stubBE struct {
	fakeBE
	name string
}

func (s *stubBE) Name() string { return s.name }

func TestUserEditNeedsUUID(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	a.Attach(&stubBE{name: "claude"})
	a.st.Apply(core.EvAppend{Item: &core.User{ID: "u1", Text: "hello"}})
	a.userEd = userEdit{id: "u1", draft: "改掉"}
	a.submitUserEdit()
	if a.toast == "" {
		t.Fatal("真实后端无 uuid 时应提示无法从该处重发")
	}
	if _, ok := a.st.UserUUIDs["u1"]; ok && a.toast != "" {
		// toast 已覆盖
	}
}

func TestMermaidCard(t *testing.T) {
	_, tt := sessionApp(t, "mermaid-samples")
	if !tt.HasText("请求") && !tt.HasText("graph") {
		t.Fatal("Mermaid 场景应渲染流程图节点或源码")
	}
}

func TestTaskDockCapsules(t *testing.T) {
	_, tt := sessionApp(t, "c-task-dock")
	for _, label := range []string{"子代理", "后台命令", "监听", "工作流", "其他", "已结束"} {
		if !tt.HasText(label) {
			t.Fatalf("任务坞应包含胶囊 %q", label)
		}
	}
}

func TestQueueSteerRow(t *testing.T) {
	a, tt, fb := pendingApp(t, demo.RunningFixBug())
	tt.Frame()
	a.steerSend("立刻检查一下路由")
	tt.Frame()
	if !tt.HasText("等 Claude 接收") {
		t.Fatal("引导行应显示等待接收")
	}
	if !isPrompt(fb.last(), "立刻检查一下路由", true) {
		t.Fatalf("应发出引导:%v", fb.last())
	}
	a.st.Apply(core.EvAppend{Item: &core.User{ID: "su", Text: "立刻检查一下路由"}})
	a.st.Apply(core.EvAppend{Item: &core.Assistant{ID: "sa", Text: "好"}})
	if len(a.q.steers) > 0 {
		a.q.steers[0].at = time.Now().Add(-2 * steerDwell)
	}
	a.syncSteers()
	tt.Frame()
	if tt.HasText("等 Claude 接收") {
		t.Fatal("接收后引导行应收起")
	}
}

func TestAskStepBarJump(t *testing.T) {
	a, tt, _ := pendingApp(t, demo.PendingQuestion3())
	_ = tt.Click("范围")
	tt.Frame()
	if a.qa["q1"].cur != 1 {
		t.Fatalf("点步骤条应跳到第 2 题,cur=%d", a.qa["q1"].cur)
	}
}

// 确保 sinkBE 与 pendingApp 的 fakeBE 用途分明。
var _ backend.Backend = (*fakeBE)(nil)
