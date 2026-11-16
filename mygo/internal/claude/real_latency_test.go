package claude

import (
	"testing"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// TestRealLatencyColdVsWarm 对比「冷启动后直接发送」与「先预热再发送」的耗时分段(会产生极少量 token 消耗)。
// 运行:cd mygo && MINI_REAL_CLAUDE=1 ../scripts/with-deepseek.sh go test ./internal/claude -run RealLatency -v
func TestRealLatencyColdVsWarm(t *testing.T) {
	realSkip(t)
	mode := core.ModePlan
	for _, warm := range []bool{false, true} {
		name := map[bool]string{false: "冷启动", true: "预热后"}[warm]
		r, _ := realRigWithCleanup(t, Config{Mode: &mode})
		if warm {
			r.b.Send(core.CmdWarmup{})
			r.waitTimeout(60*time.Second, "握手完成", func() bool { return r.st.SessionID != "" && r.st.Phase == core.PhaseIdle })
			time.Sleep(300 * time.Millisecond)
		}
		t0 := time.Now()
		r.b.Send(core.CmdPrompt{Text: "只回复两个字母:ok,不要调用任何工具。"})
		r.waitTimeout(120*time.Second, "回合结束", func() bool { return r.turnsEndedLocked() == 1 })
		wall := time.Since(t0)
		r.mu.Lock()
		var tm core.TurnTiming
		got := false
		for _, e := range r.evs {
			if x, ok := e.(core.EvTurnTiming); ok {
				tm, got = x.Timing, true
			}
		}
		r.mu.Unlock()
		skipIfLimitReply(t, r)
		if !got {
			t.Fatalf("%s:没有收到 EvTurnTiming", name)
		}
		t.Logf("%s:墙钟 %v | 后端实测 启动握手 %dms · 首响应 %dms · 总 %dms", name, wall.Round(time.Millisecond), tm.StartupMs, tm.FirstMs, tm.TotalMs)
		if tm.TotalMs <= 0 || tm.FirstMs <= 0 || tm.FirstMs > tm.TotalMs {
			t.Errorf("%s:耗时数据不合理 %+v", name, tm)
		}
		if warm && tm.StartupMs != 0 {
			t.Errorf("预热后发送不应再计启动耗时,得到 %dms", tm.StartupMs)
		}
		if !warm && tm.StartupMs <= 0 {
			t.Errorf("冷启动应记录启动握手耗时,得到 %dms", tm.StartupMs)
		}
		r.b.Close()
	}
}
