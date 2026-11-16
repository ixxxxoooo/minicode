package app

import (
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func approx(a, b float32) bool { return math.Abs(float64(a-b)) < 1e-3 }

// 缩放以光标为中心:光标下的内容点在缩放前后屏幕位置不变。
func TestZoomAtKeepsPointUnderCursor(t *testing.T) {
	scale, px, py := float32(1), float32(30), float32(20)
	cx, cy := float32(200), float32(150)
	// 光标下的内容坐标
	ux, uy := (cx-px)/scale, (cy-py)/scale
	ns, nx, ny := zoomAt(scale, px, py, cx, cy, 2, 0.2, 6)
	if !approx(ns, 2) {
		t.Fatalf("缩放应为 2,得到 %v", ns)
	}
	if !approx(nx+ux*ns, cx) || !approx(ny+uy*ns, cy) {
		t.Fatalf("光标下的点漂移了:(%v,%v) 期望 (%v,%v)", nx+ux*ns, ny+uy*ns, cx, cy)
	}
}

// 到上下限后不再改平移,也不越界。
func TestZoomAtClamps(t *testing.T) {
	ns, nx, ny := zoomAt(6, 10, 10, 50, 50, 2, 0.2, 6)
	if ns != 6 || nx != 10 || ny != 10 {
		t.Fatalf("已到上限应原样返回,得到 %v %v %v", ns, nx, ny)
	}
	ns, _, _ = zoomAt(0.25, 0, 0, 0, 0, 0.1, 0.2, 6)
	if !approx(ns, 0.2) {
		t.Fatalf("应夹到下限 0.2,得到 %v", ns)
	}
	// 非法输入不动
	if s, x, y := zoomAt(0, 1, 2, 3, 4, 2, 0.2, 6); s != 0 || x != 1 || y != 2 {
		t.Fatal("scale<=0 应原样返回")
	}
}

func TestWheelFactorDirectionAndCap(t *testing.T) {
	if wheelFactor(-40) <= 1 {
		t.Fatal("向上滚应放大")
	}
	if wheelFactor(40) >= 1 {
		t.Fatal("向下滚应缩小")
	}
	if wheelFactor(-100000) != 2 || wheelFactor(100000) != 0.5 {
		t.Fatal("单次缩放倍数应夹在 [0.5, 2]")
	}
}

// 输入处理:鼠标滚轮缩放并退出适应模式,触控板两指滑动平移,捏合缩放,无关事件不处理。
func TestViewerInputKinds(t *testing.T) {
	fit, scale, px, py := true, float32(1), float32(0), float32(0)
	h := viewerInput(&fit, &scale, &px, &py, 0.2, 6)

	if h(ui.InputEvent{Kind: ui.InputKeyDown}) {
		t.Fatal("按键不该被查看器画布吃掉")
	}
	if h(ui.InputEvent{Kind: ui.InputScroll, DY: 0}) {
		t.Fatal("零增量滚轮不处理")
	}
	if !h(ui.InputEvent{Kind: ui.InputScroll, DY: -40, X: 100, Y: 100}) || fit || scale <= 1 {
		t.Fatalf("滚轮向上应放大并退出适应:fit=%v scale=%v", fit, scale)
	}
	s0, px0, py0 := scale, px, py
	if !h(ui.InputEvent{Kind: ui.InputScroll, DX: 10, DY: 20, Precise: true}) {
		t.Fatal("触控板滑动应被处理")
	}
	if scale != s0 || !approx(px, px0-10) || !approx(py, py0-20) {
		t.Fatalf("触控板滑动应只平移:scale=%v pan=(%v,%v)", scale, px, py)
	}
	if !h(ui.InputEvent{Kind: ui.InputScroll, DY: -40, Precise: true, Mods: ui.Cmd}) || scale <= s0 {
		t.Fatal("触控板 + ⌘ 滚动应缩放")
	}
	s1 := scale
	if !h(ui.InputEvent{Kind: ui.InputZoom, Scale: 0.5}) || !approx(scale, s1*0.5) {
		t.Fatalf("捏合应按比例缩放:%v → %v", s1, scale)
	}
	if h(ui.InputEvent{Kind: ui.InputZoom, Scale: 0}) {
		t.Fatal("非法捏合比例不处理")
	}
}

// 界面级:全屏 Mermaid 里真的滚轮,缩放变大且退出「适应」;底栏写明滚轮缩放。
func TestMermaidViewerWheelZoom(t *testing.T) {
	a, tt := newTestApp(t)
	a.openMmViewer("流程图", "graph TD\n A[开始] --> B[结束]")
	tt.Frame()
	tt.Advance(300 * time.Millisecond)
	tt.Frame()
	if !a.mmView.open {
		t.Skip("测试环境没能构建出 Mermaid 场景(后台排版未就绪)")
	}
	if !tt.HasText("滚轮缩放") {
		t.Fatal("底栏应写明「滚轮缩放」")
	}
	if !a.mmView.fit {
		t.Fatal("刚打开应是适应模式")
	}
	before := a.mmView.scale
	tt.Wheel(640, 400, 0, -80)
	tt.Frame()
	if a.mmView.fit {
		t.Fatal("滚轮后应退出适应模式")
	}
	if a.mmView.scale <= before {
		t.Fatalf("滚轮向上应放大:%v → %v", before, a.mmView.scale)
	}
}
