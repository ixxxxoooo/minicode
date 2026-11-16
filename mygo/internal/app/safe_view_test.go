package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// 一块界面构建 panic:只这一块换成出错卡片,兄弟块照常显示;点「重试」后恢复。
func TestGuardIsolatesPanic(t *testing.T) {
	a := New(Options{Mode: ModeIDE})
	a.guardAlways = true
	boom := true
	view := func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			ui.Text(c, "旁边的块")
			a.guard(c, "坏块", "bad", guardCard, func() {
				ui.Column(c).Children(func() {
					if boom {
						panic("故意的")
					}
					ui.Text(c, "好了")
				})
			})
		})
	}
	tt := ui.NewTester(view, 600, 400)
	tt.Frame()
	tt.Frame()
	tt.Frame()
	if !tt.HasText("旁边的块") {
		t.Fatalf("兄弟块丢失: %v", tt.Texts())
	}
	if !tt.HasText("「坏块」出错了") {
		t.Fatalf("应显示出错卡片: %v", tt.Texts())
	}
	if a.faults["bad"] == nil {
		t.Fatal("故障应被记录")
	}
	boom = false
	tt.Click("重试")
	tt.Frame()
	tt.Frame()
	if !tt.HasText("好了") || a.faults["bad"] != nil {
		t.Fatalf("重试后应恢复: %v", tt.Texts())
	}
}

// 测试环境(没有窗口)默认不吞 panic,免得把真实缺陷藏起来。
func TestGuardRepanicsInTests(t *testing.T) {
	a := New(Options{Mode: ModeIDE})
	defer func() {
		if recover() == nil {
			t.Fatal("没有窗口时应原样抛出")
		}
	}()
	tt := ui.NewTester(func(c *ui.Context) {
		a.guard(c, "x", "x", guardCard, func() { panic("boom") })
	}, 200, 100)
	tt.Frame()
}
