package app

import (
	"testing"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// 回合结束后脚注显示实测响应耗时(没有 token 统计时也显示),悬停明细写明各段。
func TestTurnFooterShowsResponseTime(t *testing.T) {
	a, tt := sessionApp(t, "ci")
	tt.SetSize(1280, 6000)
	tt.Frame()
	var uid string
	for _, it := range a.st.Items {
		if u, ok := it.(*core.User); ok {
			uid = u.ID
		}
	}
	if uid == "" {
		t.Fatal("场景里没有用户消息")
	}
	delete(a.st.TurnStats, uid)
	delete(a.st.Durations, uid)
	a.st.Apply(core.EvTurnTiming{ID: uid, Timing: core.TurnTiming{StartupMs: 1800, FirstMs: 3200, TotalMs: 18400}})
	tt.SetSize(1280, 6001) // 换个尺寸让行缓存失效
	tt.Frame()
	if !tt.HasText("用时 18 秒") {
		t.Fatal("回复下方应显示「用时 18 秒」")
	}
}

// 输入框里开始打字时预热后端:同一个后端只预热一次,空草稿 / 演示后端不预热。
func TestWarmIfTypingOncePerBackend(t *testing.T) {
	a, _ := quickApp(t)
	fb := &fakeBE{}
	a.Attach(fb)
	count := func() int {
		n := 0
		fb.mu.Lock()
		defer fb.mu.Unlock()
		for _, c := range fb.cmds {
			if _, ok := c.(core.CmdWarmup); ok {
				n++
			}
		}
		return n
	}
	a.draft = "   "
	a.warmIfTyping()
	if count() != 0 {
		t.Fatal("空草稿不应预热")
	}
	a.draft = "你好"
	a.warmIfTyping()
	a.warmIfTyping()
	if count() != 1 {
		t.Fatalf("同一个后端只应预热一次,实际 %d", count())
	}
	fb2 := &fakeBE{}
	a.Attach(fb2) // 换后端(切会话)后要重新预热
	a.warmIfTyping()
	fb2.mu.Lock()
	n := len(fb2.cmds)
	fb2.mu.Unlock()
	if n != 1 {
		t.Fatalf("新后端应再预热一次,实际 %d", n)
	}
}
