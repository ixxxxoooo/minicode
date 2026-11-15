package app

import (
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/egoist/mygo/ui"
)

// termPointerToMouse 在鼠标上报模式下把指针事件编码给 shell;按住 ⌥ 时走选择逻辑。
func (a *App) termPointerToMouse(tp *termPanel, ev ui.InputEvent) bool {
	if tp.t == nil || !tp.t.MouseTracking() || ev.Mods&ui.Alt != 0 {
		return false
	}
	p := a.termCell(tp, ev.X, ev.Y)
	if tp.scr == nil {
		return false
	}
	// 鼠标坐标相对当前屏幕(不含历史偏移时 y 要减去 ScreenTop)
	row := p.row
	if row >= tp.scr.ScreenTop {
		row -= tp.scr.ScreenTop
	} else {
		// 点在历史区:多数程序只关心屏幕区,仍发屏幕坐标 0
		row = 0
	}
	col := p.col
	btn := uv.MouseLeft
	switch ev.Button {
	case 1:
		btn = uv.MouseRight
	case 2:
		btn = uv.MouseMiddle
	}
	var me uv.MouseEvent
	switch ev.Kind {
	case ui.InputPointerDown:
		if ev.Button != 0 && ev.Button != 2 {
			return false
		}
		me = uv.MouseClickEvent{X: col, Y: row, Button: btn, Mod: termMods(ev.Mods)}
	case ui.InputPointerUp:
		if ev.Button != 0 && ev.Button != 2 {
			return false
		}
		me = uv.MouseReleaseEvent{X: col, Y: row, Button: btn, Mod: termMods(ev.Mods)}
	case ui.InputPointerMove:
		if ev.Button == 0 {
			me = uv.MouseMotionEvent{X: col, Y: row, Button: uv.MouseNone, Mod: termMods(ev.Mods)}
		} else {
			me = uv.MouseMotionEvent{X: col, Y: row, Button: btn, Mod: termMods(ev.Mods)}
		}
	default:
		return false
	}
	tp.t.SendMouse(me)
	return true
}

// termPointerWheel 在鼠标跟踪开启时把滚轮编成 SGR 滚轮事件。
func (a *App) termPointerWheel(tp *termPanel, ev ui.InputEvent) bool {
	if tp.t == nil || !tp.t.MouseTracking() || ev.Mods&ui.Alt != 0 {
		return false
	}
	p := a.termCell(tp, ev.X, ev.Y)
	btn := uv.MouseWheelUp
	if ev.DY > 0 {
		btn = uv.MouseWheelDown
	} else if ev.DY == 0 && ev.DX > 0 {
		btn = uv.MouseWheelRight
	} else if ev.DY == 0 && ev.DX < 0 {
		btn = uv.MouseWheelLeft
	}
	row := p.row
	if row >= tp.scr.ScreenTop {
		row -= tp.scr.ScreenTop
	}
	me := uv.MouseWheelEvent{X: p.col, Y: row, Button: btn, Mod: termMods(ev.Mods)}
	tp.t.SendMouse(me)
	return true
}
