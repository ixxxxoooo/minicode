package app

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// beltChatMinW 是形态切换传送带里对话栏内容的最小排版宽度(dp):栏比它窄时内容仍按它排版、被裁剪显示。
const beltChatMinW = 360

// chatPane 是对话栏:标题行 + 会话流 + 任务坞 + 输入框。
func (a *App) chatPane(c *ui.Context, gap float32) {
	t := a.tk
	// 宽度 / 间距 = 开合因子(stageFactors)× 满开值,不用元素过渡(见 paneOpen 的说明)
	p := a.pane(c).Key("chat").Transition(noMotion)
	shown := true      // 目标状态:对话栏此刻是否应该在场(决定透明度与收起后内容保留多久)
	slide := false     // IDE 对话栏开合动画途中:内容保持定宽、随栏滑动,不逐帧重排
	var slideW float32 // slide 时内容的固定宽度
	switch {
	case a.belt.on:
		// 形态切换的传送带平移:绝对定位 + 逐帧摆位(morph_belt.go),不走弹性布局与宽度过渡
		r := a.belt.chat
		p.Absolute().Left(r.x).Top(0).Bottom(0).Width(max(0, r.w)).Shrink(0).Transition(noMotion)
		shown = r.w > .5
		// 传送带里栏宽从几像素一路涨到几百(或反过来):内容若按逐帧宽度排版,栏宽只有几像素到几十像素时
		// 文字几乎一个字一行,布局暴涨到 40–100 ms(日志里切形态掉帧都落在这些帧)。
		// 所以内容宽度不低于 beltChatMinW,窄于它时只是被外层裁掉一截,宽度够大后与按栏宽排版完全一样。
		slide, slideW = true, max(r.w, beltChatMinW)
	case a.mode == ModeAgent:
		// Agent:占满型不带自己的动画,宽度完全跟着相邻栏(左栏 / 右栏)每一帧的宽度走(设计稿里它是 flex: 1)
		p.Grow(1).Basis(0)
	default:
		// IDE:对话栏停靠在右侧,宽度 = 满开宽度 × 停靠因子,向「占满舞台」插值最大化因子。
		// 收起不再瞬间归零:宽度 / 间距 / 透明度一起走完 440 ms(设计稿原本是 width:0 瞬间消失,设计稿已同步改成过渡),
		// 工作区永远拿剩余,所以左边缘被推走的每一帧都有内容盖着,不会露出空白。
		shown = a.chatMax || (a.rightOn && !a.useDockRail())
		fd, fm := a.fDock, a.fMax
		if fm >= .999 {
			p.Grow(1).Basis(0) // 最大化稳态:占满(工作区宽度为 0)
		} else {
			w := a.dockW() * fd * (1 - fm)
			if fm > 0 { // 最大化动画途中:向整个舞台宽度插值(舞台宽度取上一帧量到的)
				w += a.stageW * fm
			}
			p.Width(w).Shrink(0).Margin(0, 0, 0, gap*fd*(1-fm))
			slide = fm <= 0 && fd < .999
			slideW = a.dockW()
		}
	}
	// 透明度 260ms ease-io(设计稿 .pane);形态切换那 440ms 里改走 chat-dock / chat-center:
	// 整块对话栏自 ∓34% 处滑入,透明度 .4 → 1(chatSlide)。
	opDur := kit.DurSlow
	if a.belt.on {
		opDur = time.Millisecond // 平移中不淡入淡出:透明度直接到位,结束后也不会再补一次淡入
	}
	p.Opacity(p.AnimateWith("op", map[bool]float32{true: 1, false: 0}[shown], opDur, kit.EaseIO))
	if shown && !a.belt.on {
		a.chatSlide(p, a.chatW)
	}
	a.probe(pzChat, p)
	// 收起期间继续画内容,被宽度动画一点点裁掉(窄窗改成图标条时对话栏在浮层里画,不能同时画两份)
	if !a.keepWhile("chat", shown, kit.DurMorph) || (!shown && a.useDockRail() && a.dockFloat) {
		p.Children(func() {})
		return
	}
	if w := p.Bounds().W; w > 0 && shown {
		a.chatW = w
	}
	hint := a.chatDrop(p)
	build := func() {
		drop := ui.Box(c).Absolute().Left(0).Right(0).Top(0).Bottom(0).PassThrough()
		if h := a.chatDrop(drop); h != "" {
			hint = h
		}
		a.ux.dropOn = hint != "" // 有文件拖过来:输入框变成虚线「拖入框」(composer.go),不再盖整栏提示
		if a.mode == ModeIDE && !a.chatMax {
			a.resizeHandle(c, lyDock, "l", a.dockW(), theme.WDock, 300, max(300, min(dockMaxW, a.inlineRoom())), func(v float32) { a.ly.dock = v })
		}
		switch {
		case a.showChatWelcome():
			a.welcome(c)
			return
		case a.sessLoad.err != "" && len(a.st.Items) == 0:
			a.chatSessionLoadFailed(c)
			return
		case a.sessLoad.active && len(a.st.Items) == 0 && (a.sessLoad.hold == nil || len(a.sessLoad.hold.Items) == 0):
			a.chatSessionLoading(c)
			return
		case len(a.st.Items) == 0 && a.curSess != "" && !a.sessLoad.active:
			a.chatEmptyOpened(c)
			return
		}
		a.syncSteers()
		a.chatHead(c, a.chatHeadTitle())
		// 会话流:虚拟列表自己贴底(FollowEnd);用户上滑离开底部后不再抢滚动位置。
		displayItems := a.flowDisplayState().Items
		if a.flowList.AtEnd() { // 在底部(或手动滚回底部):恢复跟随
			a.flowNoFollow = false
		} else if !a.flowNoFollow && len(displayItems) > 0 {
			a.flowNoFollow = true
			a.flowAwayLen = flowTailMark(displayItems) // 记下离开时的末尾,之后有变化就点亮「有新内容」
		}
		a.withFlowDisplay(func() {
			a.morphWrap(c, "chat", func() {
				a.flowListView(c)
				a.userPinnedOverlay(c)
			})
			a.flowFadeMask(c)
			a.flowScrollBtn(c)
		})
		a.chatDockColumn(c)
		_ = t
	}
	p.Children(func() {
		if slide {
			// 内容固定在停靠宽度、靠左,栏的左缘一路移动,整块就像贴着右缘滑进 / 滑出;
			// 不逐帧按变化的宽度重排会话(长会话重排很重,是掉帧、宽度「先小后突然变大」的来源)
			ui.Column(c).Key("chat-fix").Width(slideW).Grow(1).MinHeight(0).Shrink(0).Children(build)
			return
		}
		build()
	})
}
