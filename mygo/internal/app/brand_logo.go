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
		// 设计稿 .halo:图标四周外溢 26px 的强调色径向渐变(30% → 透明,closest-side),不占布局
		const spread = 26
		t := a.tk
		ui.Box(c).Size(px, px).Margin(8, 0, 4, 0).Children(func() {
			g := px + 2*spread
			ui.Box(c).Absolute().ZIndex(0).Left(-spread).Top(-spread).Size(g, g).Radius(g / 2).PassThrough().RadialGradient(ui.RadialGradient{
				Radius: g / 2,
				Stops:  []ui.GradientStop{{Color: t.Ac.Alpha(.30), At: 0}, {Color: t.Ac.Alpha(0), At: 1}},
			})
			logoTile(c, src, px, 1)
		})
		return
	}
	ui.Box(c).Size(px, px).Children(func() { logoTile(c, src, px, 0) })
}

// logoTileRatio 应用图标素材里圆角方块占整张画布的比例(macOS 图标规范四周留约 10% 透明边,实测 0.81)。
const logoTileRatio = 0.81

// logoTile 画图标,使可见的圆角方块恰好是 px × px(设计稿 U.brand.logo(px) 的方块撑满 px,
// 直接把整张 PNG 缩到 px 会让可见图标只剩约 80%)。透明边距溢出到容器外,不影响布局。
func logoTile(c *ui.Context, src ui.ImageSource, px float32, z int) {
	s := px / logoTileRatio
	off := -(s - px) / 2
	ui.Image(c, src).Absolute().ZIndex(z).Left(off).Top(off).Size(s, s).Fit(ui.Cover)
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
