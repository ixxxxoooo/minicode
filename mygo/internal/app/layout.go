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
	lySide         = "side"
	lyWs           = "ws"
	lyDock         = "dock"
	lyPanel        = "panel"
	lyTermList     = "termList"
	lySideOutline  = "sideOutline"
	lySideTimeline = "sideTimeline"
)

const (
	sideOutlineDef  float32 = 140
	sideTimelineDef float32 = 160
	sideSecMin      float32 = 64
	sideSecMax      float32 = 280
)

// layoutSizes 是用户拖出来的尺寸(0 = 用默认值)。
type layoutSizes struct {
	side, ws, dock, panel float32
	termList              float32   // 竖向终端列表宽(term.listWidth)
	sideOutline           float32   // 左栏「大纲」展开高度
	sideTimeline          float32   // 左栏「时间线」展开高度
	resizing              string    // 正在拖的把手(拖动期间面板的宽度过渡关闭,否则会跟不上鼠标)
	lastDrag              time.Time // 最近一次拖动(松手后几帧内仍不开过渡)
	panelMax              bool      // 底部面板最大化(占满工作区)
	lastW                 float32   // 上一帧窗口宽度
	winResizeAt, structAt time.Time // 窗口宽度最近一次变化 / 栏开合结构最近一次变化
	sig                   layoutSig
}

func (a *App) loadLayout() {
	get := func(k string) float32 { return float32(a.prefs.Int("layout."+k, 0)) }
	a.ly.side, a.ly.ws, a.ly.dock, a.ly.panel = get(lySide), get(lyWs), get(lyDock), get(lyPanel)
	w := float32(a.prefs.Int(pTermListW, int(termListDefW)))
	a.ly.termList = clampF(w, termListMinW, termListMaxW)
	a.ly.sideOutline = clampF(get(lySideOutline), sideSecMin, sideSecMax)
	a.ly.sideTimeline = clampF(get(lySideTimeline), sideSecMin, sideSecMax)
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

// centerMin 是主区(Agent = 对话栏,IDE = 工作区)的最小宽度:右侧栏随窗口变窄连续缩小让给它,
// 缩到自己的最小宽度仍放不下,才整体收起(浮层 / 图标条),最后只剩中间一栏。
const centerMin = 440

// sideOcc 是导航栏此刻在行内占的宽度(抽屉时为 0)。
// 开合动画途中取「本帧实际的布局宽度」而不是终点:右栏的宽度由它推算(wsRW / dockW),
// 用终点的话,一点开 / 收起导航栏右栏就按终态瞬间变宽或变窄(跳变);
// 用动画值,右栏宽度随导航栏逐帧连续变化,开合结束时两边刚好一致。
func (a *App) sideOcc() float32 {
	if a.sideCurOk {
		return a.sideCur
	}
	if a.useSideDrawer() {
		return 0
	}
	if !a.sideOpen {
		return 0
	}
	return a.sideW()
}

// inlineRoom 是主区保持 centerMin 后,右侧栏最多能拿到的宽度。
func (a *App) inlineRoom() float32 {
	return a.winW - a.sideOcc() - centerMin - a.tk.Scheme.GapPane*2
}

// wsPref / dockPref 是用户偏好(或默认)宽度,浮层用它;行内并排用下面会随窗口收缩的 wsRW / dockW。
func (a *App) wsPref() float32 {
	return clampF(orDef(a.ly.ws, theme.WWs), 360, max(360, a.winW-a.sideW()-360))
}

func (a *App) dockPref() float32 { return clampF(orDef(a.ly.dock, theme.WDock), 300, 560) }

func (a *App) wsRW() float32 {
	w := a.wsPref()
	if a.winW <= 0 {
		return w
	}
	return min(w, max(360, a.inlineRoom()))
}

func (a *App) dockW() float32 {
	w := a.dockPref()
	if a.winW <= 0 {
		return w
	}
	return min(w, max(300, a.inlineRoom()))
}

func (a *App) panelHeight() float32 {
	def := float32(172)
	if a.ide.bottomTab == "problems" || a.ide.bottomTab == "refs" {
		def = 236
	}
	hi := max(120, a.winH-theme.HBar-theme.HTab-120)
	return clampF(orDef(a.ly.panel, def), 96, hi)
}

func clampF(v, lo, hi float32) float32 { return min(max(v, lo), hi) }

// noMotion 是「立即生效」的过渡:时长取负数 = 每次变化直接显示。
// 注意不能写成零值 ElementTransition{}:框架里「什么都没选 = 全部属性都做 200 ms 缓动」,
// 拖动分隔线时面板会落后于指针、与相邻区域错位(曾让编辑器文字透到底部面板标签条上)。
var noMotion = ui.ElementTransition{Duration: -1}

// mt 是面板宽高变化的过渡:拖动分隔线或拖动窗口边缘期间关掉(跟手,不落后于指针、不分档跳变),
// 其余时候用 440 ms 变形曲线;窗口拖到断点、某一栏被收起 / 展开的那一刻仍带动画(layoutStruct 记录)。
func (a *App) mt() ui.ElementTransition {
	if a.instantLayout() {
		return noMotion
	}
	return morph
}

// instantLayout 此刻面板尺寸应直接跟随(不做过渡)。
func (a *App) instantLayout() bool {
	if a.ly.resizing != "" || kit.Since(a.ly.lastDrag) < 200*time.Millisecond {
		return true
	}
	// 窗口宽度正在连续变化,且没有哪一栏刚被收起 / 展开:连续缩放 = 立即跟随
	return kit.Since(a.ly.winResizeAt) < 250*time.Millisecond && kit.Since(a.ly.structAt) > 500*time.Millisecond
}

// trackLayout 每帧调用:记录窗口宽度是否在变,以及栏的开合结构是否变了(变了才值得做收起 / 展开动画)。
func (a *App) trackLayout() {
	if a.winW != a.ly.lastW {
		if a.ly.lastW != 0 {
			a.ly.winResizeAt = kit.Now()
		}
		a.ly.lastW = a.winW
	}
	sig := layoutSig{a.mode, a.sideOpen, a.useSideDrawer(), a.useWsOverlay(), a.useDockRail(), a.rightOn, a.chatMax, a.ly.panelMax}
	if sig != a.ly.sig {
		if a.ly.sig != (layoutSig{}) {
			a.ly.structAt = kit.Now()
		}
		a.ly.sig = sig
	}
}

type layoutSig struct {
	mode                             Mode
	side, drawer, wsOvl, rail, right bool
	chatMax, panelMax                bool
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
		a.ly.resizing, a.ly.lastDrag = key, kit.Now()
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
	case lyTermList:
		v = a.ly.termList
		a.prefs.Set(pTermListW, int(v))
		return
	case lySideOutline:
		v = a.ly.sideOutline
	case lySideTimeline:
		v = a.ly.sideTimeline
	}
	if v > 0 {
		a.prefs.Set("layout."+key, int(v))
	}
}
