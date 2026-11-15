package app

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 区域模型(设计稿 v3.1):导航栏、对话栏、工作区、状态栏。没有「检查器」。
//
//	Agent 形态:[导航栏][对话栏(居中,占满)][工作区 560(默认收起,开标签时展开)]
//	IDE   形态:[导航栏][工作区(占满)][对话栏 352]
//
// 工作区 = 编辑器标签组 + 底部面板,全窗口只有一份状态(a.ide),所以两个形态里是同一个编辑器、同一个终端;
// 它只是在两处「槽位」里轮流显示(同一时刻只有一个槽位有内容)。

// stage 是舞台:按形态排布工作区与对话栏。槽位始终在树里(带 Key),宽度变化走 440 ms 变形过渡。
func (a *App) stage(c *ui.Context) {
	gap := a.tk.Scheme.GapPane
	ui.Row(c).Grow(1).MinWidth(0).AlignItems(ui.Stretch).Children(func() {
		if a.screen != ScreenWork && a.screen != ScreenWelcome {
			if !a.page(c) {
				a.pagePlaceholder(c)
			}
			return
		}
		a.wsPane(c, gap, true)
		a.chatPane(c, gap)
		a.wsPane(c, gap, false)
	})
}

// wsActive 该槽位此刻是否承载工作区。
func (a *App) wsActive(left bool) bool {
	if left {
		return a.mode == ModeIDE && !a.chatMax
	}
	if a.useWsOverlay() {
		return false
	}
	return a.mode == ModeAgent && a.rightOn
}

// wsPane 是工作区槽位:left = IDE 形态的主区,否则 = Agent 形态的右栏。
func (a *App) wsPane(c *ui.Context, gap float32, left bool) {
	active := a.wsActive(left)
	key := "ws-r"
	if left {
		key = "ws-l"
	}
	p := a.pane(c).Key(key).ClipX().Transition(a.mt())
	switch {
	case active && left:
		p.Grow(1).Basis(0)
	case active:
		p.Width(a.wsRW()).Shrink(0).Margin(0, 0, 0, gap)
	default:
		p.Width(0).Shrink(0).Opacity(0)
	}
	if !active {
		p.Children(func() {})
		return
	}
	if w := p.Bounds().W; w > 0 {
		a.wsW = w
	}
	p.Children(func() {
		inner := ui.Column(c).Grow(1).MinHeight(0)
		if !left {
			inner.Width(a.wsRW())
			a.resizeHandle(c, lyWs, "l", a.wsRW(), theme.WWs, 360, max(360, a.winW-a.sideW()-360), func(v float32) { a.ly.ws = v })
		}
		inner.Children(func() { a.workspace(c) })
		a.flashOverlay(c)
	})
}

// flashOverlay 是「开标签唤起工作区」时边框闪一下的描边层(约 0.9 秒:快速亮起、缓慢淡出)。
func (a *App) flashOverlay(c *ui.Context) {
	el := time.Since(a.flashAt)
	on := !a.flashAt.IsZero() && el < 280*time.Millisecond
	live := !a.flashAt.IsZero() && el < 1100*time.Millisecond
	if live {
		c.Invalidate() // 动画期间持续出帧
	}
	box := ui.Box(c).Key("flash").Absolute().Left(0).Right(0).Top(0).Bottom(0).PassThrough().
		Border(2, a.tk.Ac).Radius(a.tk.RLg())
	target := float32(0)
	if on {
		target = 1
	}
	dur := kit.DurSlow
	if !on {
		dur = 600 * time.Millisecond
	}
	box.Opacity(box.AnimateWith("fl", target, dur, kit.EaseOut) * 0.9)
}

// workspace 是工作区内容:标签条 + 当前标签的内容 + 底部面板。
func (a *App) workspace(c *ui.Context) {
	t := a.tk
	if a.ly.panelMax && a.ide.bottomOpen { // 底部面板最大化:标签条与编辑器让位
		a.bottomPanel(c)
		return
	}
	if a.ide.splitDir == 0 {
		a.etabs(c)
	} else {
		a.etabsSplitBar(c)
	}
	cur := a.ide.cur()
	switch {
	case cur == nil && a.ide.splitDir == 0:
		a.wsEmpty(c)
	case a.ide.splitDir != 0 && !cur.isVirt():
		if cur != nil {
			a.crumbs(c, cur)
		}
		a.wsSplit(c, cur)
	case cur.isVirt():
		a.virtView(c, cur)
	case cur.err != "":
		a.crumbs(c, cur)
		ui.Column(c).Grow(1).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			ui.Text(c, cur.err).TextColor(t.Tx3)
		})
	case cur.diff:
		a.diffTab(c, cur)
	default:
		a.crumbs(c, cur)
		if cur.buf != nil && isMarkdownFile(cur.path) && !a.mdSource(cur) {
			if cur.reveal != nil { // 要跳到某一行 / 某段:只有源码能定位
				cur.mdMode = mdSource
				a.codeView(c, cur)
			} else {
				a.mdView(c, cur)
			}
		} else {
			a.codeView(c, cur)
		}
	}
	if a.ide.bottomOpen {
		a.bottomPanel(c)
	}
}

// ───────── 标签条 ─────────

