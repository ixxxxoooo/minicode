package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// 删除最后一张画布后整页必须显示空态,不能空指针崩溃(用户实测:删除画布闪退)。
func TestCanvasDeleteLastNoPanic(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	a.demoExtra = true
	a.screen = ScreenCanvas
	l := a.cvLib()
	for _, id := range append([]string(nil), l.IDs()...) {
		if err := l.Delete(id); err != nil {
			t.Log("删除文件:", err)
		}
	}
	a.canv = nil
	tt := ui.NewTester(a.View, 1280, 800)
	settle(tt)
	if !tt.HasText("还没有画布") {
		t.Fatal("没有画布时应显示空态")
	}
	// 空态里的「新建画布」能建出一张并回到正常页面
	a.newBlankCanvas()
	tt.Frame()
	if tt.HasText("还没有画布") {
		t.Fatal("新建后不应再是空态")
	}
	// 窄窗口同样不崩
	tt.SetSize(700, 600)
	tt.Frame()
}
