package app

import (
	_ "embed"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

//go:embed assets/icon/app-logo.png
var brandLogoPNG []byte

// brandLogo 在界面中绘制应用图标(欢迎页、关于、空状态等);失败时退回 kit mark。
func (a *App) brandLogo(c *ui.Context, px float32, glow bool) {
	src, err := ui.DecodeImage(brandLogoPNG)
	if err != nil || src == nil {
		kit.Ic(c, "mark", px).TextColor(a.tk.Ac)
		return
	}
	if glow && px >= 48 {
		ui.Row(c).Size(px+12, px+12).Radius(px/2 + 6).Background(a.tk.AcSoft).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			ui.Image(c, src).Size(px, px).Fit(ui.Cover)
		})
		return
	}
	ui.Image(c, src).Size(px, px).Fit(ui.Cover)
}
