package app

import "testing"

func TestFlowFontScale(t *testing.T) {
	a := New(Options{})
	a.prefs.Set(pFontScale, 120)
	if g, w := a.FlowFontSize(), a.ChatFontScale(); g <= 0 || w != 1.2 {
		t.Fatalf("FlowFontSize=%v scale=%v", g, w)
	}
	a.prefs.Set("appearance.density2", 0)
	if a.ToolDetailLevel() != 0 {
		t.Fatalf("ToolDetailLevel=%d", a.ToolDetailLevel())
	}
}
