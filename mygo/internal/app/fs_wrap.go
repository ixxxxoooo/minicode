package app

import (
	"github.com/ixxxxoooo/minicode/internal/workspace"
)

// wrapIDE 真实模式下为 ide.fs 套上可读写工作区外绝对路径的包装;演示 Mem 不套。
func (a *App) wrapIDE(fs workspace.FS) workspace.FS {
	if a.demoWs() || fs == nil {
		return fs
	}
	return workspace.NewWithExternal(fs)
}
