package app

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// chatPane 是对话栏:标题行 + 会话流 + 任务坞 + 输入框。
func (a *App) chatPane(c *ui.Context, gap float32) {
	t := a.tk
	// 宽度 / 间距用显式的逐帧动画值(paneW / paneGap),不用元素过渡
	p := a.pane(c).Key("chat").Transition(noMotion)
	shown := true
	slide := false // IDE 对话栏开合动画途中:内容保持定宽、随栏滑动,不逐帧重排
	switch {
	case a.belt.on:
		// 形态切换的传送带平移:绝对定位 + 逐帧摆位(morph_belt.go),不走弹性布局与宽度过渡
		r := a.belt.chat
		p.Absolute().Left(r.x).Top(0).Bottom(0).Width(max(0, r.w)).Shrink(0).Transition(noMotion)
		shown = r.w > .5
		// 只为把外边距的动画状态推到终值,平移结束后不会再补一次滑动
		_ = a.paneGap(p, a.mode == ModeIDE && !a.chatMax && a.belt.chatOn, gap)
	case a.mode == ModeAgent || a.chatMax:
		// 占满型不带自己的动画:宽度完全跟着相邻栏(左栏 / 右栏)每一帧的宽度走(设计稿里它是 flex: 1,没有 width 过渡)
		p.Grow(1).Basis(0)
		_ = a.paneGap(p, false, gap)
	case a.rightOn && !a.useDockRail():
		// IDE 形态:对话栏停靠在右侧(设计稿 .chat { flex: 1 1 0 })。
		// 开合动画途中它占满工作区(wsPane 里显式动画)让出来的那一段,展开时随工作区收窄逐帧变宽;
		// 走完 / 拖拽 / 缩放窗口时 wsPane 占满,这里定宽 --w-dock。两种分配在交接那一帧结果相同。
		p.Margin(0, 0, 0, a.paneGap(p, true, gap))
		if a.ideRest {
			p.Width(a.dockW()).Shrink(0)
		} else {
			p.Grow(1).Basis(0).ClipX()
			slide = true
		}
	default:
		// 收起(右栏关了 / 窄窗收成图标条):设计稿 .chat { flex: 0 0 0; width: 0 } 宽度瞬间归零(不过渡),
		// 只有工作区在 440ms 里向右扩展填满;透明度 260ms ease-io、间距 440ms 照常渐变
		p.Width(0).Shrink(0).Margin(0, 0, 0, a.paneGap(p, false, gap))
		shown = false
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
	// 收起期间继续画内容,被宽度动画一点点裁掉(窄窗改成图标条时对话栏在浮层里画,不能同时画两份)
	if !a.keepWhile("chat", shown, kit.DurMorph) || (!shown && a.useDockRail()) {
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
		if hint != "" {
			defer a.dropOverlay(c, hint) // 盖在最上层(后画),各条返回路径都会执行
		}
		if a.mode == ModeIDE && !a.chatMax {
			a.resizeHandle(c, lyDock, "l", a.dockW(), theme.WDock, 300, 560, func(v float32) { a.ly.dock = v })
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
			a.morphWrap(c, "chat", func() { a.flowListView(c) })
			if a.editingUser() {
				ui.ButtonBase(c).Absolute().Left(0).Right(0).Top(0).Bottom(120).ZIndex(1).Cursor(ui.CursorDefault).
					OnClick(a.cancelUserEdit)
			}
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
			ui.Column(c).Key("chat-fix").Width(a.dockW()).Grow(1).MinHeight(0).Shrink(0).Children(build)
			return
		}
		build()
	})
}
