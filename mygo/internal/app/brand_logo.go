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
	src := a.brandLogoSource(px)
	if src == nil {
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

// brandLogoSource 品牌小标跟随「设置 → 外观 → 应用图标」的样式与外观;小尺寸(≤ 28px)的织纹自动换成 2×2「经纬」。
// 内嵌素材缺失时退回随包的 app-logo.png。
func (a *App) brandLogoSource(px float32) ui.ImageSource {
	style := 0
	if a.prefs != nil {
		style = a.prefs.Int(pAppStyle, 0)
	}
	if style < 0 || style >= len(appStyles) {
		style = 0
	}
	if style == 0 && px <= 28 {
		style = 1
	}
	if src := brandImage(a.appIconName(style), false); src != nil {
		return src
	}
	if src, err := ui.DecodeImage(brandLogoPNG); err == nil {
		return src
	}
	return nil
}
