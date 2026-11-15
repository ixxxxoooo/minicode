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
	t.em.SendMouse(ev)
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