func (a *App) etabs(c *ui.Context) {
	t := a.tk
	bar := ui.Row(c).Height(theme.HTab).Shrink(0).AlignItems(ui.Stretch)
	switch t.Scheme.ID {
	case "a":
		bar.Background(t.Chrome).BorderWidth(0, 0, 1, 0).BorderColor(t.Bd2)
	case "b":
		bar.Padding(4, 4, 0, 4).Gap(2)
	default:
		bar.BorderWidth(0, 0, 1, 0).BorderColor(t.Bd1)
	}
	bar.Children(func() {
		d := a.ide
		strip := ui.ScrollHorizontal(c).Key("etabs-scroll").TrackScroll(&d.tabSt).Grow(1).MinWidth(0).AlignItems(ui.Stretch)
		if w := strip.Bounds().W; w > 0 && d.tabSeen != d.active { // 当前标签变了:滚到可见(用上一帧量到的位置)
			var left, wid float32
			for _, tb := range d.tabs {
				if tb.id == d.active {
					wid = d.tabW[tb.id]
					break
				}
				left += d.tabW[tb.id]
			}
			switch {
			case wid == 0:
			case left < d.tabSt.X:
				d.tabSt.X = left
			case left+wid > d.tabSt.X+w:
				d.tabSt.X = left + wid - w
			}
			d.tabSeen = d.active
		}
		strip.Children(func() {
			ui.Row(c).AlignItems(ui.Stretch).Children(func() {
				for _, tb := range d.tabs {
					a.etabPane(c, tb, 0)
				}
			})
		})
		ui.Row(c).Gap(1).PaddingX(6).AlignItems(ui.Center).Shrink(0).Children(func() {
			kit.Ib(c, "cols", kit.IbOpt{Small: true, Tip: "向右拆分 ⌘\\"}).OnClick(func() { a.splitEditor(1) })
			kit.Ib(c, "rows", kit.IbOpt{Small: true, Tip: "向下拆分"}).OnClick(func() { a.splitEditor(2) })
			kit.Ib(c, "more", kit.IbOpt{Small: true, Tip: "更多操作"}).Menu(a.tabStripMenu)
		})
	})
}

func (a *App) etab(c *ui.Context, tb *ideTab) { a.etabPane(c, tb, 0) }

func (a *App) etabPane(c *ui.Context, tb *ideTab, pane int8) {
	t := a.tk
	d := a.ide
	on := tb.id == *d.paneActive(pane)
	el := ui.ButtonBase(c).Key("etab-" + tb.id).PaddingX(10).Gap(6).AlignItems(ui.Center).Shrink(0).
		TextColor(t.Tx3).Cursor(ui.CursorDefault).Transition(kit.Fade)
	if isFileTab(tb) {
		dragRef(el, tb.path, false) // 拖到对话栏 = 引用这个文件
	}
	if a.ide.splitDir != 0 {
		a.etabDrag(el, tb, pane)
	}
	el.ContextMenu(func(m *ui.Menu) { a.tabMenu(m, tb) })
	hov := el.Hovered()
	if on || hov {
		el.TextColor(t.Tx1)
	}
	switch t.Scheme.ID {
	case "a":
		el.BorderWidth(0, 1, 0, 0).BorderColor(t.Bd1)
		if on {
			el.Background(t.BG)
		}
	case "b":
		el.Radius(t.RSm()).Height(24)
		if on {
			el.Background(t.Fill2)
		}
	}
	if w := el.Bounds().W; w > 0 {
		if a.ide.tabW == nil {
			a.ide.tabW = map[string]float32{}
		}
		a.ide.tabW[tb.id] = w
	}
	if el.Clicked() {
		*d.paneActive(pane) = tb.id
		d.paneSel = pane
		if tb.path != "" {
			d.sel = tb.path
		}
	}
	if el.DoubleClicked() {
		tb.preview = false
	}
	el.Children(func() {
		if on && t.Scheme.ID == "a" {
			ui.Box(c).Absolute().Left(0).Right(0).Top(0).Height(2).Background(t.Ac)
		}
		if on && t.Scheme.ID == "c" {
			ui.Box(c).Absolute().Left(10).Right(10).Bottom(0).Height(2).Radius(2, 2, 0, 0).Background(t.Ac)
		}
		ic := kit.Ic(c, a.tabIcon(tb), kit.S12)
		if tb.virt == vReview || tb.virt == vPlan {
			ic.TextColor(t.Ac)
		} else if tb.virt == "" && !tb.diff {
			ic.TextColor(a.fileTint(tb.path, t.Tx3))
		}
		nm := ui.Text(c, a.tabTitle(tb)).SingleLine()
		if tb.preview {
			nm.Italic()
		}
		if bd := a.tabBadge(tb); bd != "" {
			kit.Cnt(c, bd, true)
		}
		// 右侧:未保存圆点,悬停 / 当前标签显示关闭
		slot := ui.Row(c).Size(14, 14).Justify(ui.Center).AlignItems(ui.Center).Radius(t.RXs()).Shrink(0)
		if tb.dirty() && !hov {
			slot.Children(func() { ui.Box(c).Size(6, 6).Radius(3).Background(t.Tx3) })
			return
		}
		x := slot.Key("x-" + tb.id)
		if x.Hovered() {
			x.Background(t.Fill2)
		}
		if x.Clicked() {
			a.closeWsTab(tb.id)
		}
		x.Children(func() {
			if on || hov {
				kit.Ic(c, "x", kit.S12).TextColor(t.Tx3)
			}
		})
	})
}

// closeWsTab 关闭标签;Agent 形态下最后一个标签关掉后工作区收起(没有内容就不占位)。
func (a *App) closeWsTab(id string) {
	a.ide.closeTab(id)
	if a.mode == ModeAgent && len(a.ide.tabs) == 0 {
		a.rightOn = false
	}
}
