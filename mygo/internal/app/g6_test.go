package app

import (
	"sync"
	"testing"

	"github.com/ixxxxoooo/minicode/internal/backend"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/demo"
)

// sinkBE 记录收到的命令与是否被关闭,并保留 sink 让测试能「从后端推事件」。
type sinkBE struct {
	mu     sync.Mutex
	sink   backend.Sink
	closed bool
	cmds   []core.Command
}

func (s *sinkBE) Name() string         { return "fake" }
func (s *sinkBE) Start(k backend.Sink) { s.sink = k }
func (s *sinkBE) Close()               { s.mu.Lock(); s.closed = true; s.mu.Unlock() }
func (s *sinkBE) Send(c core.Command)  { s.mu.Lock(); s.cmds = append(s.cmds, c); s.mu.Unlock() }
func (s *sinkBE) isClosed() bool       { s.mu.Lock(); defer s.mu.Unlock(); return s.closed }

func TestBackgroundSessionKeepAlive(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	a.SetProjects(demo.Projects(), "fixbug")
	be := &sinkBE{}
	a.Attach(be)
	a.curSess = "s1"
	be.sink(core.EvTurnStarted{})
	if a.st.Phase != core.PhaseRunning {
		t.Fatal("回合应在进行")
	}
	a.OpenSession("s2")
	if got := a.BackgroundSessions(); len(got) != 1 || got[0] != "s1" || be.isClosed() {
		t.Fatalf("切走时回合进行中的会话应转入后台、后端不关:%v closed=%v", got, be.isClosed())
	}
	// 后台继续收事件,不影响当前界面的状态
	cur := a.st
	be.sink(core.EvTurnEnded{Reason: core.StopEndTurn})
	if a.st != cur {
		t.Fatal("后台事件不应改动当前会话的状态")
	}
	if len(a.BackgroundSessions()) != 0 || !be.isClosed() {
		t.Fatal("后台回合结束后应释放后端")
	}

	// 回到仍在跑的后台会话:接回同一个后端与状态
	be2 := &sinkBE{}
	a.Attach(be2)
	a.curSess = "s3"
	be2.sink(core.EvTurnStarted{})
	st3 := a.st
	a.OpenSession("s4")
	a.OpenSession("s3")
	if a.be != be2 || a.st != st3 || a.curSess != "s3" {
		t.Fatal("回到后台会话应接回原后端与状态")
	}
}

func TestPersistModelAndEffort(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	be := &sinkBE{}
	a.Attach(be)
	a.st.Models = core.DefaultModels()
	a.pickModel("opus")
	a.pickEffort(core.EffortHigh)
	if a.prefs.Str(prefModel, "") != "opus" || a.prefs.Int(prefEffort, -1) != int(core.EffortHigh) {
		t.Fatal("选择应写进偏好")
	}
	// 新会话握手后沿用
	be2 := &sinkBE{}
	a.opener = func(string) backend.Backend { return be2 }
	a.NewSession()
	be2.sink(core.EvReady{SessionID: "x", Models: core.DefaultModels(), Model: "sonnet", Mode: core.ModeDefault})
	if a.st.Model != "opus" || a.st.Effort != core.EffortHigh {
		t.Fatalf("新会话应沿用 opus / high:%s %v", a.st.Model, a.st.Effort)
	}
	if len(be2.cmds) < 2 {
		t.Fatalf("应向后端发了模型与强度命令:%v", be2.cmds)
	}
}

func TestNewSessionInProject(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	var got string
	a.opener = func(id string) backend.Backend { got = id; return &sinkBE{} }
	a.NewSessionIn(core.ProjectInfo{ID: "p1", Name: "demo", Path: "/tmp/demo"})
	if got != "new:/tmp/demo" || a.curSess != "" {
		t.Fatalf("项目内新会话应带目录且没有会话 id:%q %q", got, a.curSess)
	}
}

func TestTerminalTabs(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	a.ide.tpc() // 第一个标签
	a.newTerm()
	a.newTerm()
	if len(a.ide.tps) != 3 || a.ide.tpSel != 2 {
		t.Fatalf("应有 3 个终端标签并选中最后一个:%d / %d", len(a.ide.tps), a.ide.tpSel)
	}
	a.closeTermTab(2)
	if len(a.ide.tps) != 2 || a.ide.tpSel != 1 {
		t.Fatalf("关掉最后一个后应选中前一个:%d / %d", len(a.ide.tps), a.ide.tpSel)
	}
	a.closeTermTab(0)
	a.closeTermTab(0)
	if len(a.ide.tps) != 1 || a.ide.tps[0].t != nil {
		t.Fatalf("全部关掉后应留一个空白标签:%d", len(a.ide.tps))
	}
}
