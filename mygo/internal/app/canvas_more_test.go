package app

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/canvas"
)

// 画布列表行的「···」必须左键点击就弹出菜单(原先只挂了右键菜单,点了没反应);实机发现。
func TestCanvasListMoreOpensOnClick(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	a.demoExtra = true // 用演示画布库,不碰 ~/.minicode/canvases
	a.screen = ScreenCanvas
	// 演示库只有一张画布时不显示列表:再建一张
	if _, _, err := a.cvLib().New("第二张", canvas.Doc{Title: "第二张", Status: "草稿"}, "你", "新建", a.now()); err != nil {
		t.Fatal(err)
	}
	tt := ui.NewTester(a.View, 1280, 800)
	settle(tt)
	// 行的无障碍项把内部按钮合并进了行名(「标题 时间 更多」),按行右端「···」的位置点
	clicked := false
	for _, in := range tt.AccessAll() {
		if strings.HasSuffix(in.Name, " 更多") && in.Bounds.W > 0 && in.Bounds.X > 200 && in.Bounds.X < 500 {
			tt.ClickAt(in.Bounds.X+in.Bounds.W-18, in.Bounds.Y+in.Bounds.H/2)
			clicked = true
			break
		}
	}
	if !clicked {
		t.Fatal("找不到画布行")
	}
	tt.Frame()
	menu := strings.Join(tt.Menu(), "|")
	for _, want := range []string{"重命名", "删除"} {
		if !strings.Contains(menu, want) {
			t.Fatalf("画布行「···」菜单缺「%s」:%s", want, menu)
		}
	}
}
