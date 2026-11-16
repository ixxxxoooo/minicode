package app

import (
	"testing"

	"github.com/ixxxxoooo/minicode/internal/theme"
)

func TestFlowWidthTiersAndLegacyMigrate(t *testing.T) {
	a, _ := sessionApp(t, "flow-gaps-b4")
	a.chatW = 1200
	a.prefs.Set(pWidthLegacy, 0)
	if got := a.flowWidthIndex(); got != 1 {
		t.Fatalf("旧键 0 应迁到紧凑(1),得 %d", got)
	}
	if w := a.flowMaxWidth(); w != 640 {
		t.Fatalf("紧凑档应为 640,得 %v", w)
	}
	a.prefs.Reset(pWidthLegacy)
	a.prefs.Set(pFlowWidth, 4)
	if w := a.flowMaxWidth(); w != 1040 {
		t.Fatalf("特宽档应为 1040,得 %v", w)
	}
	a.prefs.Set(pFlowWidth, flowWidthFull)
	a.chatW = 0 // 没有容器宽度约束
	if w := a.flowMaxWidth(); w != flowWidthCapPx {
		t.Fatalf("占满档在无 chatW 约束时应封顶 %d,得 %v", flowWidthCapPx, w)
	}
	a.chatW = 900
	a.prefs.Set(pFlowWidth, flowWidthFull)
	if w := a.flowMaxWidth(); w != 900-2*20 { // tierWide pad 20
		t.Fatalf("占满应随对话栏收窄,得 %v", w)
	}
}

func TestFlowNarrowContainer(t *testing.T) {
	a, _ := sessionApp(t, "flow-gaps-b4")
	a.prefs.Set(pFlowWidth, 3)
	a.chatW = 500
	a.flowPad() // warm tier
	w := a.flowMaxWidth()
	if w > a.chatW {
		t.Fatalf("列宽 %v 不应超过对话栏 %v", w, a.chatW)
	}
}

func TestSettingsContentWidth(t *testing.T) {
	a, _ := sessionApp(t, "flow-gaps-b4")
	a.winW = 1200
	if w := a.settingsContentWidth(); w != 760 {
		t.Fatalf("默认设置列宽 760,得 %v", w)
	}
	a.prefs.Set(pSetWidth, 0)
	if w := a.settingsContentWidth(); w != 620 {
		t.Fatalf("窄档 620,得 %v", w)
	}
	a.winW = 800
	if w := a.settingsContentWidth(); w != 0 {
		t.Fatalf("窄窗应 0(铺满),得 %v", w)
	}
}

func TestFontPresetApply(t *testing.T) {
	a, _ := sessionApp(t, "flow-gaps-b4")
	a.applyFontPresetChoice(0)
	if a.prefs.Int(pUIScale, theme.DefaultUIScale) != 100 || a.prefs.Int(pFontScale, 100) != 95 {
		t.Fatalf("紧凑预设应 100/95")
	}
	a.applyFontPresetChoice(2)
	if a.prefs.Int(pUIScale, theme.DefaultUIScale) != 120 || a.prefs.Int(pFontScale, 100) != 110 {
		t.Fatalf("舒适预设应 120/110")
	}
	if theme.UIScale() < 1.19 {
		t.Fatalf("uiScale 应随预设更新,得 %v", theme.UIScale())
	}
}

// 预设行点「还原」要把界面大小与正文字号一起还原(此前只还原了预设键本身,界面仍停在放大状态)。
func TestFontPresetResetRestoresScales(t *testing.T) {
	a, _ := sessionApp(t, "flow-gaps-b4")
	a.applyFontPresetChoice(2) // 舒适 120/110
	row := setRow{key: pFontPreset, def: 1}
	a.resetPref(row)
	if got := a.prefs.Int(pUIScale, theme.DefaultUIScale); got != theme.DefaultUIScale {
		t.Fatalf("界面大小应还原为 %d,得 %d", theme.DefaultUIScale, got)
	}
	if got := a.prefs.Int(pFontScale, 100); got != 100 {
		t.Fatalf("正文字号应还原为 100,得 %d", got)
	}
	if a.prefs.Has(pFontPreset) {
		t.Fatal("预设键应回到默认(不落盘)")
	}
	if theme.UIScale() != float32(theme.DefaultUIScale)/100 {
		t.Fatalf("界面缩放应回到默认,得 %v", theme.UIScale())
	}
}
