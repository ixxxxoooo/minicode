package app

import (
	"github.com/egoist/mygo/ui/editor"

	"github.com/ixxxxoooo/minicode/internal/diag"
)

// 代码折叠(设计稿 ide-nav「折叠代码」):折叠箭头、占位胶囊、快捷键由编辑器库提供,
// 这里补右键菜单与命令面板入口,并在偏好关闭折叠时说明置灰原因。

var foldLog = diag.L("ide-fold")

// foldRun 对当前编辑器执行一条折叠命令。
func (a *App) foldRun(tb *ideTab, cmd editor.Command, what string) {
	if tb == nil || tb.st == nil {
		return
	}
	if tb.st.NoFolding {
		a.toast = "代码折叠已在设置 → 编辑器里关闭"
		return
	}
	ok := tb.st.Run(cmd)
	foldLog.Info("折叠操作", "path", tb.path, "动作", what, "生效", ok)
	if !ok {
		a.toast = "光标所在位置没有可折叠的块"
	} else {
		a.ide.saveFolds(tb)
	}
	a.redraw()
}

// foldWhy 返回折叠不可用的原因(可用为空)。
func (a *App) foldWhy(tb *ideTab) string {
	switch {
	case tb == nil || tb.st == nil:
		return "没有打开的文件"
	case tb.st.NoFolding:
		return "代码折叠已在设置 → 编辑器里关闭"
	}
	return ""
}
