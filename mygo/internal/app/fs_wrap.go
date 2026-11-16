package app

import (
	"github.com/ixxxxoooo/minicode/internal/workspace"
)

// wrapIDE 为 ide.fs 套上可读写工作区外绝对路径的包装(自定义页「在编辑器打开」等)。
func (a *App) wrapIDE(fs workspace.FS) workspace.FS {
	if fs == nil {
		return fs
	}
	return workspace.NewWithExternal(fs)
}
