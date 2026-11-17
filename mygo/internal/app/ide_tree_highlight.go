package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
)

// treeHighlightName 过滤命中时在文件名里高亮关键字(不区分大小写)。
func (a *App) treeHighlightName(c *ui.Context, name, q string) {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" || !strings.Contains(strings.ToLower(name), q) {
		ui.Text(c, name).Grow(1).SingleLine()
		return
	}
	t := a.tk
	lo := strings.ToLower(name)
	row := ui.Row(c).Grow(1).MinWidth(0).AlignItems(ui.Center)
	idx := strings.Index(lo, q)
	row.Children(func() {
		if idx > 0 {
			ui.Text(c, name[:idx]).SingleLine().Shrink(0)
		}
		ui.Text(c, name[idx:idx+len(q)]).SingleLine().Shrink(0).TextColor(t.Ac).FontWeight(600)
		if idx+len(q) < len(name) {
			ui.Text(c, name[idx+len(q):]).SingleLine().Grow(1).MinWidth(0)
		}
	})
}
