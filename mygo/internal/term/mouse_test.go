package term

import (
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

// 只开 1000 的程序不该收到移动事件;1002 只收按住时的移动;1003 全收。
func TestMouseWantedByMode(t *testing.T) {
	bare := uv.MouseMotionEvent{Button: uv.MouseNone}
	drag := uv.MouseMotionEvent{Button: uv.MouseLeft}
	click := uv.MouseClickEvent{Button: uv.MouseLeft}
	cases := []struct {
		name  string
		modes uint32
		ev    uv.MouseEvent
		want  bool
	}{
		{"1000 点击", mouseBits[ansi.ModeMouseNormal], click, true},
		{"1000 裸移动", mouseBits[ansi.ModeMouseNormal], bare, false},
		{"1000 按住移动", mouseBits[ansi.ModeMouseNormal], drag, false},
		{"1002 按住移动", mouseBits[ansi.ModeMouseButtonEvent], drag, true},
		{"1002 裸移动", mouseBits[ansi.ModeMouseButtonEvent], bare, false},
		{"1003 裸移动", mouseBits[ansi.ModeMouseAnyEvent], bare, true},
	}
	for _, c := range cases {
		if got := mouseWanted(c.modes, c.ev); got != c.want {
			t.Errorf("%s: 得到 %v,应为 %v", c.name, got, c.want)
		}
	}
}
