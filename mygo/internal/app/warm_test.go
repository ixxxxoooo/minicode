package app

import (
	"testing"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// claudeFake 假装是真实 claude 后端(预热策略只对 Name()=="claude" 生效)。
type claudeFake struct{ fakeBE }

func (f *claudeFake) Name() string { return "claude" }

func (f *claudeFake) count(pred func(core.Command) bool) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, c := range f.cmds {
		if pred(c) {
			n++
		}
	}
	return n
}
func (f *claudeFake) warms() int {
	return f.count(func(c core.Command) bool { _, ok := c.(core.CmdWarmup); return ok })
}
func (f *claudeFake) releases() int {
	return f.count(func(c core.Command) bool { _, ok := c.(core.CmdRelease); return ok })
}

// warmApp 造一个带可拨时钟的应用,后端是 claudeFake,窗口默认在前台。
func warmApp(t *testing.T) (*App, *claudeFake, *time.Time, *bool) {
	t.Helper()
	a, _ := quickApp(t)
	now := time.Date(2026, 11, 16, 15, 0, 0, 0, time.Local)
	a.SetClock(func() time.Time { return now })
	fg := true
	a.warm.fgHook = func() bool { return fg }
	fb := &claudeFake{}
	a.Attach(fb)
	return a, fb, &now, &fg
}

// 打开会话:去抖之前不预热,到点才预热,且只预热一次。
func TestWarmOnOpenAfterDebounce(t *testing.T) {
	a, fb, now, _ := warmApp(t)
	a.warmTick()
	if fb.warms() != 0 {
		t.Fatal("去抖时间内不应预热(快速翻会话列表不该起进程)")
	}
	*now = now.Add(warmDebounce + time.Millisecond)
	a.warmTick()
	a.warmTick()
	if fb.warms() != 1 {
		t.Fatalf("到点后应预热且只预热一次,实际 %d", fb.warms())
	}
}

// 快速切换:前一个会话还没到点就被换掉,不能给它起进程。
func TestWarmSkippedWhenSwitchedAwayQuickly(t *testing.T) {
	a, fb1, now, _ := warmApp(t)
	fb2 := &claudeFake{}
	a.Attach(fb2)
	*now = now.Add(time.Second)
	a.warmTick()
	if fb1.warms() != 0 {
		t.Fatal("已切走的会话不应预热")
	}
	if fb2.warms() != 1 {
		t.Fatalf("当前会话应预热一次,实际 %d", fb2.warms())
	}
}

// 窗口不在前台:一直等,回到前台才预热。
func TestWarmWaitsForForeground(t *testing.T) {
	a, fb, now, fg := warmApp(t)
	*fg = false
	a.warm.fgWas = false
	*now = now.Add(5 * time.Second)
	a.warmTick()
	if fb.warms() != 0 {
		t.Fatal("窗口在后台不应预热")
	}
	*fg = true
	a.warmTick() // 回前台:排期
	*now = now.Add(warmDebounce + time.Millisecond)
	a.warmTick()
	if fb.warms() != 1 {
		t.Fatalf("回前台后应预热,实际 %d", fb.warms())
	}
}

// 演示 / 测试后端(Name != claude)永远不预热、不释放。
func TestWarmIgnoresNonClaudeBackend(t *testing.T) {
	a, _ := quickApp(t)
	fb := &fakeBE{}
	a.Attach(fb)
	a.draft = "你好"
	a.warmTypingSignal()
	a.warmTick()
	if len(fb.cmds) != 0 {
		t.Fatalf("非 claude 后端不应收到任何预热 / 释放命令,实际 %v", fb.cmds)
	}
}

