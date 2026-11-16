package app

import (
	"testing"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// 用量页每帧都会调自动刷新:查询失败(limitsAt 仍为 0)时不能每帧重发,
// 否则状态在「更新中 / 查询失败」之间来回翻,整页闪烁。
func TestUsageLimitsAutoRefreshThrottled(t *testing.T) {
	a := New(Options{Mode: ModeAgent})
	a.demoExtra = false
	if a.cuDemo() {
		t.Skip("演示模式不走真实查询")
	}
	a.usageRefreshLimits(false)
	first := a.use.limitsTry
	if first.IsZero() || a.use.limitsState != core.LimitsChecking {
		t.Fatalf("第一次应发起查询: try=%v state=%v", first, a.use.limitsState)
	}
	a.use.limitsState = core.LimitsUnavailable // 模拟查询失败
	time.Sleep(5 * time.Millisecond)
	for i := 0; i < 20; i++ {
		a.usageRefreshLimits(false)
	}
	if a.use.limitsTry != first || a.use.limitsState != core.LimitsUnavailable {
		t.Fatalf("失败后自动路径不该重发: try 变了=%v state=%v", a.use.limitsTry != first, a.use.limitsState)
	}
	a.usageRefreshLimits(true) // 手动「重试」不受节流
	if a.use.limitsState != core.LimitsChecking {
		t.Fatal("手动重试应立即发起查询")
	}
}
