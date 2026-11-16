package app

import (
	"sync"
	"testing"
	"time"

	"github.com/ixxxxoooo/minicode/internal/backend"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/demo"
	"github.com/ixxxxoooo/minicode/internal/diag"
)

// gatedBE 在 release 关闭前不往界面送事件(模拟历史回放较慢)。
type gatedBE struct {
	evs     []core.Event
	release chan struct{}
	once    sync.Once
}

func (g *gatedBE) Name() string { return "gated" }
func (g *gatedBE) Close()       {}
func (g *gatedBE) Send(core.Command) {
}
func (g *gatedBE) Start(sink backend.Sink) {
	go func() {
		defer diag.Recover("gatedBE")
		<-g.release
		for _, e := range g.evs {
			sink(e)
		}
	}()
}

func newGatedBE(evs []core.Event) *gatedBE {
	return &gatedBE{evs: evs, release: make(chan struct{})}
}

func (g *gatedBE) unlock() {
	g.once.Do(func() { close(g.release) })
}

func TestOpenSessionHoldContentNotWelcome(t *testing.T) {
	a, tt, _ := pendingApp(t, demo.FixBug(false))
	if !tt.HasText("cargo") && !tt.HasText("路由") {
		t.Fatal("预置会话应有可见内容")
	}
	gate := newGatedBE([]core.Event{
		core.EvAppend{Item: &core.Assistant{ID: "a2", Text: "另一条会话"}},
		core.EvReady{SessionID: "s-other", Models: core.DefaultModels(), Model: "sonnet", Mode: core.ModeDefault},
	})
	a.SetSessionOpener(func(id string) backend.Backend { return gate })
	a.OpenSession("s-other")
	tt.Frame()
	if tt.HasText("解释这个项目") || tt.HasText("今天想在") {
		t.Fatal("切换已有会话时不应闪现欢迎页")
	}
	if a.sessLoad.active && !tt.HasText("正在载入会话…") && !tt.HasText("路由") {
		t.Fatal("载入中应显示骨架或暂留上一会话内容")
	}
	gate.unlock()
	time.Sleep(30 * time.Millisecond)
	tt.Frame()
	if !tt.HasText("另一条会话") {
		t.Fatal("回放完成后应显示目标会话")
	}
	if a.sessLoad.active {
		t.Fatal("EvReady 后应结束载入态")
	}
}

func TestOpenSessionStaleReplayDropped(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	a.SetProjects(demo.Projects(), "fixbug")
	var slow, fast *gatedBE
	a.SetSessionOpener(func(id string) backend.Backend {
		if id == "slow" {
			slow = newGatedBE([]core.Event{
				core.EvAppend{Item: &core.Assistant{ID: "slow", Text: "来自慢会话"}},
				core.EvReady{SessionID: "slow", Models: core.DefaultModels(), Model: "sonnet", Mode: core.ModeDefault},
			})
			return slow
		}
		fast = newGatedBE([]core.Event{
			core.EvAppend{Item: &core.Assistant{ID: "fast", Text: "来自快会话"}},
			core.EvReady{SessionID: "fast", Models: core.DefaultModels(), Model: "sonnet", Mode: core.ModeDefault},
		})
		return fast
	})
	a.OpenSession("slow")
	a.OpenSession("fast")
	fast.unlock()
	time.Sleep(20 * time.Millisecond)
	slow.unlock()
	time.Sleep(30 * time.Millisecond)
	for _, item := range a.st.Items {
		if asst, ok := item.(*core.Assistant); ok && asst.Text == "来自慢会话" {
			t.Fatal("迟到的慢会话回放不应覆盖当前会话")
		}
	}
	got := ""
	for _, item := range a.st.Items {
		if asst, ok := item.(*core.Assistant); ok {
			got = asst.Text
		}
	}
	if got != "来自快会话" {
		t.Fatalf("当前会话内容=%q", got)
	}
}

func TestNewSessionStillWelcome(t *testing.T) {
	a, tt, _ := pendingApp(t, demo.FixBug(false))
	a.NewSession()
	tt.Frame()
	if !tt.HasText("解释这个项目") {
		t.Fatal("新会话仍应显示欢迎页起手卡")
	}
}

// 确保 gatedBE 满足 backend.Backend(与其它测试里的 fakeBE 并列)。
var _ backend.Backend = (*gatedBE)(nil)

func TestChatWelcomeLogic(t *testing.T) {
	a := New(Options{})
	a.curSess = ""
	a.st = core.NewState()
	if !a.showChatWelcome() {
		t.Fatal("无 id 空状态应变欢迎页")
	}
	a.beginSessLoad("real-id")
	if a.showChatWelcome() {
		t.Fatal("载入中不应变欢迎页")
	}
	a.finishSessLoad()
	a.curSess = "real-id"
	if a.showChatWelcome() {
		t.Fatal("有 id 无条目不应回落欢迎页")
	}
}
