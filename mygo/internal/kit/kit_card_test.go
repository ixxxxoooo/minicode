package kit

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// Card 在无窗口 Tester 里悬停态难以可靠复现(指针悬停不会反映到 Bounds),
// 这里只验证开启动效时渲染不崩;上浮逻辑见 kit_card.go 的 Translate(0,-1)。
func TestCardRenderMotionOn(t *testing.T) {
	SetMotion(true)
	defer SetMotion(true)
	view := func(c *ui.Context) {
		ui.Column(c).Padding(12).Children(func() {
			Card(c).Width(200).Children(func() {
				ui.Text(c, "卡片")
			})
		})
	}
	ui.NewTester(view, 320, 120).Frame()
	ui.NewTester(view, 320, 120).Frame()
	SetMotion(false)
	ui.NewTester(view, 320, 120).Frame()
}
