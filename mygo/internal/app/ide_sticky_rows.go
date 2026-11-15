package app

// stickyRows 返回粘性滚动条占用的行数(供 CodeLens 下移避免重叠)。
func (a *App) stickyRows(tb *ideTab, first int) int {
	if tb == nil || tb.st == nil {
		return 0
	}
	syms := a.ide.fileSyms(tb.path)
	var rows int
	for _, k := range enclosing(syms, first) {
		if syms[k].Line < first && syms[k].Kind != "h" {
			rows++
		}
	}
	if rows > 2 {
		rows = 2
	}
	return rows
}
