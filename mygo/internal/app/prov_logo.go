package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/claudecfg"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

// 模型 / 服务商的品牌 logo 辅助(图形与匹配规则见 kit/logos_gen.go)。

// provBrand 服务商的品牌:先按名称,再按域名(不用带路径的地址,DeepSeek 的地址里有 /anthropic)。
func provBrand(name, baseURL string) string {
	return kit.BrandOf(name, claudecfg.HostOf(baseURL))
}

// modelBrand 模型的品牌:先看模型 ID;认不出就沿用所属服务商的品牌(自建网关里认不出的模型显示占位图标)。
func modelBrand(model, provBrand string) string {
	if b := kit.BrandOf(model); b != "" {
		return b
	}
	return provBrand
}

// logoBox 是列表行左侧 26px 方块,里面放品牌 logo(设置里的服务商行)。
func logoBox(a *App, c *ui.Context, brand string) {
	ui.Row(c).Size(26, 26).Radius(a.tk.RSm()).Background(a.tk.Fill1).Justify(ui.Center).AlignItems(ui.Center).Shrink(0).
		Children(func() { kit.Logo(c, brand, 16, a.tk.Tx2) })
}
