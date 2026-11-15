package app

// searchPreviewSide 在右侧拆分组以预览标签打开匹配处。
func (a *App) searchPreviewSide(mt searchMatch) {
	a.openSide(mt.Path, true)
	a.ide.openAt(mt.Path, mt.Line, mt.Col, mt.End, false)
	d := a.ide
	if d.splitDir == 0 {
		a.splitEditor(1)
	}
	d.paneSel = 1
}
