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

// 进程没启动且还没有明细时点开面板:只查一次明细并顺带启动进程(历史会话惰性启动)。
func TestCtxNoProcStartsAndQueriesOnce(t *testing.T) {
	a := New(Options{})
	fb := &claudeFakeBE{}
	a.Attach(fb)
	fb.cmds = nil // 打开会话时的预热命令不算
	a.sessProcLive = false
	a.ctxEnsureLoaded()
	var n int
	for _, c := range fb.cmds {
		if u, ok := c.(core.CmdRefreshUsage); ok && u.Context && u.StartProc {
			n++
		}
	}
	if n != 1 || !a.ctxLoading {
		t.Fatalf("应发一次带启动进程的明细查询并处于刷新中, got %d loading=%v", n, a.ctxLoading)
	}
}

// 明细已经有了(后端在会话就绪 / 回合结束时自动取):点开弹层不再重查明细。
func TestCtxOpenSkipsDetailWhenLoaded(t *testing.T) {
	a := New(Options{})
	fb := &claudeFakeBE{}
	a.Attach(fb)
	a.sessProcLive = true
	a.st.Usage.ContextCategories = []core.ContextCategory{{Name: "System prompt", Tokens: 2600}}
	fb.cmds = nil
	a.ctxRefreshAt = time.Time{}
	a.ctxEnsureLoaded()
	for _, c := range fb.cmds {
		if u, ok := c.(core.CmdRefreshUsage); ok && u.Context {
			t.Fatalf("已有明细,点开不应再查明细: %v", c)
		}
	}
	if a.ctxLoading {
		t.Fatal("没发明细查询,不应处于刷新中")
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
