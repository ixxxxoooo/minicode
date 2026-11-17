package app

import (
	"strings"
)

// 光标行末尾的追溯提示(GitLens 的 current line blame):永远是单行的淡色小字,不能撑出换行。
// 业界做法:提交说明只取首行,放不下就截断(完整说明在悬停卡里)。这里的截断「随当前宽度自适应」:
// 用编辑器此刻的宽度减去该行已占的宽度(开了自动换行就是折行后的最后一行),剩多少放多少,
// 放不下补「…」,窄到放不下像样的内容就不显示——而不是写死一个字数。
// 库自带的 Ellipsis 靠不住:遇到换行机会(如全角分号)会折到下一行,甚至把「…」单独折下去。
//
// @author ixxxxoooo

const (
	blameAuthorCells = 16 // 作者名最多占几格
	blameMsgFallback = 50 // 量不出编辑器宽度时(首帧)的说明格数上限
	blameMinRoom     = 14 // 剩余格数少于这个就不显示(放不下「作者 · 时间」)
	blameSafetyCells = 4  // 估算与实际排版的误差余量(实测编辑器的折行边界比「宽度 − 行号栏 − 轨道」略靠左,留 4 格兜住)
)

// cellWidth 一个字符占的显示格数:中日韩 / 全角 2 格,其余 1 格。
func cellWidth(r rune) int {
	switch {
	case r >= 0x1100 && r <= 0x115F, r >= 0x2E80 && r <= 0xA4CF, r >= 0xAC00 && r <= 0xD7A3,
		r >= 0xF900 && r <= 0xFAFF, r >= 0xFE30 && r <= 0xFE6F, r >= 0xFF00 && r <= 0xFF60, r >= 0xFFE0 && r <= 0xFFE6,
		r >= 0x1F300 && r <= 0x1FAFF:
		return 2
	}
	return 1
}

// glyphW 追溯提示里一个字符的宽度(单位:一个英文等宽字的宽):12px 等宽字 7.2px,
// 中日韩字走系统回退字体、宽度约等于字号 12px,即 1.67 格。比「按 2 格」更贴近右边缘,少留白。
func glyphW(r rune) float32 {
	if cellWidth(r) == 2 {
		return 1.67
	}
	return 1
}

// textCells 一段提示文字占的格数(带小数)。
func textCells(s string) float32 {
	var n float32
	for _, r := range s {
		n += glyphW(r)
	}
	return n
}

// clipCells 把 s 压成单行并截到最多 max 格,超出补「…」(补上的省略号也算在 max 里)。
func clipCells(s string, max float32) string {
	s = strings.Join(strings.Fields(strings.ReplaceAll(s, " ", " ")), " ") // 换行 / 制表 / 连续空白都压成一个空格
	if textCells(s) <= max {
		return s
	}
	var b strings.Builder
	var w float32
	for _, r := range s {
		cw := glyphW(r)
		if w+cw > max-1 { // 留 1 格给省略号
			break
		}
		b.WriteRune(r)
		w += cw
	}
	return strings.TrimRight(b.String(), " ") + "…"
}

// lastRowCells 估算一行代码在 cols 格宽的编辑区里排完后,最后一行占了多少格。
// wrap 为假时不折行,就是整行宽度(Tab 对齐到 tabW)。为真时按「词不拆开、放不下整词移到下一行、
// 中日韩字之间任意处可断、超长词按字符断」估算——与编辑器的折行规则接近,误差由 blameSafetyCells 兜住。
func lastRowCells(line string, cols, tabW int, wrap bool) int {
	if tabW <= 0 {
		tabW = 4
	}
	line = strings.TrimRight(line, "\r\n")
	x, ww := 0, 0 // x 当前行已占格数;ww 末尾那个词(不含空白)的宽度
	for _, r := range line {
		w := cellWidth(r)
		if r == '\t' {
			w = tabW - x%tabW
		}
		switch {
		case r == ' ' || r == '\t':
			ww = 0
		case w == 2: // 宽字符前后都能断,自成一「词」
			ww = w
		default:
			ww += w
		}
		if wrap && cols > 0 && x+w > cols {
			if ww > w && ww <= x { // 整个词移到下一行
				x = ww
			} else {
				x, ww = w, w
			}
			continue
		}
		x += w
	}
	return x
}

// blameInlayFit 行末追溯提示的文字:「作者 · 时间 · 摘要首行」,在 room 格内排满,摘要放不下就截断。
// room < blameMinRoom 时返回空串(不显示);room <= 0 表示量不出宽度,按 blameMsgFallback 截说明。
func blameInlayFit(author, when, summary string, room int) string {
	head := "    " + clipCells(author, blameAuthorCells) + " · " + when
	if room <= 0 {
		return head + " · " + clipCells(summary, blameMsgFallback)
	}
	if room < blameMinRoom {
		return ""
	}
	if hc := textCells(head); hc > float32(room) {
		return clipCells(head, float32(room)) // 连「作者 · 时间」都放不下:截断(开头的空格会被压掉,凑合显示)
	}
	msgMax := float32(room) - textCells(head) - 3
	if msgMax < 6 {
		return head
	}
	return head + " · " + clipCells(summary, msgMax)
}

// blameRoom 估算光标行末尾还剩多少格可放追溯提示;量不出时返回 0。
func (a *App) blameRoom(tb *ideTab, line int) int {
	if a.ide.edViewW <= 0 || tb.st == nil || tb.buf == nil {
		return 0
	}
	cw := edChar * edScale
	// 编辑区文字宽 = 编辑器宽 − 行号栏 − 左留白 − 滚动条轨道 − 右侧一点余量
	cols := int((a.ide.edViewW - edGutterW(tb) - edTextPad - edTrackW - 6) / cw)
	if cols <= 0 {
		return 0
	}
	used := lastRowCells(tb.buf.Line(line), cols, tb.st.TabSize, tb.st.SoftWrap)
	return max(cols-used-blameSafetyCells, 1) // 至少 1,免得被当成「量不出」
}
