package app

// formatSelection 格式化选定内容(仅语言服务器;无选区时提示)。
func (a *App) formatSelection() {
	tb := a.ide.cur()
	if tb == nil || tb.st == nil || tb.buf == nil {
		return
	}
	sel := tb.st.Selection()
	a0, a1 := sel.Anchor, sel.Head
	if a1 < a0 {
		a0, a1 = a1, a0
	}
	if a0 == a1 {
		a.toast = "先选中要格式化的内容"
		return
	}
	if why := a.lspEditorWhy(tb); why != "" {
		a.toast = why
		return
	}
	// 暂无区间格式化 RPC:整篇格式化并说明。
	a.formatDoc()
	a.toast = "此服务器只支持整篇格式化,已格式化整个文档"
}
