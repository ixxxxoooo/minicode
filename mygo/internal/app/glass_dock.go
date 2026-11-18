package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

// dockEdge 是输入区背后的渐隐(macOS 26 的 scroll edge 思路):会话内容滚到输入区上方时逐渐淡出,
// 到输入区里只剩淡淡的影子,否则正文会直接透到「描述任务…」这类小字下面,又乱又影响可读。
// 玻璃输入框抓取的背后画面包含它,所以仍能看到内容在玻璃后面被折射,只是不再抢字。
// 流光的壁纸是纯色(窗口底色 t.Chrome),没有额外绘制。
func (a *App) dockEdge(c *ui.Context) {
	t := a.tk
	h := a.dockH + 40
	ui.Box(c).Key("dock-edge").Absolute().Left(0).Right(0).Bottom(0).Height(h).PassThrough().
		LinearGradient(ui.LinearGradient{Angle: 180, Stops: []ui.GradientStop{
			{Color: t.BG.Alpha(0), At: 0}, {Color: t.BG.Alpha(.9), At: 40 / h}, {Color: t.BG.Alpha(.9), At: 1},
		}})
}

// glassIsland 把整页里的一块区域(设置页的导航 / 内容)画成玻璃岛,与主界面的面板一致:
// 方案 B 是圆角 + 渐变高光边 + 玻璃阴影;A / C 面板相接,保留调用方原有的细线。非玻璃主题不做任何事。
func (a *App) glassIsland(el ui.Element, clarity float32, tint ui.Color) {
	t := a.tk
	if !t.Glass() {
		return
	}
	el.Material(kit.GlassPane(clarity, tint)).Clip()
	if t.Scheme.ID != "b" {
		return
	}
	el.Radius(t.RLg())
	kit.GlassRim(el)
	for _, s := range t.GlassShadow() {
		el.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
	}
}
