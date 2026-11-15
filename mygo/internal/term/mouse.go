package term

import (
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

// mouseModes 是位掩码:哪些鼠标跟踪 DEC 模式当前为开。
var mouseBits = map[ansi.Mode]uint32{
	ansi.ModeMouseX10:         1 << 0,
	ansi.ModeMouseNormal:      1 << 1,
	ansi.ModeMouseHighlight:   1 << 2,
	ansi.ModeMouseButtonEvent: 1 << 3,
	ansi.ModeMouseAnyEvent:    1 << 4,
}

// MouseTracking 报告仿真器是否正在向程序上报鼠标事件。
func (t *Term) MouseTracking() bool {
	return t.mouseModes.Load() != 0
}

// SendMouse 把鼠标事件编码后发给 shell。
func (t *Term) SendMouse(ev uv.MouseEvent) {
	if t.em == nil {
		return
	}
	if !mouseWanted(t.mouseModes.Load(), ev) {
		return
	}
	t.em.SendMouse(ev)
}

// mouseWanted 按程序开的跟踪模式过滤事件:vt 库不区分,会把移动事件发给只开了 1000(仅点击)的程序。
//   - 1000 / 1001 / 9:只要点击与释放(滚轮属于按键事件,照常放行)
//   - 1002:再加「按住按键时的移动」
//   - 1003:再加所有移动
func mouseWanted(modes uint32, ev uv.MouseEvent) bool {
	if _, ok := ev.(uv.MouseMotionEvent); !ok {
		return true
	}
	if modes&mouseBits[ansi.ModeMouseAnyEvent] != 0 {
		return true
	}
	return modes&mouseBits[ansi.ModeMouseButtonEvent] != 0 && ev.Mouse().Button != uv.MouseNone
}

func (t *Term) onMouseMode(mode ansi.Mode, on bool) {
	bit, ok := mouseBits[mode]
	if !ok {
		return
	}
	for {
		old := t.mouseModes.Load()
		var neu uint32
		if on {
			neu = old | bit
		} else {
			neu = old &^ bit
		}
		if t.mouseModes.CompareAndSwap(old, neu) {
			return
		}
	}
}
