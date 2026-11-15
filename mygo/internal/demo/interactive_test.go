package demo

import (
	"sync"
	"testing"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// 演示后端端到端:发含「权限」的提示 → 停在权限卡 → 处理后回合正常结束。
func TestInteractivePermission(t *testing.T) {
	var mu sync.Mutex
	st := core.NewState()
	b := New(func() []Step {
		return []Step{{0, core.EvReady{SessionID: "t", Models: core.DefaultModels(), Model: "sonnet"}}}
	}, 0.01)
	b.Start(func(ev core.Event) { mu.Lock(); st.Apply(ev); mu.Unlock() })
	read := func(f func() bool) bool {
		for i := 0; i < 300; i++ {
			mu.Lock()
			ok := f()
			mu.Unlock()
			if ok {
				return true
			}
			time.Sleep(10 * time.Millisecond)
		}
		return false
	}
	b.Send(core.CmdPrompt{Text: "请求权限"})
	if !read(func() bool { return st.AwaitingUser() }) {
		t.Fatal("应停在权限卡上")
	}
	var id string
	mu.Lock()
	for _, it := range st.Items {
		if p, ok := it.(*core.PermissionRequest); ok {
			id = p.ID
		}
	}
	mu.Unlock()
	b.Send(core.CmdResolvePermission{ID: id, Option: "allow"})
	if !read(func() bool { return st.Phase == core.PhaseIdle && !st.AwaitingUser() && st.LastStop != nil }) {
		t.Fatal("处理后回合应结束")
	}
	b.Close()
}

// 取消:回合以「已中断」结束,待处理的卡片不再续播。
func TestInteractiveCancel(t *testing.T) {
	var mu sync.Mutex
	st := core.NewState()
	b := New(func() []Step { return nil }, 0.01)
	b.Start(func(ev core.Event) { mu.Lock(); st.Apply(ev); mu.Unlock() })
	b.Send(core.CmdPrompt{Text: "提问一下"})
	time.Sleep(200 * time.Millisecond)
	b.Send(core.CmdCancel{})
	time.Sleep(100 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	if st.LastStop == nil || *st.LastStop != core.StopCancelled {
		t.Fatalf("应以已中断结束:%v", st.LastStop)
	}
	b.Close()
}
