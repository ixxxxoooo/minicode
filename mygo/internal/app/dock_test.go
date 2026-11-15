package app

import (
	"sync"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/backend"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/demo"
)

// fakeBE 记录界面发来的命令,不产出事件。
type fakeBE struct {
	mu   sync.Mutex
	cmds []core.Command
}

func (f *fakeBE) Name() string        { return "fake" }
func (f *fakeBE) Start(backend.Sink)  {}
func (f *fakeBE) Close()              {}
func (f *fakeBE) Send(c core.Command) { f.mu.Lock(); f.cmds = append(f.cmds, c); f.mu.Unlock() }
func (f *fakeBE) last() core.Command {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.cmds) == 0 {
		return nil
	}
	return f.cmds[len(f.cmds)-1]
}

// pendingApp 建一个停在某个待处理场景上的应用,命令交给 fakeBE。
func pendingApp(t *testing.T, steps []demo.Step) (*App, *ui.Tester, *fakeBE) {
	t.Helper()
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	a.SetProjects(demo.Projects(), "fixbug")
	fb := &fakeBE{}
	a.Attach(fb)
	demo.Replay(a.State(), steps)
	tt := ui.NewTester(a.View, 1280, 800)
	tt.Frame()
	return a, tt, fb
}

func TestPermissionCard(t *testing.T) {
	a, tt, fb := pendingApp(t, demo.PendingPermission())
	if _, ok := a.pendingDock().(*core.PermissionRequest); !ok {
		t.Fatal("应有待处理的权限请求")
	}
	if err := tt.Click("允许一次"); err != nil {
		t.Fatal(err)
	}
	want := core.CmdResolvePermission{ID: "b1", Option: "allow"}
	if got := fb.last(); got != want {
		t.Fatalf("点允许应发 %v,实际 %v", want, got)
	}
	// 键盘:Esc = 拒绝
	tt.Key(0, ui.KeyEscape)
	if got := fb.last(); got != (core.CmdResolvePermission{ID: "b1", Option: "deny"}) {
		t.Fatalf("Esc 应拒绝,实际 %v", got)
	}
	tt.Key(0, ui.KeyEnter)
	if got := fb.last(); got != want {
		t.Fatalf("回车应允许,实际 %v", got)
	}
}

func TestQuestionWizard(t *testing.T) {
	a, tt, fb := pendingApp(t, demo.PendingQuestion())
	if _, ok := a.pendingDock().(*core.QuestionRequest); !ok {
		t.Fatal("应有待处理的提问")
	}
	// 单选必须先选:没选时「下一题」不前进
	_ = tt.Click("下一题")
	tt.Frame()
	if a.qa["q1"].cur != 0 {
		t.Fatal("单选未选择不应前进")
	}
	if err := tt.Click("严格区分"); err != nil {
		t.Fatal(err)
	}
	_ = tt.Click("下一题")
	tt.Frame()
	if a.qa["q1"].cur != 1 {
		t.Fatalf("应进入第 2 题,cur=%d", a.qa["q1"].cur)
	}
	// 第 2 题多选:选两项后提交
	_ = tt.Click("加一个开关")
	_ = tt.Click("写进文档")
	if err := tt.Click("提交"); err != nil {
		t.Fatal(err)
	}
	got, ok := fb.last().(core.CmdAnswerQuestion)
	if !ok {
		t.Fatalf("应发出回答,实际 %v", fb.last())
	}
	if len(got.Answers) != 2 || got.Answers[0][0] != "严格区分" || len(got.Answers[1]) != 2 {
		t.Fatalf("回答不对:%v", got.Answers)
	}
}

func TestPlanCard(t *testing.T) {
	_, tt, fb := pendingApp(t, demo.PendingPlan())
	if err := tt.Click("自动模式"); err != nil {
		t.Fatal(err)
	}
	got, ok := fb.last().(core.CmdResolvePlan)
	if !ok || !got.Approve || got.Mode == nil || *got.Mode != core.ModeAuto {
		t.Fatalf("自动模式应批准并切到自动模式:%v", fb.last())
	}
	_ = tt.Click("继续规划")
	if got := fb.last().(core.CmdResolvePlan); got.Approve {
		t.Fatal("继续规划应拒绝")
	}
}

// 回合进行中 Enter 排队;正常结束后自动发送;中断则暂停。
func TestQueueFlow(t *testing.T) {
	a, tt, fb := pendingApp(t, demo.FixBug(false))
	a.st.Apply(core.EvTurnStarted{})
	tt.Frame()
	a.submitKey("再跑一遍测试", false)
	a.submitKey("更新 README", false)
	if a.q.len() != 2 {
		t.Fatalf("应排队 2 条,实际 %d", a.q.len())
	}
	a.submitKey("马上改一下", true)
	if got := fb.last(); !isPrompt(got, "马上改一下", true) {
		t.Fatalf("⌘Enter 应立即引导:%v", got)
	}
	// 正常结束:自动发出队首
	a.afterEventForTest(core.EvTurnEnded{Reason: core.StopEndTurn})
	if got := fb.last(); !isPrompt(got, "再跑一遍测试", false) || a.q.len() != 1 {
		t.Fatalf("结束后应自动发队首:%v 剩 %d", got, a.q.len())
	}
	// 再来一轮被中断:暂停,不自动发
	a.st.Apply(core.EvTurnStarted{})
	a.afterEventForTest(core.EvTurnEnded{Reason: core.StopCancelled})
	if !a.q.paused || a.q.len() != 1 {
		t.Fatalf("中断应暂停并保留队列:paused=%v len=%d", a.q.paused, a.q.len())
	}
	tt.Frame()
	if err := tt.Click("继续"); err != nil {
		t.Fatal(err)
	}
	if got := fb.last(); !isPrompt(got, "更新 README", false) {
		t.Fatalf("点继续应发出剩余排队:%v", got)
	}
}

// afterEventForTest 与真实路径一致:先归约再处理排队。
func (a *App) afterEventForTest(ev core.Event) {
	a.st.Apply(ev)
	a.afterEvent(ev)
}

func TestWelcomeAndNewSession(t *testing.T) {
	a, tt, _ := pendingApp(t, demo.FixBug(false))
	a.SetSessionOpener(func(id string) backend.Backend { return &fakeBE{} })
	a.NewSession()
	tt.Frame()
	if len(a.st.Items) != 0 {
		t.Fatal("新会话应为空")
	}
	// 起手卡把提示填进输入框
	if err := tt.Click("解释这个项目"); err != nil {
		t.Fatal(err)
	}
	if a.draft == "" {
		t.Fatal("点起手卡应填入提示")
	}
}

func isPrompt(c core.Command, text string, steer bool) bool {
	p, ok := c.(core.CmdPrompt)
	return ok && p.Text == text && p.Steer == steer
}
