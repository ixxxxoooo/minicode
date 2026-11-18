package app

import (
	"strconv"
	"strings"
	"time"

	"github.com/ixxxxoooo/minicode/internal/diag"
)

// 各形态各自记住的布局(设计稿 forms / resp):右侧栏、底部面板、导航栏开合、对话占满;左栏分段不记,切形态一律回默认。
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

// restoreLayoutFor 切到形态 m 时套用它记住的开合;左栏分段不记忆,一律回到该形态的默认
// (设计稿 setMode:Agent 绑定「会话」、IDE 绑定「文件」,a.nav 为空即取默认,见 NavID)。
func (a *App) restoreLayoutFor(m Mode) {
	a.nav = ""
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

// autoPersist 边用边存:侧栏 / 右栏 / 底部终端面板的开合、形态、导航分段、标签、会话、项目 / 工作区变了就落盘。
// 只靠退出时保存不可靠(开发版重启是 SIGTERM 杀进程、崩溃、强退都来不及保存),表现为「重启后界面不是退出前的样子」。
// 每帧只做几次比较;运行状态落盘限流(≥1.5s 一次)。
func (a *App) autoPersist() {
	if a.secondary || a.prefs == nil || !a.RestoreEnabled() {
		return
	}
	// 1) 各形态开合:几个布尔打包成整数,变了才写偏好
	cur := a.currentLayMem()
	n := 0
	for i, m := range []layMem{a.memBy[ModeAgent], a.memBy[ModeIDE]} {
		if Mode(i) == a.mode {
			m = cur
		}
		if m.right {
			n |= 1 << (i * 4)
		}
		if m.panel {
			n |= 2 << (i * 4)
		}
		if m.side {
			n |= 4 << (i * 4)
		}
		if m.chatMax {
			n |= 8 << (i * 4)
		}
	}
	n |= 1 << 16 // 非 0,区分「还没算过」
	if n != a.autoLay {
		first := a.autoLay == 0
		a.autoLay = n
		if !first {
			a.saveLayMemToPrefs()
			a.persistLastMode()
		}
	}
	// 2) 运行状态:会话 / 项目 / 工作区 / 导航 / 底部页签 / 编辑器与终端标签
	sig := a.runtimeSig()
	if sig != a.autoRtSig && a.now().Sub(a.autoRtAt) >= 1500*time.Millisecond {
		first := a.autoRtSig == ""
		a.autoRtSig, a.autoRtAt = sig, a.now()
		if !first {
			a.SaveRuntimeState()
		}
	}
}

// runtimeSig 是运行状态里会引起「重启后不一样」的那些量的签名(不含光标位置这类高频变化)。
func (a *App) runtimeSig() string {
	var b strings.Builder
	b.WriteString(a.curSess)
	b.WriteByte('|')
	b.WriteString(a.proj)
	if a.ws != nil {
		b.WriteString("|ws:" + a.ws.ID)
	}
	b.WriteString("|" + a.navBy[ModeAgent] + "|" + a.navBy[ModeIDE] + "|" + a.ide.bottomTab)
	b.WriteString("|" + strconv.Itoa(int(a.mode)) + strconv.Itoa(a.ide.tpSel) + strconv.Itoa(len(a.ide.tps)) + strconv.Itoa(int(a.ide.splitDir)))
	b.WriteString("|" + a.ide.active)
	for _, tb := range a.ide.tabs {
		if tb != nil {
			b.WriteString(";" + tb.path)
		}
	}
	for _, tb := range a.ide.tabs2 {
		if tb != nil {
			b.WriteString(";;" + tb.path)
		}
	}
	return b.String()
}