// 策略:输入时 = 打开不预热,打字才预热;不预热 = 打字也不预热。
func TestWarmPolicies(t *testing.T) {
	a, fb, now, _ := warmApp(t)
	a.prefs.Set(pWarm, 1)
	fb2 := &claudeFake{}
	a.Attach(fb2)
	*now = now.Add(time.Second)
	a.warmTick()
	if fb2.warms() != 0 {
		t.Fatal("「输入时」策略下打开会话不应预热")
	}
	a.draft = "写个函数"
	a.warmTypingSignal()
	a.warmTypingSignal()
	if fb2.warms() != 1 {
		t.Fatalf("开始打字应预热一次,实际 %d", fb2.warms())
	}

	a.prefs.Set(pWarm, 2)
	fb3 := &claudeFake{}
	a.Attach(fb3)
	*now = now.Add(time.Second)
	a.draft = "再打一段"
	a.warmTypingSignal()
	a.warmTick()
	if fb3.warms() != 0 {
		t.Fatal("「不预热」策略下不应预热")
	}
	_ = fb
}

// 空闲释放:闲置超时才释放;回合中 / 有排队 / 有后台任务 / 窗口在后台时间缩短。
func TestIdleRelease(t *testing.T) {
	a, fb, now, fg := warmApp(t)
	a.st.Apply(core.EvConn{State: core.ConnReady})
	a.st.Phase = core.PhaseIdle
	a.warm.lastAct = *now
	a.warm.sent = true

	*now = now.Add(9 * time.Minute)
	a.warmTick()
	if fb.releases() != 0 {
		t.Fatal("默认 10 分钟,9 分钟不应释放")
	}

	// 回合进行中不释放
	a.st.Phase = core.PhaseRunning
	*now = now.Add(5 * time.Minute)
	a.warmTick()
	if fb.releases() != 0 {
		t.Fatal("回合进行中不应释放")
	}
	a.st.Phase = core.PhaseIdle

	// 有在跑的后台任务不释放
	a.st.Tasks = []*core.BackgroundTask{{TaskID: "t1", Status: core.TaskRunning, Backgrounded: true}}
	a.warmTick()
	if fb.releases() != 0 {
		t.Fatal("有后台任务时不应释放")
	}
	a.st.Tasks = nil

	a.warmTick()
	if fb.releases() != 1 {
		t.Fatalf("闲置超时应释放一次,实际 %d", fb.releases())
	}
	// 60 秒内不重复发
	*now = now.Add(10 * time.Second)
	a.warmTick()
	if fb.releases() != 1 {
		t.Fatal("拒绝回收后 60 秒内不应重复发送")
	}

	// 窗口在后台:限时 ÷3(10 分钟 → 3 分 20 秒)
	fb2 := &claudeFake{}
	a.Attach(fb2)
	a.st.Apply(core.EvConn{State: core.ConnReady})
	a.st.Phase = core.PhaseIdle
	a.warm.lastAct, a.warm.sent = *now, true
	*fg = false
	a.warm.fgWas = false
	*now = now.Add(4 * time.Minute)
	a.warmTick()
	if fb2.releases() != 1 {
		t.Fatalf("窗口在后台 4 分钟(> 3 分 20 秒)应释放,实际 %d", fb2.releases())
	}
}

// 「不释放」策略永不回收;释放后再打字会重新预热。
func TestNeverReleaseAndRewarmAfterRelease(t *testing.T) {
	a, fb, now, _ := warmApp(t)
	a.prefs.Set(pIdle, 3)
	a.st.Apply(core.EvConn{State: core.ConnReady})
	a.st.Phase = core.PhaseIdle
	a.warm.lastAct, a.warm.sent = *now, true
	*now = now.Add(5 * time.Hour)
	a.warmTick()
	if fb.releases() != 0 {
		t.Fatal("「不释放」不应回收")
	}

	a.prefs.Set(pIdle, 1)
	a.warmTick()
	if fb.releases() != 1 {
		t.Fatalf("改回 10 分钟应释放,实际 %d", fb.releases())
	}
	// 后端回报已释放 → 再打字重新预热
	a.afterEvent(core.EvConn{State: core.ConnNone})
	a.st.Apply(core.EvConn{State: core.ConnNone})
	before := fb.warms()
	a.draft = "我回来了"
	a.warmTypingSignal()
	if fb.warms() != before+1 {
		t.Fatalf("释放后打字应重新预热,实际 %d → %d", before, fb.warms())
	}
}
