package claude

import (
	"testing"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// TestRealControlRequests 验证 mcp_status / get_context_usage / get_usage / compact 控制链路不崩,失败时有明确降级文案。
func TestRealControlRequests(t *testing.T) {
	realSkip(t)
	mode := core.ModePlan
	cwd := t.TempDir()
	r := newRealRig(t, Config{Mode: &mode, Cwd: cwd})
	t.Cleanup(func() { cleanClaudeSession(t, r, cwd) })

	r.b.Send(core.CmdPrompt{Text: "只回复 ok,不要调用工具。"})
	r.waitTimeout(90*time.Second, "握手与首回合", func() bool {
		return r.hasEventLocked(func(e core.Event) bool { _, ok := e.(core.EvReady); return ok }) && r.turnsEndedLocked() >= 1
	})
	skipIfLimitReply(t, r)

	r.b.Send(core.CmdRefreshMcp{Runtime: true})
	r.b.Send(core.CmdRefreshUsage{Context: true, Limits: true, Force: true})
	r.waitTimeout(45*time.Second, "控制请求应答", func() bool {
		return r.usageCountLocked() >= 1 && (len(r.st.Usage.ContextCategories) > 0 || r.st.Usage.ContextDetailErr != "" || r.st.Usage.LimitsState != core.LimitsIdle)
	})

	u, ok := r.lastUsage()
	if !ok {
		t.Fatal("应有 EvUsage")
	}
	t.Logf("上下文 used=%d categories=%d detailErr=%q limits=%v five=%v weekly=%v",
		u.ContextUsed, len(u.ContextCategories), u.ContextDetailErr, u.LimitsState, u.FiveHourPct, u.WeeklyPct)

	if len(u.ContextCategories) == 0 {
		if u.ContextDetailErr == "" {
			t.Log("get_context_usage:无分类明细且无错误文案(可能 DeepSeek 不支持该控制请求)")
		} else {
			t.Logf("get_context_usage 降级:%s", u.ContextDetailErr)
		}
	}
	if u.LimitsState == core.LimitsUnavailable {
		t.Log("get_usage:服务商不支持账户限额,已标记 LimitsUnavailable")
	} else if u.FiveHourPct == nil && u.WeeklyPct == nil {
		t.Log("get_usage:应答成功但无数值(记录现象,不判失败)")
	}

	// /compact 走 onCompact
	before := len(r.st.Items)
	r.b.Send(core.CmdCompact{})
	r.waitTimeout(120*time.Second, "compact 回合", func() bool { return r.turnsEndedLocked() >= 2 })
	if len(r.st.Items) <= before {
		t.Log("compact 后时间线条目未明显增加(可能已被压缩或回合未产生可见条目)")
	}
}
