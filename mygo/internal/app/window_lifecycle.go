package app

import (
	"github.com/egoist/mygo"
)

const pCloseWin = "general.close"

// CloseHides 红色关闭按钮是否隐藏窗口而非退出。
func (a *App) CloseHides() bool {
	if a == nil || a.prefs == nil {
		return true
	}
	return a.prefs.Int(pCloseWin, 0) == 0
}

// WireMainWindowClose 接通 general.close:隐藏时保持进程与会话运行,Dock 再点可 Show。
func (a *App) WireMainWindowClose(win *mygo.Window) {
	if a == nil || win == nil || a.secondary {
		return
	}
	win.OnClose(func(e *mygo.CloseEvent) {
		if a.CloseHides() {
			e.PreventDefault()
			win.Hide()
			log.Info("关闭窗口", "action", "隐藏")
		} else {
			log.Info("关闭窗口", "action", "退出")
		}
	})
}
