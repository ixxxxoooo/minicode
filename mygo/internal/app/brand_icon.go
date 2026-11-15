package app

import (
	"os"
	"strings"

	"github.com/egoist/mygo"
)

// dockIconPath 按偏好 appearance.appIcon* 与无障碍回退选择 Dock 图标文件。
func (a *App) dockIconPath(dark bool) string {
	if a.prefs == nil {
		if dark {
			return iconAssetPath("MiniCode-dark.icns")
		}
		return iconAssetPath("MiniCode.icns")
	}
	style := a.prefs.Int(pAppStyle, 0)
	mode := a.prefs.Int(pAppMode, 1) // 0 自动 1 浅色 2 深色 3 透明 4 着色
	if mode == 2 {
		dark = true
	} else if mode == 1 {
		dark = false
	}
	suffix := map[bool]string{true: "-dark", false: ""}[dark]
	if mode == 3 {
		if a11yReduceTransparency() || a11yIncreaseContrast() {
			suffix = map[bool]string{true: "-dark", false: ""}[dark]
		} else {
			suffix = "-clear"
		}
	} else if mode == 4 {
		if a11yReduceTransparency() || a11yIncreaseContrast() {
			suffix = map[bool]string{true: "-dark", false: ""}[dark]
		} else {
			suffix = "-tint"
		}
	}
	cand := []string{
		"app-style" + itoa(style) + suffix + ".icns",
		"app-style" + itoa(style) + suffix + ".png",
	}
	if style == 2 {
		cand = append(cand, map[bool]string{true: "MiniCode-dark.icns", false: "MiniCode.icns"}[dark])
	}
	cand = append(cand,
		map[bool]string{true: "app-dark.png", false: "app-light.png"}[dark],
		map[bool]string{true: "MiniCode-dark.icns", false: "MiniCode.icns"}[dark],
	)
	for _, c := range cand {
		if p := iconAssetPath(c); p != "" {
			return p
		}
	}
	return ""
}

func (a *App) setDockIconFromPrefs(dark bool) {
	p := a.dockIconPath(dark)
	if p == "" {
		a.setDockIcon(dark)
		return
	}
	data, err := os.ReadFile(p)
	if err != nil {
		a.setDockIcon(dark)
		return
	}
	if strings.HasSuffix(p, ".icns") {
		if png, ok := loadIcnsAsPNG(p); ok {
			data = png
		}
	}
	if len(data) > 0 {
		if err := mygo.App.Dock.SetIcon(data); err != nil {
			brandLog.Warn("设置 Dock 图标失败", "err", err)
		}
	}
}
