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

func TestScrollBarFadeAfterIdle(t *testing.T) {
	Use(theme.Resolve(theme.Palettes[0], theme.Schemes[0], ""))
	SetScrollbarMode(ScrollbarAutoHide)
	SetMotion(true)
	base := time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)
	SetScrollClock(func() time.Time { return base })
	regScroll("fade-test", &ui.ScrollState{}, false, true)
	markScrollActivity("fade-test")
	SetScrollClock(func() time.Time { return base.Add(200 * time.Millisecond) })
	if scrollAlpha("fade-test") < 0.5 {
		t.Fatalf("滚动后应显示滚动条,alpha=%v", scrollAlpha("fade-test"))
	}
	SetScrollClock(func() time.Time { return base.Add(4 * time.Second) })
	if scrollAlpha("fade-test") != 0 {
		t.Fatalf("停止 4s 后应隐藏,alpha=%v", scrollAlpha("fade-test"))
	}
}
