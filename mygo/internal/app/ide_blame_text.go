package app

import "strings"

// 光标行末尾的追溯提示(GitLens 的 current line blame):永远是单行的淡色小字,不能撑出换行。
// 业界做法:提交说明只取首行并截到约 50 个字符(GitLens 默认 `${message|50?}`),完整说明放在悬停卡里;
// 这里按「显示宽度」截(中文 / 全角占 2 格),免得一条长中文说明比英文长出一倍。
//
// @author ixxxxoooo

const (
	blameMsgCells    = 50 // 提交说明最多占几格
	blameAuthorCells = 16 // 作者名最多占几格
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

// clipCells 把 s 压成单行并截到最多 max 格,超出补「…」(补上的省略号也算在 max 里)。
func clipCells(s string, max int) string {
	s = strings.Join(strings.Fields(strings.ReplaceAll(s, " ", " ")), " ") // 换行 / 制表 / 连续空白都压成一个空格
	w := 0
	for _, r := range s {
		w += cellWidth(r)
	}
	if w <= max {
		return s
	}
	var b strings.Builder
	w = 0
	for _, r := range s {
		cw := cellWidth(r)
		if w+cw > max-1 { // 留 1 格给省略号
			break
		}
		b.WriteRune(r)
		w += cw
	}
	return strings.TrimRight(b.String(), " ") + "…"
}

// blameInlayText 行末追溯提示的文字:「作者 · 时间 · 摘要首行」,摘要与作者名各自截断。
func blameInlayText(author, when, summary string) string {
	return "    " + clipCells(author, blameAuthorCells) + " · " + when + " · " + clipCells(summary, blameMsgCells)
}
