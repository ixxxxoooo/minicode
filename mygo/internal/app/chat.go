package app

import (
	"math"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// chatPane 是对话栏:标题行 + 会话流 + 任务坞 + 输入框。
func (a *App) chatPane(c *ui.Context, gap float32) {
	t := a.tk
	p := a.pane(c).Key("chat").Transition(a.mt())
	switch {
	case a.mode == ModeAgent || a.chatMax:
		p.Grow(1).Basis(0)
	case a.rightOn && !a.useDockRail():
		// IDE 形态:对话栏停靠在右侧,固定 352 宽(设计稿 --w-dock)
		p.Width(a.dockW()).Shrink(0).Margin(0, 0, 0, gap)
	case a.useDockRail():
		p.Width(0).Shrink(0).Opacity(0)
		p.Children(func() {})
		return
	default:
		p.Width(0).Shrink(0).Opacity(0)
		p.Children(func() {})
		return
	}
	paneW := p.Bounds().W
	if paneW > 0 {
		a.chatW = paneW
	}
	pin, pinned := a.flowPinFor() // 形态切换期间:三档与文字断行都按终态宽度算,不随过渡逐帧重排
	if pinned {
		a.chatW = pin.paneW
	}
	hint := a.chatDrop(p)
	p.Children(func() {
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
		// 会话流:停在底部时随新内容跟随,用户上滑后不再抢滚动位置
		displayItems := a.flowDisplayState().Items
		if a.flowNoFollow && a.flowSt.MaxY-a.flowSt.Y < 8 { // 手动滚回底部:恢复跟随
			a.flowNoFollow = false
		}
		if !a.flowNoFollow && a.flowSt.MaxY-a.flowSt.Y > 32 {
			a.flowNoFollow = true
			a.flowAwayLen = flowTailMark(displayItems) // 记下离开时的末尾,之后有变化就点亮「有新内容」
		}
		if !a.flowNoFollow && a.flowSt.Y >= a.flowSt.MaxY-8 {
			a.flowSt.Y = math.MaxFloat32
		}
		a.withFlowDisplay(func() {
			ui.Scroll(c).Key("flow").TrackScroll(&a.flowSt).Grow(1).MinHeight(0).Children(func() {
				a.morphWrap(c, "chat", func() {
					col := ui.Column(c).AlignItems(ui.Center)
					if pinned {
						col.Width(pin.flowW).AlignSelf(ui.Start)
					} else {
						a.rememberFlowPin(paneW, col.Bounds().W)
					}
					col.Children(func() {
						a.flowCol(c, func() { a.flow(c) })
					})
				})
			})
			if a.editingUser() {
				ui.ButtonBase(c).Absolute().Left(0).Right(0).Top(0).Bottom(120).ZIndex(1).Cursor(ui.CursorDefault).
					OnClick(a.cancelUserEdit)
			}
			a.flowFadeMask(c)
			a.flowScrollBtn(c)
		})
		a.chatDockColumn(c)
		_ = t
	})
}
