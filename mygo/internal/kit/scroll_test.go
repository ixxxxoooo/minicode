package kit

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

func TestHorizontalWheelOnHScroll(t *testing.T) {
	Use(theme.Resolve(theme.Palettes[0], theme.Schemes[0], ""))
	SetScrollbarMode(ScrollbarFollowSystem)
	var st ui.ScrollState
	tt := ui.NewTester(func(c *ui.Context) {
		HScroll(c).Key("strip").TrackScroll(&st).Width(200).Height(40).Children(func() {
			ui.Box(c).Width(800).Height(30).Shrink(0)
		})
	}, 200, 40)
	tt.Move(100, 20)
	// 纯横向容器只认 dx;鼠标纵向刻度需 Shift 转换,见 MyGo ui/input.go(测试里 Wheel 不带 Mods,故用 dx 测)。
	tt.Wheel(100, 20, 40, 0)
	tt.Advance(200 * time.Millisecond)
	tt.Frame()
	if st.X <= 0 {
		t.Fatalf("横向容器滚轮未横向滚动: X=%v", st.X)
	}
}

func TestShiftWheelViaScrollBoth(t *testing.T) {
	Use(theme.Resolve(theme.Palettes[0], theme.Schemes[0], ""))
	SetScrollbarMode(ScrollbarFollowSystem)
	var st ui.ScrollState
	tt := ui.NewTester(func(c *ui.Context) {
		XYScroll(c).Key("both").TrackScroll(&st).Width(200).Height(120).Children(func() {
			ui.Box(c).Width(600).Height(400).Shrink(0)
		})
	}, 200, 120)
	tt.Move(100, 60)
	// Shift+滚轮:框架把 dy 转为 dx;用 Scroll(精确)模拟 Shift 后的横向增量。
	tt.Scroll(100, 60, 40, 0)
	tt.Frame()
	if st.Y != 0 {
		t.Fatalf("精确横向滚动不应改 Y: Y=%v", st.Y)
	}
	if st.X <= 0 {
		t.Fatalf("双向容器应横向滚动: X=%v", st.X)
	}
}

func TestScrollBarHidesAfterIdle(t *testing.T) {
	Use(theme.Resolve(theme.Palettes[0], theme.Schemes[0], ""))
	SetScrollbarMode(ScrollbarAutoHide)
	base := time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)
	defer SetScrollClock(nil)
	SetScrollClock(func() time.Time { return base })
	st := &ui.ScrollState{}
	if scrollShown("idle-test", st, false) {
		t.Fatal("刚出现、没有滚动过,不应显示滚动条")
	}
	st.Y = 40 // 偏移变化 = 有滚动(键盘 / 拖动 / 程序也算)
	if !scrollShown("idle-test", st, false) {
		t.Fatal("滚动后应显示滚动条")
	}
	SetScrollClock(func() time.Time { return base.Add(2900 * time.Millisecond) })
	if !scrollShown("idle-test", st, false) {
		t.Fatal("静止 2.9 秒内应仍显示")
	}
	if !ScrollFrameActive(base.Add(2900 * time.Millisecond)) {
		t.Fatal("等待隐藏期间应请求续帧")
	}
	SetScrollClock(func() time.Time { return base.Add(3100 * time.Millisecond) })
	if scrollShown("idle-test", st, false) {
		t.Fatal("静止满 3 秒应隐藏")
	}
	// 指针停在滚动条轨道上:不论多久都保持显示;指针离开后恢复计时
	setScrollEdge("idle-test", true)
	SetScrollClock(func() time.Time { return base.Add(30 * time.Second) })
	if !scrollShown("idle-test", st, true) {
		t.Fatal("指针停在滚动条上应一直显示")
	}
}

func TestScrollBarModeAppliesFlags(t *testing.T) {
	Use(theme.Resolve(theme.Palettes[0], theme.Schemes[0], ""))
	defer SetScrollbarMode(ScrollbarAutoHide)
	var st ui.ScrollState
	build := func(c *ui.Context) {
		VScroll(c).Key("flag-test").TrackScroll(&st).Width(200).Height(100).Children(func() {
			ui.Box(c).Width(100).Height(600).Shrink(0)
		})
	}
	SetScrollbarMode(ScrollbarAutoHide)
	tt := ui.NewTester(build, 200, 100)
	tt.Frame()
	tt.Scroll(100, 50, 0, 40)
	tt.Frame()
	if !scrollShown("flag-test", &st, false) {
		t.Fatal("滚动后应进入显示期")
	}
	tt.Move(198, 50) // 右边缘 = 滚动条轨道
	scrollMu.Lock()
	edge := scrollSlots["flag-test"].edge
	scrollMu.Unlock()
	if !edge {
		t.Fatal("指针移到右边缘应记为在滚动条上")
	}
	tt.Move(50, 50)
	scrollMu.Lock()
	edge = scrollSlots["flag-test"].edge
	scrollMu.Unlock()
	if edge {
		t.Fatal("指针离开边缘应清除")
	}
}
