package app

import (
	"testing"

	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 改应用图标样式 / 外观 / 深浅,都必须让 Dock 图标签名变化(此前只比较深浅,改样式不生效)。
func TestDockSignatureChanges(t *testing.T) {
	a, _ := newTestApp(t)
	base := a.dockSignature(false)
	a.prefs.Set(pAppStyle, 2)
	if a.dockSignature(false) == base {
		t.Fatal("改图标样式后签名应变化")
	}
	s2 := a.dockSignature(false)
	a.prefs.Set(pAppMode, 3)
	if a.dockSignature(false) == s2 {
		t.Fatal("改图标外观后签名应变化")
	}
	a.prefs.Set(pAppMode, 0)
	if a.dockSignature(true) == a.dockSignature(false) {
		t.Fatal("深浅不同签名应不同")
	}
}

// 应用图标配色默认跟随主题:切主题、固定配色都必须改变签名与素材名(此前配色偏好只存不用)。
func TestAppIconFollowsTheme(t *testing.T) {
	a, _ := newTestApp(t)
	a.prefs.Reset(pAppTint)
	for _, tc := range []struct{ theme, pal string }{{"paper", ""}, {"ink", ""}, {"mist", "mist"}, {"celadon", "celadon"}, {"graphite", "graphite"}, {"aurora", "aurora"}} {
		for _, p := range theme.Palettes {
			if p.ID == tc.theme {
				a.pal = p
			}
		}
		a.retheme()
		if got := a.appIconPal(); got != tc.pal {
			t.Fatalf("主题 %s 的图标配色应为 %q,得到 %q", tc.theme, tc.pal, got)
		}
		if !embeddedHas(a.appIconName(0)) {
			t.Fatalf("主题 %s 的图标素材缺失:%s", tc.theme, a.appIconName(0))
		}
	}
	// 固定为第 3 套(青瓷)后不再随主题变
	a.prefs.Set(pAppTint, 3)
	sig := a.dockSignature(false)
	for _, p := range theme.Palettes {
		if p.ID == "aurora" {
			a.pal = p
		}
	}
	a.retheme()
	if a.dockSignature(false) != sig || a.appIconPal() != "celadon" {
		t.Fatalf("固定配色后不应随主题变化:%s", a.appIconPal())
	}
}
