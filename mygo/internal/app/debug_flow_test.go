package app

import (
	"testing"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// 回归:截图 / 测试会把 a.now() 注入成固定的「昨天」,SeedSteers 登记的引导不能因此被
// 20 秒兜底超时立刻放出(以前用 a.now() 登记,syncSteers 用真实时钟,差了一天)。
func TestSeedSteersSurvivesInjectedClock(t *testing.T) {
	a, _ := newTestApp(t)
	a.SetClock(func() time.Time { return time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC) })
	a.st.Phase = core.PhaseRunning
	a.SeedSteers("先只看 normalize 行为", "README 里补一句说明")
	a.syncSteers()
	if got := len(a.q.steers); got != 2 {
		t.Fatalf("引导应还留在待发送区,实际 %d 条", got)
	}
}
