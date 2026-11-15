package app

import (
	"strings"

	"github.com/ixxxxoooo/minicode/internal/kit"
)

// applyCheapPrefs 把零成本高收益的设置偏好套到运行时(applyBehavior 末尾调用)。
func (a *App) applyCheapPrefs() {
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

// ignoreGlobLines 把偏好 editor.ignore 拆成行(供搜索 / 监视读取)。
func (a *App) ignoreGlobLines() []string {
	raw := a.prefs.Str("editor.ignore", "target\nnode_modules\n.git\ndist")
	var out []string
	for _, ln := range strings.Split(raw, "\n") {
		ln = strings.TrimSpace(ln)
		if ln != "" {
			out = append(out, ln)
		}
	}
	return out
}
