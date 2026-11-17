package app

import (
	"testing"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

func TestCtxDebounceNoDoubleCmd(t *testing.T) {
	a := New(Options{})
	fb := &claudeFakeBE{}
	a.Attach(fb)
	fb.cmds = nil         // 打开会话时的预热命令不算
	a.sessProcLive = true // 进程已启动,点开才会发查询
	a.ctxRefreshAt = time.Now()
	a.ctxEnsureLoaded()
	if len(fb.cmds) != 0 {
		t.Fatalf("防抖期内不应发命令, got %d", len(fb.cmds))
	}
	a.ctxRefreshAt = time.Time{}
	a.ctxEnsureLoaded()
	if len(fb.cmds) != 1 {
		t.Fatalf("应发一次命令, got %d", len(fb.cmds))
	}
	last := fb.cmds[0]
	u, ok := last.(core.CmdRefreshUsage)
	if !ok || !u.Context {
		t.Fatalf("命令不对:%T", last)
	}
}

type claudeFakeBE struct{ fakeBE }

func (*claudeFakeBE) Name() string { return "claude" }

// 进程没启动时点开面板:不发命令,也不能一直挂着「刷新中」。
func TestCtxNoProcNotLoading(t *testing.T) {
	a := New(Options{})
	fb := &claudeFakeBE{}
	a.Attach(fb)
	fb.cmds = nil // 打开会话时的预热命令不算
	a.sessProcLive = false
	a.ctxEnsureLoaded()
	if len(fb.cmds) != 0 {
		t.Fatalf("进程未启动不应发命令, got %d", len(fb.cmds))
	}
	if a.ctxLoading {
		t.Fatal("进程未启动不应处于刷新中")
	}
}

func TestCtxOpenRefreshesLimitsWhenOfficial(t *testing.T) {
	a := New(Options{})
	fb := &claudeFakeBE{}
	a.Attach(fb)
	a.sessProcLive = true
	fb.cmds = nil
	a.ctxRefreshAt = time.Time{}
	a.ctxEnsureLoaded()
	var lim int
	for _, c := range fb.cmds {
		if u, ok := c.(core.CmdRefreshUsage); ok && u.Limits && u.LimitsQuick {
			lim++
		}
	}
	if lim != 1 {
		t.Fatalf("官方账号点开应发一次限额查询, got %d cmds %v", lim, fb.cmds)
	}
}

func TestCtxOpenSkipsLimitsForThirdParty(t *testing.T) {
	a := New(Options{})
	fb := &claudeFakeBE{}
	a.Attach(fb)
	a.sessProcLive = true
	a.pvLoad().store.Activate("deepseek", "m1")
	fb.cmds = nil
	a.ctxRefreshAt = time.Time{}
	a.ctxEnsureLoaded()
	for _, c := range fb.cmds {
		if u, ok := c.(core.CmdRefreshUsage); ok && u.Limits {
			t.Fatalf("第三方不应自动查限额: %v", c)
		}
	}
}
