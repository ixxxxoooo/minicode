package claude

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// connStates 按顺序取出事件流里的连接状态。
func connStates(r *rig) []core.ConnState {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []core.ConnState
	for _, e := range r.evs {
		if c, ok := e.(core.EvConn); ok {
			out = append(out, c.State)
		}
	}
	return out
}

func timingOf(r *rig) core.TurnTiming {
	r.mu.Lock()
	defer r.mu.Unlock()
	var tm core.TurnTiming
	for _, e := range r.evs {
		if x, ok := e.(core.EvTurnTiming); ok {
			tm = x.Timing
		}
	}
	return tm
}

// 预热:拉起进程并握手,期间不出现「回合开始」;之后发送不再计启动耗时。
func TestWarmupReportsConnAndSkipsStartup(t *testing.T) {
	r := newRig(t, Config{})
	if r.st.Conn != core.ConnNone {
		t.Fatalf("惰性后端一开始应是未连接:%v", r.st.Conn)
	}
	r.b.Send(core.CmdWarmup{})
	r.wait("握手完成", func() bool { return r.st.Conn == core.ConnReady })
	r.mu.Lock()
	started := r.hasEventLocked(func(e core.Event) bool { _, ok := e.(core.EvTurnStarted); return ok })
	r.mu.Unlock()
	if started {
		t.Fatal("预热不应产生回合")
	}
	r.b.Send(core.CmdPrompt{Text: "hi"})
	r.wait("回合结束", func() bool { return turnsEnded(r) == 1 })
	if tm := timingOf(r); tm.StartupMs != 0 {
		t.Fatalf("预热后发送不应再计启动耗时:%+v", tm)
	}
	got := connStates(r)
	if len(got) < 2 || got[0] != core.ConnStarting || got[1] != core.ConnReady {
		t.Fatalf("连接状态应依次是 连接中 → 已连接:%v", got)
	}
}

// 空闲回收:进程被结束、状态回到未连接且不弹「进程已重新启动」;再发消息自动重连并接着聊。
func TestReleaseThenReconnectOnSend(t *testing.T) {
	r := newRig(t, Config{})
	r.b.Send(core.CmdWarmup{})
	r.wait("握手完成", func() bool { return r.st.Conn == core.ConnReady })
	// 握手一完成后端会自己发一次「上下文用量」请求;还有未完成的请求时回收会被拒绝(不能打断它们),
	// 所以先等这个请求答完(出现 EvUsage),再回收,避免和它抢时序。
	r.wait("握手后的用量请求完成", func() bool {
		return r.hasEventLocked(func(e core.Event) bool { _, ok := e.(core.EvUsage); return ok })
	})
	r.b.Send(core.CmdRelease{Why: "测试"})
	r.wait("回收后未连接", func() bool { return r.st.Conn == core.ConnNone })

	r.b.Send(core.CmdPrompt{Text: "hi"})
	r.wait("重连后回合结束", func() bool { return turnsEnded(r) == 1 })
	if tm := timingOf(r); tm.StartupMs <= 0 {
		t.Fatalf("回收后的第一条消息要重新启动,应计启动耗时:%+v", tm)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range r.evs {
		if n, ok := e.(core.EvNotice); ok && strings.Contains(n.Text, "重新启动") {
			t.Fatalf("空闲回收后的重连不应提示「进程已重新启动」:%s", n.Text)
		}
	}
	if r.st.Conn != core.ConnReady {
		t.Fatalf("发送后应已连接:%v", r.st.Conn)
	}
}

// 回合进行中(权限卡未处理)不能被回收,否则会把正在做的事打断。
func TestReleaseRefusedWhileTurnActive(t *testing.T) {
	r := newRig(t, Config{})
	r.b.Send(core.CmdPrompt{Text: "tool"}) // 假 claude 在这里停在权限请求上
	r.wait("出现权限请求", func() bool {
		for _, it := range r.st.Items {
			if _, ok := it.(*core.PermissionRequest); ok {
				return true
			}
		}
		return false
	})
	r.b.Send(core.CmdRelease{Why: "测试"})
	r.b.Send(core.CmdWarmup{}) // 同一个命令队列里的哨兵:它被处理就说明前面的回收命令已处理完
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.st.Conn == core.ConnNone {
		t.Fatalf("回合进行中不应被回收:%v", connStatesLocked(r))
	}
}

func connStatesLocked(r *rig) []core.ConnState {
	var out []core.ConnState
	for _, e := range r.evs {
		if c, ok := e.(core.EvConn); ok {
			out = append(out, c.State)
		}
	}
	return out
}

// 预热失败:状态为「连接失败」;60 秒内再预热不会反复尝试,发送时仍会再试一次并给出原因。
func TestWarmupFailureBackoff(t *testing.T) {
	t.Setenv("PATH", "/nonexistent")
	t.Setenv("MINI_CLAUDE_PATH", "")
	t.Setenv("HOME", t.TempDir()) // 让「常见目录里找 claude」也找不到
	var mu sync.Mutex
	var evs []core.Event
	b := New(Config{Cwd: t.TempDir(), ClaudePath: "/nonexistent/claude"})
	b.Start(func(e core.Event) { mu.Lock(); evs = append(evs, e); mu.Unlock() })
	defer b.Close()
	failed := func() int {
		mu.Lock()
		defer mu.Unlock()
		n := 0
		for _, e := range evs {
			if c, ok := e.(core.EvConn); ok && c.State == core.ConnFailed {
				n++
			}
		}
		return n
	}
	b.Send(core.CmdWarmup{})
	deadline := time.Now().Add(5 * time.Second)
	for failed() == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if failed() != 1 {
		t.Fatalf("预热失败应报告一次「连接失败」,实际 %d", failed())
	}
	for i := 0; i < 3; i++ {
		b.Send(core.CmdWarmup{})
	}
	time.Sleep(200 * time.Millisecond) // 命令按序处理,留点时间让上面三条跑完
	if failed() != 1 {
		t.Fatalf("退避期内不应再次尝试预热,实际失败事件 %d 个", failed())
	}
	b.Send(core.CmdPrompt{Text: "hi"}) // 真正发送不受退避影响:再试一次并结束回合、给出原因
	deadline = time.Now().Add(5 * time.Second)
	for failed() < 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if failed() != 2 {
		t.Fatalf("发送时应再试一次(失败事件 2 个),实际 %d", failed())
	}
}
