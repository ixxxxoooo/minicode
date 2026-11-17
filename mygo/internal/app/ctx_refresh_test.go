package app

import (
	"testing"
	"time"

	"github.com/ixxxxoooo/minicode/internal/claudecfg"
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
	var ctxCmds int
	for _, c := range fb.cmds {
		if u, ok := c.(core.CmdRefreshUsage); ok && u.Context {
			ctxCmds++
		}
	}
	if ctxCmds != 1 {
		t.Fatalf("应发一次上下文查询, got %d cmds %v", ctxCmds, fb.cmds)
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
	realReg(a, t)
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
	realReg(a, t)
	fb := &claudeFakeBE{}
	a.Attach(fb)
	a.sessProcLive = true
	pv := a.pvLoad()
	pv.store = claudecfg.Store{
		Providers: []claudecfg.Provider{{ID: "ds", Name: "DS", BaseURL: "https://x", Models: []string{"m1"}}},
		Mapping:   claudecfg.Mapping{Enabled: true, Provider: "ds", Default: "m1"},
	}
	pv.keys = map[string]string{"ds": "k"} // 映射要有密钥才算真正启用
	fb.cmds = nil
	a.ctxRefreshAt = time.Time{}
	a.ctxEnsureLoaded()
	for _, c := range fb.cmds {
		if u, ok := c.(core.CmdRefreshUsage); ok && u.Limits {
			t.Fatalf("第三方不应自动查限额: %v", c)
		}
	}
}
