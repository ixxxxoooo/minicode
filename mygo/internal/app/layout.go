package app

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 栏宽 / 面板高度的拖拽调整(对标 VS Code 的 Sash:悬停出现强调色细线、拖动实时生效、双击恢复默认、
// 松手后写偏好,下次启动沿用;所有尺寸都夹在合理范围内,窗口变小时自动让位)。
//
//	side   导航栏宽度            180 – 420   默认 228(方案 C 212)
//	ws     Agent 形态右侧工作区宽度 360 – 窗口宽 − 导航栏 − 360   默认 560
//	dock   IDE 形态右侧对话栏宽度  300 – 560   默认 352
//	panel  底部面板高度           96 – 工作区高度 − 120   默认 172(问题 / 引用 236)
const (
	lySide  = "side"
	lyWs    = "ws"
	lyDock  = "dock"
	lyPanel = "panel"
)

// layoutSizes 是用户拖出来的尺寸(0 = 用默认值)。
type layoutSizes struct {
	side, ws, dock, panel float32
	resizing              string    // 正在拖的把手(拖动期间面板的宽度过渡关闭,否则会跟不上鼠标)
	lastDrag              time.Time // 最近一次拖动(松手后几帧内仍不开过渡)
	panelMax              bool      // 底部面板最大化(占满工作区)
}

func (a *App) loadLayout() {
	get := func(k string) float32 { return float32(a.prefs.Int("layout."+k, 0)) }
	a.ly.side, a.ly.ws, a.ly.dock, a.ly.panel = get(lySide), get(lyWs), get(lyDock), get(lyPanel)
}

func orDef(v, def float32) float32 {
	if v <= 0 {
		return def
	}
	return v
}

// sideW / wsRW / dockW / panelHeight 取当前宽高(已夹在范围内)。
func (a *App) sideW() float32 {
	return clampF(orDef(a.ly.side, a.sideWidth()), 180, 420)
}

func (a *App) wsRW() float32 {
	return clampF(orDef(a.ly.ws, theme.WWs), 360, max(360, a.winW-a.sideW()-360))
}

func (a *App) dockW() float32 { return clampF(orDef(a.ly.dock, theme.WDock), 300, 560) }

func (a *App) panelHeight() float32 {
	def := float32(172)
	if a.ide.bottomTab == "problems" || a.ide.bottomTab == "refs" {
		def = 236
	}
	hi := max(120, a.winH-theme.HBar-theme.HTab-120)
	return clampF(orDef(a.ly.panel, def), 96, hi)
}

func clampF(v, lo, hi float32) float32 { return min(max(v, lo), hi) }

// mt 是面板宽高变化的过渡:拖动调整期间关掉(跟手),其余时候用 440 ms 变形曲线。
func (a *App) mt() ui.ElementTransition {
	if a.ly.resizing != "" || time.Since(a.ly.lastDrag) < 200*time.Millisecond {
		return ui.ElementTransition{}
	}
	return morph
}

// resizeHandle 在面板的一条边内侧放一条拖拽把手。edge:
//
//	"l" 把手在面板左缘,向左拖变宽   "r" 在右缘,向右拖变宽   "t" 在上缘,向上拖变高
//
// cur 是当前尺寸,set 写回新尺寸,lo / hi 是范围,def 是双击恢复的默认值。
func (a *App) resizeHandle(c *ui.Context, key, edge string, cur, def, lo, hi float32, set func(float32)) {
	t := a.tk
	vertical := edge == "t"
	h := ui.Box(c).Key("rz-" + key).Absolute()
	if vertical {
		h.Left(0).Right(0).Top(-3).Height(7).Cursor(ui.CursorResizeNS)
	} else {
		h.Top(0).Bottom(0).Width(7).Cursor(ui.CursorResizeEW)
		if edge == "l" {
			h.Left(-3)
		} else {
			h.Right(-3)
		}
	}
	dx, dy, dragging := h.Dragged()
	if dragging {
		var d float32
		switch edge {
		case "l":
			d = -dx
		case "r":
			d = dx
		default:
			d = -dy
		}
		if d != 0 {
			set(clampF(cur+d, lo, hi))
		}
		a.ly.resizing, a.ly.lastDrag = key, time.Now()
	} else if a.ly.resizing == key { // 松手:写偏好
		a.ly.resizing = ""
		a.saveLayout(key)
	}
	if h.DoubleClicked() {
		set(def)
		a.prefs.Reset("layout." + key)
	}
	lit := dragging || h.Hovered()
	h.Children(func() {
		line := ui.Box(c).Background(t.Ac).Opacity(h.AnimateTo("lit", map[bool]float32{true: .85, false: 0}[lit], uiFade))
		if vertical {
			line.Absolute().Left(0).Right(0).Top(3).Height(2)
		} else {
			line.Absolute().Top(0).Bottom(0).Left(2.5).Width(2)
		}
	})
}

var uiFade = ui.Animation{Duration: kit.DurFast, Ease: kit.EaseOut}

func (a *App) saveLayout(key string) {
	if a.secondary {
		return
	}
	var v float32
	switch key {
	case lySide:
		v = a.ly.side
	case lyWs:
		v = a.ly.ws
	case lyDock:
		v = a.ly.dock
	case lyPanel:
		v = a.ly.panel
	}
	if v > 0 {
		a.prefs.Set("layout."+key, int(v))
	}
}
