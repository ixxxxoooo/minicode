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
	a.sessProcLive = false
	a.ctxEnsureLoaded()
	if len(fb.cmds) != 0 {
		t.Fatalf("进程未启动不应发命令, got %d", len(fb.cmds))
	}
	if a.ctxLoading {
		t.Fatal("进程未启动不应处于刷新中")
	}
}
