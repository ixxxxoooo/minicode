package app

import (
	"strings"
	"testing"
)

// 演示模式打开设置页,不能把假的快捷键写进(共用的)偏好里。
func TestDemoSettingsDoNotSeedShortcutPrefs(t *testing.T) {
	t.Setenv("MINI_SHORTCUT_DEMO", "")
	a, tt := newTestApp(t)
	a.openSettings("shortcuts")
	tt.Frame()
	for _, k := range a.prefs.Keys() {
		if strings.HasPrefix(k, "shortcuts.") {
			t.Fatalf("不应预置快捷键偏好,却出现 %s", k)
		}
	}
}
