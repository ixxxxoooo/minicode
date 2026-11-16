package app

import "testing"

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
