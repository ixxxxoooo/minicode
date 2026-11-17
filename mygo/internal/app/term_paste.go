package app

import "strings"

// termPaste 向终端粘贴;多行时按偏好确认。
func (a *App) termPaste(tp *termPanel, s string) {
	if tp == nil || tp.t == nil {
		return
	}
	s = strings.TrimSuffix(s, "\n")
	if a.termPasteNeedsConfirm(s) {
		a.confirmDlg("粘贴多行文字?", "剪贴板里有多行内容,将一次性送入 shell。", "粘贴", false, func() {
			tp.t.Paste(s)
			tp.focusIt = true
		})
		return
	}
	tp.t.Paste(s)
	tp.focusIt = true
}
