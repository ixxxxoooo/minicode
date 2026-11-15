package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// termFindMarks 在滚动区域右侧画搜索命中刻度。
func (a *App) termFindMarks(c *ui.Context, tp *termPanel, body ui.Element) {
	if tp.scr == nil || len(tp.scr.Lines) < 2 {
		return
	}
	t := a.tk
	b := body.Bounds()
	total := float32(len(tp.scr.Lines)) * termLineH
	if total <= 0 {
		return
	}
	trackW := float32(3)
	trackX := b.X + b.W - trackW - 4
	ui.Box(c).Absolute().Left(trackX).Top(b.Y + 6).Width(trackW).Height(b.H - 12).
		Background(t.Bd2).Radius(2).PassThrough()
	for _, h := range tp.find.hits {
		y := float32(h.row) * termLineH
		hh := termLineH
		if hh < 2 {
			hh = 2
		}
		rel := y / total
		top := b.Y + 6 + rel*(b.H-12)
		col := t.Warn.Alpha(.85)
		if tp.find.open && tp.find.cur < len(tp.find.hits) {
			cur := tp.find.hits[tp.find.cur]
			if cur.row == h.row && cur.c0 == h.c0 {
				col = t.Ac
			}
		}
		ui.Box(c).Absolute().Left(trackX).Top(top).Width(trackW).Height(hh).Background(col).Radius(1).PassThrough()
	}
	_ = theme.FsCap
}
