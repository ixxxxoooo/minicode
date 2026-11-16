package kit

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// 带标识的滚动区外面的容器必须还能收到指针按下 / 移动:
// 框架只把指针事件交给命中链上最里面的处理器,滚动区自己挂 HandleInput 会让外层(如终端正文的选择)永远收不到。
func TestKeyedScrollDoesNotStealPointer(t *testing.T) {
	var st ui.ScrollState
	downs := 0
	view := func(c *ui.Context) {
		outer := ui.Box(c).Key("outer").Size(300, 200).HandleInput(func(ev ui.InputEvent) bool {
			if ev.Kind == ui.InputPointerDown {
				downs++
				return true
			}
			return false
		})
		outer.Children(func() {
			VScroll(c).Key("inner-scroll").TrackScroll(&st).Grow(1).Children(func() {
				ui.Column(c).Children(func() {
					for i := 0; i < 40; i++ {
						ui.Text(c, "行文字内容")
					}
				})
			})
		})
	}
	tt := ui.NewTester(view, 300, 200)
	tt.Frame()
	tt.Frame()
	tt.ClickAt(80, 30)
	tt.Frame()
	if downs == 0 {
		t.Fatal("外层容器没有收到指针按下:滚动区把事件吞了")
	}
}
