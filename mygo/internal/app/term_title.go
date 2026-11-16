package app

import (
	"path/filepath"
	"strings"

	"github.com/ixxxxoooo/minicode/internal/kit"
)

const (
	termTabMinW       float32 = 72
	termTabMaxW       float32 = 168
	termListMinW      float32 = 120
	termListMaxW      float32 = 280
	termListDefW      float32 = 168
	termAutoMinW      float32 = 640
	termForceHorizW   float32 = 560
	termTitleFontSize float32 = 11.5
	termTabPadX       float32 = 7 + 12 + 4 // chip 左内边距 + 图标 + 间隙
)

const (
	pTermTabs   = "term.tabs"
	pTermListW  = "term.listWidth"
	termTabHorz = 0
	termTabVert = 1
	termTabAuto = 2
)

// termTabRawTitle 按设计稿优先级取标签标题全文(不截断)。
func (a *App) termTabRawTitle(i int) string {
	if i < 0 || i >= len(a.ide.tps) {
		return ""
	}
	tp := a.ide.tps[i]
	if s := strings.TrimSpace(tp.label); s != "" {
		return s
	}
	if tp.t != nil {
		if s := strings.TrimSpace(tp.t.Title()); s != "" {
			return s
		}
	}
	if s := strings.TrimSpace(tp.demoTitle); s != "" {
		return s
	}
	// 前台进程名:PTY 层暂未实现
	cwd := tp.cwd
	if cwd == "" {
		cwd = a.proj
	}
	if cwd != "" {
		return filepath.Base(cwd)
	}
	if i == 0 {
		return shellName()
	}
	return "终端 " + itoa(i+1)
}

// termPathLike 是否应按路径做中间截断。
func termPathLike(s string) bool {
	return strings.Contains(s, "/") || strings.HasPrefix(s, "~")
}

// termNormalizeTitle 展示前规范化(超长命令取可执行文件 basename)。
func termNormalizeTitle(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}
	sp := strings.IndexByte(raw, ' ')
	if sp <= 0 {
		return raw
	}
	head := raw[:sp]
	if strings.Contains(head, "/") && kit.TextW(raw, termTitleFontSize) > termTabMaxW {
		return filepath.Base(head) + raw[sp:]
	}
	return raw
}

// termTabTitle 把全文截到不超过 maxPx 的显示宽度。
func termTabTitle(raw string, maxPx float32) string {
	raw = termNormalizeTitle(raw)
	if maxPx <= 0 {
		return raw
	}
	if kit.TextW(raw, termTitleFontSize) <= maxPx {
		return raw
	}
	if termPathLike(raw) {
		return ellipsizeMiddle(raw, maxPx, termTitleFontSize)
	}
	return ellipsizeTail(raw, maxPx, termTitleFontSize)
}

// ellipsizeTail 尾部省略号。
func ellipsizeTail(s string, maxW, fontSize float32) string {
	const ell = "…"
	if maxW <= kit.TextW(ell, fontSize) {
		return ell
	}
	runes := []rune(s)
	lo, hi := 0, len(runes)
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if kit.TextW(string(runes[:mid])+ell, fontSize) <= maxW {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	if lo == 0 {
		return ell
	}
	return string(runes[:lo]) + ell
}

// ellipsizeMiddle 中间省略,尽量保留首段与末段。
func ellipsizeMiddle(s string, maxW, fontSize float32) string {
	const ell = "…"
	if maxW <= kit.TextW(ell, fontSize) {
		return ell
	}
	parts := strings.Split(s, "/")
	if len(parts) >= 2 {
		head := parts[0]
		tail := parts[len(parts)-1]
		candidate := head + ell + tail
		if kit.TextW(candidate, fontSize) <= maxW {
			return candidate
		}
	}
	runes := []rune(s)
	n := len(runes)
	if n < 4 {
		return ellipsizeTail(s, maxW, fontSize)
	}
	for keep := n / 3; keep >= 1; keep-- {
		candidate := string(runes[:keep]) + ell + string(runes[n-keep:])
		if kit.TextW(candidate, fontSize) <= maxW {
			return candidate
		}
	}
	return ell
}

// termTabWidths 为横向标签条分配宽度:先压到 max,再等比缩,最后落到 min 并依赖滚动。
func termTabWidths(texts []string, avail float32) []float32 {
	if len(texts) == 0 {
		return nil
	}
	out := make([]float32, len(texts))
	var sum float32
	for i, raw := range texts {
		text := termTabTitle(raw, termTabMaxW-termTabPadX)
		w := kit.TextW(text, termTitleFontSize) + termTabPadX + 8
		w = min(max(w, termTabMinW), termTabMaxW)
		out[i], sum = w, sum+w
	}
	if avail <= 0 || sum <= avail {
		return out
	}
	scale := avail / sum
	for i := range out {
		out[i] = max(termTabMinW, out[i]*scale)
		sum = 0
		for _, w := range out {
			sum += w
		}
		if sum <= avail {
			return out
		}
	}
	for i := range out {
		out[i] = termTabMinW
	}
	return out
}

// termTabsVertical 是否用竖向终端列表。
func (a *App) termTabsVertical(panelW float32) bool {
	if len(a.ide.tps) < 2 || a.ide.termSel != "" {
		return false
	}
	if panelW > 0 && panelW < termForceHorizW {
		return false
	}
	if a.chatTier() == tierNarrow {
		return false
	}
	switch a.prefs.Int(pTermTabs, termTabAuto) {
	case termTabHorz:
		return false
	case termTabVert:
		return true
	default:
		return panelW >= termAutoMinW
	}
}

// termListWidth 竖向列表当前宽度。
func (a *App) termListWidth() float32 {
	if a.ly.termList > 0 {
		return a.ly.termList
	}
	return termListDefW
}
