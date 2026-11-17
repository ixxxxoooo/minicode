package app

import (
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

// 布局类动效(设计稿 css/ws.css、shell.css、motion.css):
//
//	左栏开合      width --dur-slow --ease-io;opacity --dur-fast(CSS 默认 ease)
//	右栏开合      width / margin --dur-morph --ease-morph;opacity --dur-slow --ease-io
//	底部面板开合  height --dur-slow --ease-io;opacity --dur-fast
//	Agent ⇄ IDE  对话栏 chat-dock / chat-center:自 ∓34% 滑入 + opacity .4 → 1,--dur-morph --ease-morph
//	左栏换内容    a-swap:自左 8px 滑入 + 淡入,--dur-base --ease-out

// slotLife 记录一个槽位(右栏 / 对话栏 / 底部面板)最近一次由「活跃」变成「不活跃」的时刻。
type slotLife struct {
	seen   bool // 已经记录过一帧
	active bool
	at     time.Time // 变成不活跃的时刻
}

// keepWhile 报告槽位是否仍要画内容:活跃时恒为 true;刚收起后的 d 时长内也为 true,
// 这样宽度 / 高度的收起动画里内容被一点点裁掉(设计稿里内容一直在 DOM 里),而不是第一帧就消失。
// 每帧每个槽位只能调用一次。
func (a *App) keepWhile(key string, active bool, d time.Duration) bool {
	if a.slotLife == nil {
		a.slotLife = map[string]*slotLife{}
	}
	s := a.slotLife[key]
	if s == nil {
		s = &slotLife{}
		a.slotLife[key] = s
	}
	defer func() { s.seen, s.active = true, active }()
	if active {
		return true
	}
	if s.seen && s.active { // 这一帧刚收起
		s.at = kit.Now()
	}
	if s.at.IsZero() || d <= 0 {
		return false
	}
	if kit.Since(s.at) < d {
		a.frameSoon()
		return true
	}
	return false
}

// syncNavSwap 左栏内容的「形态 + 分段」变了就记下入场起点(首帧不播)。
func (a *App) syncNavSwap() {
	key := a.NavID() + "|" + strconv.Itoa(int(a.mode))
	if a.navSwapKey != "" && a.navSwapKey != key {
		a.navSwapAt = kit.Now()
	}
	a.navSwapKey = key
}

// chatDockOn IDE 形态下对话栏是否以右侧停靠栏的形式占位(右栏开着、不是窄窗图标条、没被最大化)。
func (a *App) chatDockOn() bool {
	return a.mode == ModeIDE && !a.chatMax && a.rightOn && !a.useDockRail()
}

// absF 是 float32 的绝对值。
func absF(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

// paneSnap 此刻面板尺寸应直接到位、不做开合动画:形态切换进行中(平移由传送带摆位)、拖分隔线 / 连续缩放窗口时。
func (a *App) paneSnap() bool {
	if _, done := a.morphProgress(); !done || a.belt.on {
		return true
	}
	return a.instantLayout()
}

// paneOpen 是一块面板的「开合因子」0..1 的逐帧动画值(设计稿 .side / .pane 的 width 过渡:
// 左栏 260ms ease-io,右栏 440ms 变形曲线)。
//
// 为什么动画「因子」而不是动画「宽度」:面板的实际宽度 = 因子 × 满开宽度,而满开宽度(wsRW / dockW / sideW)
// 会随窗口大小、相邻栏当前宽度连续变化。若直接动画像素宽度,目标每帧都在变,动画值永远追在后面
// (表现为:收起导航栏时对话栏先被撑宽、过一会儿才缩回;缩放窗口时右栏落后一帧抖动)。
// 因子只表达「开 / 合」这件结构性的事,满开宽度每帧按最新的窗口宽度与导航栏动画宽度重算,两者相乘互不干扰,
// 开合与缩放可以同时发生。不用 MyGo 的 Transition:它的位置按父级局部坐标插值,舞台原点一跳,面板就先在左边再滑向右边。
// 参数 slow:true = 260ms ease-io(左栏),false = 440ms 变形曲线(右栏 / 对话栏停靠 / 最大化)。
// 拖分隔线 / 连续缩放窗口 / 形态切换传送带:直接取终值(见 paneSnap)。
func (a *App) paneOpen(p ui.Element, key string, on bool, slow bool) float32 {
	return a.paneOpenSnap(p, key, on, slow, a.paneSnap())
}

// paneOpenFree 同 paneOpen,但形态切换进行中也照常做开合动画(只有拖分隔线 / 连续缩放窗口时才直接到位)。
// 给传送带之外的面板用(导航栏):两个形态各自记着导航栏的开合,IDE(左栏开)切到 Agent(左栏收)时,
// 若导航栏在切换那一帧直接到位,整个舞台会在一帧里向左 / 向右跳 228 px,传送带的几何跟着错位,右边露出空白。
func (a *App) paneOpenFree(p ui.Element, key string, on bool, slow bool) float32 {
	return a.paneOpenSnap(p, key, on, slow, a.instantLayout())
}

func (a *App) paneOpenSnap(p ui.Element, key string, on bool, slow bool, snap bool) float32 {
	target := float32(0)
	if on {
		target = 1
	}
	dur, ease := kit.DurMorph, kit.EaseMorph
	if slow {
		dur, ease = kit.DurSlow, kit.EaseIO
	}
	if dur <= 0 || snap {
		dur = time.Millisecond
	}
	v := p.AnimateWith(key, target, dur, ease)
	if snap {
		return target
	}
	return clampF(v, 0, 1)
}

// stageFactors 每帧在画舞台之前算好工作区 / 对话栏的开合因子(两块面板互相依赖,必须先于各自构建)。
// row 是舞台那一行(因子的动画状态挂在它上面,身份稳定)。
//
//	Agent  fWs   右侧工作区:右栏开着且不是浮层
//	IDE    fDock 对话栏停靠:右栏开着且不是图标条,或被最大化
//	IDE    fMax  对话栏最大化:占满舞台,工作区让位
func (a *App) stageFactors(row ui.Element) {
	agent := a.mode == ModeAgent
	a.fWs = a.paneOpen(row, "f-ws", agent && a.wsOccupies(), false)
	a.fDock = a.paneOpen(row, "f-dock", !agent && (a.chatMax || (a.rightOn && !a.useDockRail())), false)
	a.fMax = a.paneOpen(row, "f-max", !agent && a.chatMax, false)
}

// morphProgress 形态切换自身的进度 0..1(EaseMorph);没有切换或已结束返回 1,done = true。
func (a *App) morphProgress() (p float32, done bool) {
	if !kit.MotionOn() || morph.Duration <= 0 || a.morphAt.IsZero() {
		return 1, true
	}
	el := kit.Since(a.morphAt)
	if el >= morph.Duration {
		return 1, true
	}
	if a.win != nil {
		a.frameSoon()
	}
	return kit.EaseMorph(float32(el) / float32(morph.Duration)), false
}

// chatSlide 对话栏在形态切换时的滑动(chat-dock / chat-center):
// 进 IDE 自左侧 34% 处滑入,回 Agent 自右侧 34% 处滑入;opacity .4 → 1。
func (a *App) chatSlide(p ui.Element, w float32) {
	e, done := a.morphProgress()
	if done {
		return
	}
	dir := float32(-1)
	if a.mode == ModeAgent {
		dir = 1
	}
	p.Translate(dir*.34*w*(1-e), 0)
	p.Opacity(.4 + .6*e)
}
