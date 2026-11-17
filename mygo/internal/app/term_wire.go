package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

const (
	pTermFontWeight    = "term.fontWeight"
	pTermLigatures     = "term.ligatures"
	pTermLetterSpacing = "term.letterSpacing" // 保留键;暂未接 CellWidth
	pTermCursorDim     = "term.cursorDim"
	pTermWheel         = "term.wheel"
	pTermRightClick    = "term.rightClick"
	pTermOptionMeta    = "term.optionMeta"
	pTermLinkMod       = "term.linkMod"
	pTermBell          = "term.bell"
	pTermPasteMulti    = "term.pasteMultiConfirm"
	pTermCloseConfirm  = "term.closeConfirm"
	pTermShellArgs     = "term.shellArgs"
	pTermEnv           = "term.env"
)

// termWheelLines 滚轮回看行数(1–5)。
func (a *App) termWheelLines() int {
	n := a.prefs.Int(pTermWheel, 3)
	return min(max(n, 1), 5)
}

// termFontFeatureList 终端 OpenType 特性(连字 + macOS 关 kern)。
func (a *App) termFontFeatureList() string {
	if a.prefs.Bool(pTermLigatures, false) {
		return "kern=0,calt,liga"
	}
	return "kern=0,calt=0,liga=0"
}

// termBaseFontWeight 非 ANSI 粗体时的字重。
func (a *App) termBaseFontWeight() int {
	if a.prefs.Int(pTermFontWeight, 0) == 1 {
		return 700
	}
	return 400
}

// termLinkUsesSuper 链接点击修饰键:false = ⌃/⌘(Super)。
func (a *App) termLinkUsesSuper() bool {
	return a.prefs.Int(pTermLinkMod, 0) == 0
}

// termLinkModHeld 是否按住打开链接的修饰键。
func (a *App) termLinkModHeld(m ui.Modifiers) bool {
	if a.termLinkUsesSuper() {
		return m&ui.Super != 0
	}
	return m&ui.Ctrl != 0
}

// termRightClickPastes 右键直接粘贴(不弹出菜单)。
func (a *App) termRightClickPastes() bool {
	return a.prefs.Int(pTermRightClick, 0) == 1
}

// termOptionAsMeta Option 映射为 Meta(给 shell / Emacs 键位)。
func (a *App) termOptionAsMeta() bool {
	return a.prefs.Bool(pTermOptionMeta, false)
}

// termBellMode 0=关 1=视觉 2=系统声音(尽力播放)。
func (a *App) termBellMode() int {
	return min(max(a.prefs.Int(pTermBell, 1), 0), 2)
}

// parseTermEnv 解析 KEY=VALUE 行(一行一个,# 开头忽略)。
func parseTermEnv(raw string) []string {
	var out []string
	for line := range strings.SplitSeq(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if i := strings.IndexByte(line, '='); i > 0 {
			out = append(out, line)
		}
	}
	return out
}

// parseShellArgs 简易拆分(不支持引号内空格;复杂参数用手动 Shell 包装脚本)。
func parseShellArgs(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	return strings.Fields(raw)
}

// termCloseNeedsConfirm 关闭标签前是否确认。
func (a *App) termCloseNeedsConfirm(running bool) bool {
	switch a.prefs.Int(pTermCloseConfirm, 0) {
	case 2:
		return false
	case 1:
		return true
	default:
		return running
	}
}

// termPasteNeedsConfirm 多行粘贴是否弹确认。
func (a *App) termPasteNeedsConfirm(text string) bool {
	if !a.prefs.Bool(pTermPasteMulti, true) {
		return false
	}
	return strings.Contains(text, "\n") || strings.Contains(text, "\r")
}

// applyTermRuntimePrefs 把终端偏好写入包级变量(applyBehavior 调用)。
func (a *App) applyTermRuntimePrefs() {
	termFontFeats = a.termFontFeatureList()
	termBaseWeight = a.termBaseFontWeight()
	termOptionAsMeta = a.termOptionAsMeta()
}

// termBell 终端响铃(BEL):视觉闪屏;声音模式同样闪屏(系统蜂鸣未接)。
func (a *App) termBell(tp *termPanel) {
	if a.termBellMode() == 0 || tp == nil {
		return
	}
	tp.bellUntil = kit.Now()
	a.redraw()
}

var termFontFeats string
var termBaseWeight = 400
var termOptionAsMeta bool

// termCursorIdleOpacity 非聚焦光标不透明度。
func (a *App) termCursorIdleOpacity() float32 {
	switch a.prefs.Int(pTermCursorDim, 1) {
	case 0:
		return 0.25
	case 2:
		return 0.5
	default:
		return 0.35
	}
}

// termScrollDelta 根据滚轮事件与偏好算滚动偏移(px)。
func (a *App) termScrollDelta(dy float32) float32 {
	if dy == 0 {
		return 0
	}
	sign := float32(1)
	if dy > 0 {
		sign = -1
	}
	return sign * float32(a.termWheelLines()) * termLineH
}
