package app

import (
	"strings"
)

// termCmdClick 点击命令装饰点:复制该命令块的输出。
func (a *App) termCmdClick(tp *termPanel, p cellPos) bool {
	if tp.t == nil || tp.scr == nil {
		return false
	}
	absRow := tp.scr.LineBase + p.row
	for _, b := range tp.t.CommandBlocks() {
		if absRow != b.StartRow || p.col > 2 {
			continue
		}
		endAbs := b.EndRow
		if endAbs <= b.StartRow {
			endAbs = b.StartRow + 1
		}
		startY, endY := b.StartRow-tp.scr.LineBase, endAbs-tp.scr.LineBase
		var rows []string
		for y := startY; y < endY && y >= 0 && y < len(tp.scr.Lines); y++ {
			rows = append(rows, lineString(tp.scr.Lines[y]))
		}
		if len(rows) == 0 {
			return true
		}
		a.copyText = strings.TrimSpace(strings.Join(rows, "\n"))
		a.toast = "已复制命令输出"
		return true
	}
	return false
}
