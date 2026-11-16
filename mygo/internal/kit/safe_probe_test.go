package kit

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// 探针:在嵌套 Children 里 recover 一个 panic,后续兄弟节点与下一帧是否仍正常。
func TestRecoverInsideChildren(t *testing.T) {
	boom := true
	view := func(c *ui.Context) {
		ui.Column(c).Size(300, 200).Children(func() {
			ui.Text(c, "前")
			func() {
				defer func() { _ = recover() }()
				ui.Column(c).Key("bad").Children(func() {
					ui.Text(c, "坏块里")
					ui.Row(c).Children(func() {
						if boom {
							panic("测试 panic")
						}
						ui.Text(c, "好了")
					})
				})
			}()
			ui.Text(c, "后")
		})
	}
	tt := ui.NewTester(view, 300, 200)
	tt.Frame()
	tt.Frame()
	if !tt.HasText("前") || !tt.HasText("后") {
		t.Fatalf("recover 之后兄弟节点丢失: %v", tt.Texts())
	}
	boom = false
	tt.Frame()
	tt.Frame()
	if !tt.HasText("好了") || !tt.HasText("后") {
		t.Fatalf("恢复后画面不对: %v", tt.Texts())
	}
}
