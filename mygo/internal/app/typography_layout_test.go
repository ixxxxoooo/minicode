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
	if a.prefs.Int(pUIScale, theme.DefaultUIScale) != 100 || a.prefs.Int(pFontScale, 100) != 100 {
		t.Fatalf("紧凑预设应 100/100")
	}
	a.applyFontPresetChoice(2)
	if a.prefs.Int(pUIScale, theme.DefaultUIScale) != 120 || a.prefs.Int(pFontScale, 100) != 118 {
		t.Fatalf("舒适预设应 120/118")
	}
	if theme.UIScale() < 1.19 {
		t.Fatalf("uiScale 应随预设更新,得 %v", theme.UIScale())
	}
}
