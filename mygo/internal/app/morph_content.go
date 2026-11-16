package app

import (
	"github.com/egoist/mygo/ui"
)

// morphOpacity 形态切换时各轨内容的透明度。
// 传送带式平移(morph_belt.go)里旧 / 新工作区、对话栏都是刚性整体平移,不再各自淡入淡出,所以恒为 1;
// 导航栏内容换成文件树 / 会话 = 左栏的 a-swap 入场(side.go + kit.SwapIn),也不在这里处理。
func (a *App) morphOpacity(track string) float32 { return 1 }

// morphWrap 包住形态切换时要分轨处理的内容区(目前只是一个占满的容器)。
func (a *App) morphWrap(c *ui.Context, track string, inner func()) {
	ui.Column(c).Grow(1).MinHeight(0).Children(inner)
}
