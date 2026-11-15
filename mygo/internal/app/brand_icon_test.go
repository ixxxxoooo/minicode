package app

import (
	"testing"

	"github.com/ixxxxoooo/minicode/internal/prefs"
)

func TestDockIconPathFallback(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true, Prefs: prefs.Memory()})
	a.prefs.Set(pAppStyle, 9)
	p := a.dockIconPath(false)
	if p == "" {
		t.Fatal("应有默认图标路径")
	}
	if !endsWithAny(p, "MiniCode.icns", "app-light.png") {
		t.Fatalf("意外路径 %s", p)
	}
}

func endsWithAny(p string, suffixes ...string) bool {
	for _, s := range suffixes {
		if len(p) >= len(s) && p[len(p)-len(s):] == s {
			return true
		}
	}
	return false
}

func TestTrayA11yLabel(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true, Prefs: prefs.Memory()})
	l := a.trayA11yLabel(traySum{pending: 1, running: 2})
	if l != "MiniCode,1 个待确认,2 个运行中" {
		t.Fatalf("标签不对: %q", l)
	}
}
