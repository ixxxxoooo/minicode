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

// showMainWindow 把主窗口带回前台:窗口被「关闭」(其实是隐藏)之后,Dock / 托盘 / 通知 / 再次打开都要能让它重新出现。
// 只调 mygo.App.Focus() 不够:它只激活应用,不会把已隐藏的窗口显示出来(此前关闭窗口后点 Dock 没反应)。
func (a *App) showMainWindow(why string) {
	if a == nil || a.win == nil {
		mygo.App.Focus()
		return
	}
	if !a.win.IsVisible() {
		a.win.Show()
		log.Info("重新显示主窗口", "原因", why)
	}
	mygo.App.Focus()
	a.win.Focus()
}

// WireReopen 接通「再次打开」:点 Dock 图标(没有可见窗口)/ 应用被激活(⌘Tab、启动器、Manico)时显示主窗口。
func (a *App) WireReopen() {
	if a == nil || a.win == nil || a.secondary {
		return
	}
	mygo.App.OnActivate(func(hasVisibleWindows bool) {
		if !hasVisibleWindows {
			a.showMainWindow("点 Dock 图标")
		}
	})
	mygo.App.OnDidBecomeActive(func() {
		if !a.win.IsVisible() {
			a.showMainWindow("应用被激活")
		}
	})
}
