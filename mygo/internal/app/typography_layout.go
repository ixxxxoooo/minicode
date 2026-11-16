package app

import (
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 会话内容列宽档位(appearance.flowWidth);默认 2 = 标准 712。
const (
	pFlowWidth     = "appearance.flowWidth"
	pWidthLegacy   = "appearance.width"
	flowWidthFull  = 5
	flowWidthCapPx = 1400
)

var flowWidthTiers = []float32{560, 640, theme.WCol, 860, 1040}

// flowWidthIndex 读档位;旧 appearance.width 0/1/2 → 紧凑 / 标准 / 宽。
func (a *App) flowWidthIndex() int {
	if a == nil || a.prefs == nil {
		return 2
	}
	if a.prefs.Has(pFlowWidth) {
		return clampInt(a.prefs.Int(pFlowWidth, 2), 0, flowWidthFull)
	}
	if a.prefs.Has(pWidthLegacy) {
		switch a.prefs.Int(pWidthLegacy, 1) % 3 {
		case 0:
			return 1
		case 2:
			return 3
		default:
			return 2
		}
	}
	return 2
}

// flowAvailWidth 是对话栏里内容列的可用宽(扣 flowPad)。
func (a *App) flowAvailWidth() float32 {
	if a == nil || a.chatW <= 0 {
		return 0
	}
	return a.chatW - 2*a.flowPad()
}

// flowContentMaxWidth 返回会话流 / 输入框共用的最大列宽(已随容器收窄)。
func (a *App) flowContentMaxWidth() float32 {
	idx := a.flowWidthIndex()
	avail := a.flowAvailWidth()
	if idx == flowWidthFull {
		cap := float32(flowWidthCapPx)
		if avail > 0 {
			return min(avail, cap)
		}
		return cap
	}
	pref := flowWidthTiers[idx]
	if avail > 0 {
		return min(pref, avail)
	}
	return pref
}

// ───────── 设置页内容列宽 ─────────

const (
	pSetWidth    = "appearance.setWidth"
	setWidthFull = 4
)

var setWidthTiers = []float32{620, 760, 900, 1080}

// settingsContentWidth 设置主栏目标宽度;窄窗返回 0 表示 Grow。
func (a *App) settingsContentWidth() float32 {
	if a == nil || a.settingsNarrow() {
		return 0
	}
	if a.prefs == nil {
		return setWidthTiers[1]
	}
	idx := clampInt(a.prefs.Int(pSetWidth, 1), 0, setWidthFull)
	if idx == setWidthFull {
		// 占满:主区减左右 padding(18+28) 与导航
		w := a.winW - 196 - 46
		if w < setWidthTiers[0] {
			return setWidthTiers[0]
		}
		return float32(w)
	}
	return setWidthTiers[idx]
}

func clampInt(v, lo, hi int) int {
	return min(max(v, lo), hi)
}
