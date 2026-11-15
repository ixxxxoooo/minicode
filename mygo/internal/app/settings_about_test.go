package app

import "testing"

// 关于页:从未检查过更新时不能写「已是最新」。
func TestAboutNotCheckedYet(t *testing.T) {
	a, tt := newTestApp(t)
	a.updSt = updState{}
	a.openSettings("about")
	tt.Frame()
	if tt.HasText("已是最新") {
		t.Fatal("还没检查过更新,不应显示「已是最新」")
	}
	if !tt.HasText("尚未检查更新") {
		t.Fatalf("应提示尚未检查更新,实际: %v", tt.Texts())
	}
}
