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
		s.at = time.Now()
	}
	if s.at.IsZero() || d <= 0 {
		return false
	}
	if time.Since(s.at) < d {
		a.frameSoon()
		return true
	}
	return false
}

// syncNavSwap 左栏内容的「形态 + 分段」变了就记下入场起点(首帧不播)。
func (a *App) syncNavSwap() {
	key := a.NavID() + "|" + strconv.Itoa(int(a.mode))
	if a.navSwapKey != "" && a.navSwapKey != key {
		a.navSwapAt = time.Now()
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

// paneW 是侧栏 / 工作区 / 对话栏停靠的「布局宽度」逐帧动画值(设计稿 .side / .pane 的 width 过渡:
// 左栏 260ms ease-io,右栏 440ms 变形曲线)。
// 宽度直接作为布局宽度:占满型的栏(对话栏 / IDE 工作区)自然跟着相邻栏每一帧的宽度走,
// 和 CSS 的 flex 布局语义一致;不用 MyGo 的 Transition——它的位置按父级局部坐标插值,
// 左栏一收舞台原点瞬间跳走、右栏的位置再慢慢追,合成出来先在左边、再向右滑,看着就不对。
// 参数 slow:true = 260ms ease-io(左栏),false = 440ms 变形曲线(右栏 / 对话栏停靠)。
func (a *App) paneW(p ui.Element, target float32, slow bool) float32 {
	dur, ease := kit.DurMorph, kit.EaseMorph
	if slow {
		dur, ease = kit.DurSlow, kit.EaseIO
	}
	snap := a.paneSnap()
	if dur <= 0 || snap {
		dur = time.Millisecond
	}
	v := p.AnimateWith("w", target, dur, ease)
	if snap { // 拖分隔线 / 缩放窗口:直接取目标值(动画值会落后一帧,拖拽把手又按目标值算位置,两边对不上就抖)
		return target
	}
	return v
}

// paneGap 是面板左侧外边距(面板间距)的动画值:开合时随宽度同曲线渐变(设计稿 .pane 的 margin 过渡),
// 否则收起 / 展开的第一帧会有一个间距宽度的跳动。
func (a *App) paneGap(p ui.Element, on bool, gap float32) float32 {
	target := float32(0)
	if on {
		target = gap
	}
	snap := a.paneSnap()
	dur := kit.DurMorph
	if dur <= 0 || snap {
		dur = time.Millisecond
	}
	v := p.AnimateWith("gap", target, dur, kit.EaseMorph)
	if snap {
		return target
	}
	return v
}

// morphProgress 形态切换自身的进度 0..1(EaseMorph);没有切换或已结束返回 1,done = true。
func (a *App) morphProgress() (p float32, done bool) {
	if !kit.MotionOn() || morph.Duration <= 0 || a.morphAt.IsZero() {
		return 1, true
	}
	el := time.Since(a.morphAt)
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
