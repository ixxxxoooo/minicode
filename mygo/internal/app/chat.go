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
	if w := p.Bounds().W; w > 0 {
		a.chatW = w
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
		if len(a.st.Items) == 0 {
			a.welcome(c)
			return
		}
		a.syncSteers()
		a.chatHead(c, a.st.Title)
		// 会话流:停在底部时随新内容跟随,用户上滑后不再抢滚动位置
		if a.flowNoFollow && a.flowSt.MaxY-a.flowSt.Y < 8 { // 手动滚回底部:恢复跟随
			a.flowNoFollow = false
		}
		if !a.flowNoFollow && a.flowSt.MaxY-a.flowSt.Y > 32 {
			a.flowNoFollow = true
			a.flowAwayLen = flowTailMark(a.st.Items) // 记下离开时的末尾,之后有变化就点亮「有新内容」
		}
		if !a.flowNoFollow && a.flowSt.Y >= a.flowSt.MaxY-8 {
			a.flowSt.Y = math.MaxFloat32
		}
		ui.Scroll(c).Key("flow").TrackScroll(&a.flowSt).Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).AlignItems(ui.Center).Children(func() {
				a.flowCol(c, func() { a.flow(c) })
			})
		})
		if a.editingUser() {
			ui.ButtonBase(c).Absolute().Left(0).Right(0).Top(0).Bottom(120).ZIndex(1).Cursor(ui.CursorDefault).
				OnClick(a.cancelUserEdit)
		}
		a.flowFadeMask(c)
		a.flowScrollBtn(c)
		ui.Column(c).Padding(4, 0, 10, 0).Shrink(0).AlignItems(ui.Center).Children(func() {
			a.flowCol(c, func() {
				a.syncDockComposer()
				a.taskDock(c)
				if !a.dockCard(c) {
					a.queueCard(c)
					a.composer(c)
				}
			})
		})
		_ = t
	})
}
