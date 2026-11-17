package app

import (
	"math"
	"testing"
)

func near(a, b float32) bool { return math.Abs(float64(a-b)) < 0.01 }

// 长文件:内容比框高时地图随滚动比例平移,滑块始终在框内,顶 / 底分别贴顶 / 贴底(此前滑块按绝对坐标算会跑出框)。
func TestMinimapGeometryLongFile(t *testing.T) {
	const n, boxH = 6000, float32(1800)
	total, view := float32(n*20), float32(1400) // 编辑器:每行 20px,视口 1400px
	lines := func(scroll float32) (float32, float32) { return scroll / 20, (scroll + view) / 20 }
	at := func(scroll float32) mmGeom {
		l0, l1 := lines(scroll)
		return mmLayout(n, boxH, scroll, view, total, l0, l1)
	}
	g0 := at(0)
	if !near(g0.top, 0) || !near(g0.off, 0) {
		t.Fatalf("滚动在顶:滑块应贴顶、地图不偏移,得 %+v", g0)
	}
	if want := (boxH - mmPad) - g0.h; g0.top+want < 0 {
		t.Fatalf("滑块高度超出框:%+v", g0)
	}
	gEnd := at(total - view)
	if !near(gEnd.top+gEnd.h, gEnd.visH) {
		t.Fatalf("滚动到底:滑块应贴框底,得 top=%v h=%v visH=%v", gEnd.top, gEnd.h, gEnd.visH)
	}
	// 滚动约 20%:滑块在框的约 20% 处(与滚动条同一比例,而不是跑到底部)
	gMid := at((total - view) * 0.2)
	if ratio := gMid.top / (gMid.visH - gMid.h); ratio < 0.18 || ratio > 0.22 {
		t.Fatalf("滚动 20%% 时滑块应在行程的约 20%%,得 %.3f(%+v)", ratio, gMid)
	}
	for _, f := range []float32{0, .1, .37, .5, .9, 1} {
		g := at((total - view) * f)
		if g.top < -0.01 || g.top+g.h > g.visH+0.01 {
			t.Fatalf("f=%v 滑块跑出框:%+v", f, g)
		}
		// 反算:按滑块当前位置回推滚动,应回到原来的滚动(拖动不漂移)
		if got, want := g.scrollFor(g.top, total, view), (total-view)*f; !near(got, want) && math.Abs(float64(got-want)) > 1 {
			t.Fatalf("f=%v 反算滚动 %v ≠ %v", f, got, want)
		}
	}
}

// 短文件:内容比框矮,地图不滚动,滑块按行位置直接落在内容上。
func TestMinimapGeometryShortFile(t *testing.T) {
	const n = 100
	total, view := float32(n*20), float32(600)
	g := mmLayout(n, 1800, 400, view, total, 20, 50)
	if !near(g.off, 0) || !near(g.top, 20*mmRowH) || !near(g.h, 30*mmRowH) {
		t.Fatalf("短文件应不平移、滑块在行位置上:%+v", g)
	}
	if got := g.scrollFor(g.top, total, view); !near(got, 400) {
		t.Fatalf("反算滚动应为 400,得 %v", got)
	}
	// 整篇都在视口里:滑块占满,无可动范围,滚动恒为 0
	full := mmLayout(10, 1800, 0, 600, 200, 0, 10)
	if full.scrollFor(0, 200, 600) != 0 {
		t.Fatalf("整篇可见时不应滚动:%+v", full)
	}
}
