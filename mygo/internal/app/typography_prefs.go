package app

import (
	"github.com/ixxxxoooo/minicode/internal/theme"
)

const (
	pFontPreset = "appearance.fontPreset"
	pLineHeight = "appearance.lineHeight"
	fontCustom  = 4
)

var fontPresetPairs = []struct{ ui, font int }{
	{100, 95},
	{theme.DefaultUIScale, 100}, // 标准 = 两个滑块的默认值,全新安装即在此档
	{120, 110},
	{135, 125},
}

var flowLineHeights = []float32{1.5, 1.65, 1.8}

// FlowReadLH 是会话流正文行距(appearance.lineHeight)。
func (a *App) FlowReadLH() float32 {
	if a == nil || a.prefs == nil {
		return theme.LhRead
	}
	i := clampInt(a.prefs.Int(pLineHeight, 1), 0, len(flowLineHeights)-1)
	return flowLineHeights[i]
}

// syncFontPresetFromSliders 滑块偏离已知组合时记为「自定义」。
func (a *App) syncFontPresetFromSliders() {
	if a == nil || a.prefs == nil {
		return
	}
	ui := a.prefs.Int(pUIScale, theme.DefaultUIScale)
	font := a.prefs.Int(pFontScale, 100)
	for i, p := range fontPresetPairs {
		if ui == p.ui && font == p.font {
			if a.prefs.Int(pFontPreset, 1) != i {
				if i == 1 {
					a.prefs.Reset(pFontPreset)
				} else {
					a.prefs.Set(pFontPreset, i)
				}
			}
			return
		}
	}
	if a.prefs.Int(pFontPreset, fontCustom) != fontCustom {
		a.prefs.Set(pFontPreset, fontCustom)
	}
}

// applyFontPresetChoice 应用字号预设(0–3)。
func (a *App) applyFontPresetChoice(i int) {
	if a == nil || a.prefs == nil || i < 0 || i >= len(fontPresetPairs) {
		return
	}
	p := fontPresetPairs[i]
	if p.ui == theme.DefaultUIScale {
		a.prefs.Reset(pUIScale)
	} else {
		a.prefs.Set(pUIScale, p.ui)
	}
	if p.font == 100 {
		a.prefs.Reset(pFontScale)
	} else {
		a.prefs.Set(pFontScale, p.font)
	}
	if i == 1 {
		a.prefs.Reset(pFontPreset)
	} else {
		a.prefs.Set(pFontPreset, i)
	}
	theme.SetUIScale(p.ui)
}

// onTypographyPref 在 setPref 之后处理排版相关键的联动。
func (a *App) onTypographyPref(key string, v any) {
	switch key {
	case pFontPreset:
		if i, ok := v.(int); ok && i >= 0 && i < len(fontPresetPairs) {
			a.applyFontPresetChoice(i)
		}
	case pUIScale, pFontScale:
		a.syncFontPresetFromSliders()
	}
}
