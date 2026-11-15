package app

import (
	"strings"
)

// termCmdClick 点击命令装饰点:复制该命令块的输出。
func (a *App) termCmdClick(tp *termPanel, p cellPos) bool {
	if tp.t == nil || tp.scr == nil {
		return false
	}
	for _, b := range tp.t.CommandBlocks() {
		if p.row != b.StartRow || p.col > 2 {
			continue
		}
		end := b.EndRow
		if end <= b.StartRow || end > len(tp.scr.Lines) {
			end = len(tp.scr.Lines)
		}
		var rows []string
		for y := b.StartRow; y < end && y < len(tp.scr.Lines); y++ {
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
