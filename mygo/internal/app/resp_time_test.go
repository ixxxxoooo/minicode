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
