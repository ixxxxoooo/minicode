package app

import (
	"os"
	"strings"

	"github.com/egoist/mygo"
)

// dockIconPath 按偏好 appearance.appIcon* 与深浅色选择图标文件。
func (a *App) dockIconPath(dark bool) string {
	style := a.prefs.Int(pAppStyle, 0)
	mode := a.prefs.Int(pAppMode, 0) // 0 浅色 1 系统 2 深色
	tint := a.prefs.Int(pAppTint, 0)
	if mode == 2 {
		dark = true
	} else if mode == 0 {
		dark = false
	}
	names := []string{
		"MiniCode.icns", "MiniCode-dark.icns",
	}
	if style > 0 || tint > 0 {
		suffix := map[bool]string{true: "-dark", false: ""}[dark]
		cand := []string{
			"app-style" + itoa(style) + suffix + ".icns",
			"app-style" + itoa(style) + suffix + ".png",
			"app-light" + suffix + ".png",
		}
		for _, c := range cand {
			if p := iconAssetPath(c); p != "" {
				return p
			}
		}
	}
	if dark {
		return iconAssetPath(names[1])
	}
	return iconAssetPath(names[0])
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
		_ = mygo.App.Dock.SetIcon(data)
	}
}
