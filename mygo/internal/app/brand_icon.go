package app

import (
	"os"
	"strings"

	"github.com/egoist/mygo"
)

// dockIconSuffix 按外观偏好(0 自动 1 浅色 2 深色 3 透明 4 着色)与无障碍设置,得出素材后缀与是否深色。
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

// dockIconData 按偏好 appearance.appIcon* 选 Dock 图标,返回 PNG 字节。
// 顺序:内嵌的 512px 样式图 → 磁盘上的样式 icns / PNG → 包内默认织纹。
func (a *App) dockIconData(dark bool) []byte {
	style := 0
	if a.prefs != nil {
		style = a.prefs.Int(pAppStyle, 0)
	}
	if style < 0 || style >= len(appStyles) {
		style = 0
	}
	suffix, dark := a.dockIconSuffix(dark)
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
		brandLog.Warn("没有找到可用的 Dock 图标素材,保持包内默认图标")
		return
	}
	if err := mygo.App.Dock.SetIcon(data); err != nil {
		brandLog.Warn("设置 Dock 图标失败", "err", err)
	}
}
