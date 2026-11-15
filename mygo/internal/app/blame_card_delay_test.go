package app

import (
	"testing"
	"time"
)

// 追溯提交卡:鼠标离开后先保持 250ms(够移进卡片),到点才收起;重置会清掉计时。
func TestBlameCardCloseDelay(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	if !a.blameKeepOpen() {
		t.Fatal("刚离开时应继续保持显示")
	}
	if a.ide.blameLeaveAt.IsZero() {
		t.Fatal("应记下离开时刻")
	}
	a.ide.blameLeaveAt = time.Now().Add(-blameCloseDelay - 10*time.Millisecond)
	if a.blameKeepOpen() {
		t.Fatal("超过延迟后应收起")
	}
	a.ide.blameCardShow = true
	a.resetBlameCard()
	if a.ide.blameCardShow || !a.ide.blameLeaveAt.IsZero() {
		t.Fatal("重置应清掉显示与计时")
	}
}
