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
