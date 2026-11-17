package app

import "github.com/egoist/mygo/ui/editor"

// 迷你地图的几何,按 VS Code / Monaco 的做法:
//   - 地图按固定行高(mmRowH)把整篇画出来,内容总高 hc = 行数 × 行高;
//   - 内容比框矮:不滚动,滑块按行位置直接落在内容上;
//   - 内容比框高:地图内容随编辑器的滚动比例 f 整体上移 off = f × (hc − 框高),
//     滑块始终在框内:滚到顶贴顶、滚到底贴底(此前滑块按绝对坐标算,长文件会跑出框外、和滚动条对不上);
//   - 滑块高度 = 视口覆盖的行数 × 行高(软换行时按「可见行」折算,不是按像素比例)。
//
// 拖动 / 点击用滑块的「抓取偏移」反算滚动比例:点在滑块上就保持抓住的那一点,点在空白处把滑块中心移到指针。
//
// @author ixxxxoooo

const mmPad = 4 // 地图内容距框顶的留白

// mmGeom 是某一帧迷你地图的几何(坐标都是框内、已去掉 mmPad 的内容坐标)。
type mmGeom struct {
	hc   float32 // 整篇内容的总高 = 行数 × 行高
	visH float32 // 地图可用高度 = min(内容总高, 框高 − 留白)
	off  float32 // 内容整体上移的距离(内容比框高时随滚动比例变化)
	f    float32 // 编辑器滚动比例 0..1
	top  float32 // 滑块顶
	h    float32 // 滑块高
}

func mmClamp01(v float32) float32 { return min(max(v, 0), 1) }

// mmLayout 由编辑器的滚动状态算出地图几何(纯函数,便于测试)。
// 参数:n 行数;boxH 地图框高;scroll / view / total 编辑器滚动位置 / 视口高 / 内容总高(像素);
// l0 / l1 视口上下沿对应的行号(可带小数,软换行时是折算后的行位置)。
func mmLayout(n int, boxH, scroll, view, total, l0, l1 float32) mmGeom {
	g := mmGeom{hc: mmRowH * float32(n)}
	g.visH = min(g.hc, max(boxH-mmPad, 0))
	if span := total - view; span > 0 {
		g.f = mmClamp01(scroll / span)
	}
	g.off = g.f * max(g.hc-g.visH, 0)
	g.h = min(max((l1-l0)*mmRowH, 8), g.visH)
	g.top = min(max(l0*mmRowH-g.off, 0), max(g.visH-g.h, 0))
	return g
}

// scrollFor 由滑块顶的位置反算编辑器应滚到哪(像素);滑块没有可动范围时返回 0。
func (g mmGeom) scrollFor(top, total, view float32) float32 {
	travel := g.visH - g.h
	if travel <= 0 || total <= view {
		return 0
	}
	return mmClamp01(top/travel) * (total - view)
}

// mmLineAt 把编辑器内容坐标 y(像素)换成行位置(带小数):在第几行的几分之几。
// 折叠行不占高度,二分找「顶边不超过 y 的最后一行」。
func mmLineAt(st *editor.State, n int, y float64) float32 {
	if n <= 0 || y <= 0 {
		return 0
	}
	if y >= st.LineTop(n) {
		return float32(n)
	}
	lo, hi := 0, n-1
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if st.LineTop(mid) <= y {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	top, next := st.LineTop(lo), st.LineTop(lo+1)
	if next <= top {
		return float32(lo)
	}
	return float32(lo) + float32((y-top)/(next-top))
}

// mmGeometry 取当前标签地图的几何。boxH 是地图框的高度(上一帧量到的,首帧为 0 时地图不滚动)。
func mmGeometry(st *editor.State, n int, boxH float32) mmGeom {
	total := float32(st.LineTop(n))
	view := float32(st.ViewportHeight())
	scroll := st.ScrollY()
	return mmLayout(n, boxH, float32(scroll), view, total,
		mmLineAt(st, n, scroll), mmLineAt(st, n, scroll+float64(view)))
}
