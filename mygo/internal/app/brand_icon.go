package app

import (
	"os"
	"strings"

	"github.com/egoist/mygo"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// dockIconSuffix 按外观偏好(0 自动 1 浅色 2 深色 3 透明 4 着色)与无障碍设置,得出素材后缀与是否深色。「自动」按传入的 dark(当前主题深浅)走。
func (a *App) dockIconSuffix(dark bool) (suffix string, isDark bool) {
	mode := 0 // 默认「自动」,与设置页一致
	if a.prefs != nil {
		mode = a.prefs.Int(pAppMode, 0)
	}
	switch mode {
	case 2:
		dark = true
	case 1:
		dark = false
	}
	suffix = map[bool]string{true: "-dark", false: ""}[dark]
	reduced := a11yReduceTransparency() || a11yIncreaseContrast() // 透明 / 着色在这两种无障碍设置下退回实色
	switch {
	case mode == 3 && !reduced:
		suffix = "-clear"
	case mode == 4 && !reduced:
		suffix = "-tint"
	}
	return suffix, dark
}

// iconPalOf 把主题 id 换成图标配色名:paper / ink 共用默认橙色(返回空串,素材沿用无前缀的 app-style*.png),
// 其余四套各有一组素材 app-<配色>-style*.png。
func iconPalOf(themeID string) string {
	switch themeID {
	case "mist", "celadon", "graphite", "aurora":
		return themeID
	}
	return ""
}

// appIconPal 当前应用图标的配色:偏好「配色」为 0 = 跟随当前主题,1..6 = 固定为第 N 套主题的配色。
func (a *App) appIconPal() string {
	if a.prefs != nil {
		if i := a.prefs.Int(pAppTint, 0); i >= 1 && i <= len(theme.Palettes) {
			return iconPalOf(theme.Palettes[i-1].ID)
		}
	}
	if a.tk != nil {
		return iconPalOf(a.tk.Palette.ID)
	}
	return iconPalOf(a.pal.ID)
}

// appIconAsset 某样式在配色 pal、素材后缀 suffix 下的内嵌素材名。
func appIconAsset(style int, pal, suffix string) string {
	if pal == "" {
		return "app-style" + itoa(style) + suffix + ".png"
	}
	return "app-" + pal + "-style" + itoa(style) + suffix + ".png"
}

// dockSignature 把决定 Dock 图标的全部输入拼成签名:样式、配色、外观后缀、深浅。
func (a *App) dockSignature(dark bool) string {
	style := 0
	if a.prefs != nil {
		style = a.prefs.Int(pAppStyle, 0)
	}
	suffix, d := a.dockIconSuffix(dark)
	return itoa(style) + "|" + a.appIconPal() + "|" + suffix + "|" + map[bool]string{true: "d", false: "l"}[d]
}

// dockIconData 按偏好 appearance.appIcon* 选 Dock 图标,返回 PNG 字节。
// 顺序:内嵌的当前配色 512px 样式图 → 默认配色的同款 → 磁盘上的样式 icns / PNG → 包内默认图标。
func (a *App) dockIconData(dark bool) []byte {
	style := 0
	if a.prefs != nil {
		style = a.prefs.Int(pAppStyle, 0)
	}
	if style < 0 || style >= len(appStyles) {
		style = 0
	}
	suffix, dark := a.dockIconSuffix(dark)
	if b, ok := embeddedIcon(appIconAsset(style, a.appIconPal(), suffix)); ok {
		return b
	}
	base := "app-style" + itoa(style) + suffix
	if b, ok := embeddedIcon(base + ".png"); ok {
		return b
	}
	for _, c := range []string{
		base + ".icns", base + ".png",
		map[bool]string{true: "MiniCode-dark.icns", false: "MiniCode.icns"}[dark],
		map[bool]string{true: "app-dark.png", false: "app-light.png"}[dark],
	} {
		p := iconAssetPath(c)
		if p == "" {
			continue
		}
		if strings.HasSuffix(p, ".icns") {
			if png, ok := loadIcnsAsPNG(p); ok {
				return png
			}
			continue
		}
		if b, err := os.ReadFile(p); err == nil && len(b) > 0 {
			return b
		}
	}
	return nil
}

func (a *App) setDockIconFromPrefs(dark bool) {
	data := a.dockIconData(dark)
	if len(data) == 0 {
		a.brand.dockState = "失败:没有找到图标素材,保持包内默认图标"
		brandLog.Warn("没有找到可用的 Dock 图标素材,保持包内默认图标")
		return
	}
	if err := mygo.App.Dock.SetIcon(data); err != nil {
		a.brand.dockState = "失败:" + err.Error()
		brandLog.Warn("设置 Dock 图标失败", "err", err)
		return
	}
	a.brand.dockState = "已套用"
}
