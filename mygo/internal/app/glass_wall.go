package app

import "github.com/egoist/mygo/ui"

// glassWall 把窗口根画成流光的壁纸:纵向两色底 + 四团柔和的色晕(设计稿 --wall)。
// 壁纸是静态的,整窗只在重绘时画这几笔;玻璃面板抓取的「背后」就是它,
// 所以面板之间互不重叠、背后只放壁纸,会话流式输出与终端刷新都不会牵动面板的模糊。
func (a *App) glassWall(root ui.Element) {
	t := a.tk
	top, bottom, blobs := t.Wall()
	root.Background(bottom)
	root.Draw(func(p *ui.Painter, r ui.Rect) {
		p.FillGradient(r, ui.LinearGradient{Angle: 180, Stops: []ui.GradientStop{{Color: top, At: 0}, {Color: bottom, At: 1}}}, 0)
		for _, b := range blobs {
			p.FillRadialGradient(r, ui.RadialGradient{
				Center: ui.Point{X: b.X, Y: b.Y},
				Radius: b.K * (r.W + r.H) / 2,
				Stops:  []ui.GradientStop{{Color: b.Color, At: 0}, {Color: b.Color.Alpha(0), At: 1}},
			}, 0)
		}
	})
}
