package app

import "github.com/egoist/mygo/ui"

// glassWall 把窗口根画成流光的壁纸:纵向两色底 + 几团低饱和色晕 + 一道斜向高光带(设计稿 --wall)。
// 壁纸是静态的,整窗只在重绘时画这几笔;玻璃面板抓取的「背后」就是它,
// 所以面板之间互不重叠、背后只放壁纸,会话流式输出与终端刷新都不会牵动面板的模糊。
// 高光带是为折射准备的:玻璃边缘把它弯出一道弧,才看得出是「玻璃」而不是一层白膜。
func (a *App) glassWall(root ui.Element) {
	w := a.tk.Wall()
	root.Background(w.Bottom)
	root.Draw(func(p *ui.Painter, r ui.Rect) {
		p.FillGradient(r, ui.LinearGradient{Angle: 180, Stops: []ui.GradientStop{{Color: w.Top, At: 0}, {Color: w.Bottom, At: 1}}}, 0)
		for _, b := range w.Blobs {
			p.FillRadialGradient(r, ui.RadialGradient{
				Center: ui.Point{X: b.X, Y: b.Y},
				Radius: b.K * (r.W + r.H) / 2,
				Stops:  []ui.GradientStop{{Color: b.Color, At: 0}, {Color: b.Color.Alpha(0), At: 1}},
			}, 0)
		}
		// 斜向高光带:118° 方向,中心在 50%,两侧各 16% 渐隐
		p.FillGradient(r, ui.LinearGradient{Angle: 118, Stops: []ui.GradientStop{
			{Color: w.Streak.Alpha(0), At: .34}, {Color: w.Streak, At: .5}, {Color: w.Streak.Alpha(0), At: .66},
		}}, 0)
	})
}

// dockEdge 是输入区背后的渐隐(macOS 26 的 scroll edge 思路):会话内容滚到输入区上方时逐渐淡出,
// 到输入区里只剩淡淡的影子,否则正文会直接透到「描述任务…」这类小字下面,又乱又影响可读。
// 玻璃输入框抓取的背后画面包含它,所以仍能看到内容在玻璃后面被折射,只是不再抢字。
func (a *App) dockEdge(c *ui.Context) {
	t := a.tk
	h := a.dockH + 40
	ui.Box(c).Key("dock-edge").Absolute().Left(0).Right(0).Bottom(0).Height(h).PassThrough().
		LinearGradient(ui.LinearGradient{Angle: 180, Stops: []ui.GradientStop{
			{Color: t.BG.Alpha(0), At: 0}, {Color: t.BG.Alpha(.72), At: 40 / h}, {Color: t.BG.Alpha(.72), At: 1},
		}})
}
