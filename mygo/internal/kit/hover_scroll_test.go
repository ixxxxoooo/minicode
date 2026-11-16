package kit

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// 滚动区里的行必须能悬停:文件树 / 列表行的悬停底色依赖 Hovered()。
func TestRowHoverInsideScroll(t *testing.T) {
	var st ui.ScrollState
	hovered := false
	view := func(c *ui.Context) {
		ui.Box(c).Size(300, 200).Children(func() {
			VScroll(c).Key("sc").TrackScroll(&st).Grow(1).Children(func() {
				ui.Column(c).Children(func() {
					for i := 0; i < 5; i++ {
						r := RowEl(c, false).Key("row" + string(rune('a'+i))).Height(24)
						if i == 1 && r.Hovered() {
							hovered = true
						}
						r.Children(func() { ui.Text(c, "行") })
					}
				})
			})
		})
	}
	tt := ui.NewTester(view, 300, 200)
	tt.Frame()
	tt.Frame()
	tt.Move(50, 36) // 第二行(24 高)
	tt.Frame()
	tt.Frame()
	if !hovered {
		t.Fatal("滚动区里的行没有收到悬停")
	}
}
