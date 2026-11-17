package app

import (
	"github.com/ixxxxoooo/minicode/internal/kit"
)

// applyCheapPrefs 把零成本高收益的设置偏好套到运行时(applyBehavior 末尾调用)。
func (a *App) applyCheapPrefs() {
	kit.WaitSpinStyle.Store(int32(a.prefs.Int(pWaitAnim, 0)))
	a.ApplyDiagPrefs()
	a.ApplySystemAppearance()
	// 终端回看行数等仅影响新终端,此处只记日志
	if a.prefs.Has("term.scrollback") {
		setLog.Debug("终端回看行数偏好已更新", "scrollback", a.prefs.Str("term.scrollback", ""))
	}
	// 搜索 / 监视忽略:editor.ignore 在搜索模块读取(见 ide_search)
	if a.prefs.Has("editor.ignore") {
		setLog.Debug("忽略规则偏好已更新")
	}
	// 括号与语法类偏好:编辑器接线待 B5;动效已处理
	_ = kit.MotionOn()
}
