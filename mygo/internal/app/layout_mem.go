package app

import (
	"github.com/ixxxxoooo/minicode/internal/diag"
)

// 各形态各自记住的布局(设计稿 forms / resp):右侧栏、底部面板、导航栏开合、对话占满、左栏分段。
const (
	kLastMode  = "layout.lastMode"
	kStartMode = "general.start"
)

func (a *App) currentLayMem() layMem {
	return layMem{
		right:   a.rightOn,
		panel:   a.ide.bottomOpen,
		side:    a.sideOpen,
		chatMax: a.chatMax,
	}
}

func (a *App) applyLayMem(m layMem) {
	a.rightOn, a.ide.bottomOpen = m.right, m.panel
	a.sideOpen, a.chatMax = m.side, m.chatMax
}

// rememberLayout 在切形态前把当前形态的布局写入内存(并尝试落盘)。
func (a *App) rememberLayout() {
	a.navBy[a.mode] = a.nav
	a.memBy[a.mode] = a.currentLayMem()
	a.persistLastMode()
}

func (a *App) restoreLayoutFor(m Mode) {
	a.nav = a.navBy[m]
	a.applyLayMem(a.memBy[m])
}

// ResolveStartupMode 按 general.start 与上次形态决定启动时的 Mode。
func ResolveStartupMode(p PrefsReader, envIDE bool) Mode {
	if envIDE {
		return ModeIDE
	}
	switch p.int(kStartMode, 2) {
	case 0:
		return ModeAgent
	case 1:
		return ModeIDE
	default:
		if p.int(kLastMode, 0) == 1 {
			return ModeIDE
		}
		return ModeAgent
	}
}

// PrefsReader 是启动阶段只读偏好的薄封装(避免 app 未建好)。
type PrefsReader struct {
	getInt func(key string, def int) int
}

// NewPrefsReader 构造偏好只读访问器。
func NewPrefsReader(getInt func(key string, def int) int) PrefsReader {
	return PrefsReader{getInt: getInt}
}

func (p PrefsReader) int(key string, def int) int {
	if p.getInt == nil {
		return def
	}
	return p.getInt(key, def)
}

func (a *App) persistLastMode() {
	if a.secondary || a.prefs == nil {
		return
	}
	a.prefs.Set(kLastMode, int(a.mode))
}

// loadLayMemFromPrefs 从偏好恢复各形态的右侧栏 / 底栏(启动时调用;导航分段仍用内存默认)。
func (a *App) loadLayMemFromPrefs() {
	if a.prefs == nil {
		return
	}
	if !a.RestoreEnabled() {
		return
	}
	// general.insp: Agent 形态开标签时是否自动展开右侧工作区
	if a.prefs.Bool("general.insp", true) {
		a.memBy[ModeAgent].right = true
	}
	load := func(key string, m *layMem) {
		if !a.prefs.Has(key) {
			return
		}
		v := a.prefs.Int(key, 0)
		m.right = v&1 != 0
		m.panel = v&2 != 0
		m.side = v&4 != 0
		m.chatMax = v&8 != 0
	}
	load("layout.memAgent", &a.memBy[ModeAgent])
	load("layout.memIDE", &a.memBy[ModeIDE])
	logMem := diag.L("layout")
	logMem.Debug("载入形态布局", "agent", a.memBy[ModeAgent], "ide", a.memBy[ModeIDE])
}

func (a *App) saveLayMemToPrefs() {
	if a.secondary || a.prefs == nil {
		return
	}
	if !a.RestoreEnabled() {
		return
	}
	pack := func(m layMem) int {
		n := 0
		if m.right {
			n |= 1
		}
		if m.panel {
			n |= 2
		}
		if m.side {
			n |= 4
		}
		if m.chatMax {
			n |= 8
		}
		return n
	}
	a.memBy[a.mode] = a.currentLayMem()
	a.prefs.Set("layout.memAgent", pack(a.memBy[ModeAgent]))
	a.prefs.Set("layout.memIDE", pack(a.memBy[ModeIDE]))
}
