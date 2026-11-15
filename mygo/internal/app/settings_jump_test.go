package app

import "testing"

func TestSettingsScrollY(t *testing.T) {
	a, _, _ := settingsApp(t, "appearance")
	y := a.settingsScrollY("appearance", "appearance.fontScale")
	if y <= 0 {
		t.Fatalf("应找到字号行偏移,得到 %v", y)
	}
}
