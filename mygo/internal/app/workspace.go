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
	return a.mode == ModeAgent && a.rightOn
}

// wsPane 是工作区槽位:left = IDE 形态的主区,否则 = Agent 形态的右栏。
func (a *App) wsPane(c *ui.Context, gap float32, left bool) {
	active := a.wsActive(left)
	key := "ws-r"
	if left {
		key = "ws-l"
	}
	p := a.pane(c).Key(key).ClipX().Transition(morph)
	switch {
	case active && left:
		p.Grow(1).Basis(0)
	case active:
		p.Width(theme.WWs).Shrink(0).Margin(0, 0, 0, gap)
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
			inner.Width(theme.WWs)
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
	a.etabs(c)
	cur := a.ide.cur()
	switch {
	case cur == nil:
		a.wsEmpty(c)
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
		a.codeView(c, cur)
	}
	if a.ide.bottomOpen {
		a.bottomPanel(c)
	}
}

// wsEmpty 是没有任何标签时的占位。
func (a *App) wsEmpty(c *ui.Context) {
	t := a.tk
	ui.Column(c).Grow(1).Justify(ui.Center).AlignItems(ui.Center).Gap(6).Children(func() {
		ui.Text(c, "没有打开的标签").TextColor(t.Tx2).FontSize(theme.FsTitle).FontWeight(500)
		ui.Text(c, "从文件树、源代码管理或会话流里打开内容").TextColor(t.Tx3)
		if a.mode == ModeIDE {
			ui.Row(c).Gap(6).Margin(6, 0, 0, 0).Children(func() {
				kit.Kbd(c, "⌘", "P")
				ui.Text(c, "快速打开").TextColor(t.Tx4).FontSize(theme.FsCap)
			})
		}
	})
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
		ui.Row(c).Grow(1).MinWidth(0).ClipX().AlignItems(ui.Stretch).Children(func() {
			for _, tb := range a.ide.tabs {
				a.etab(c, tb)
			}
		})
		ui.Row(c).Gap(1).PaddingX(6).AlignItems(ui.Center).Shrink(0).Children(func() {
			kit.Ib(c, "cols", kit.IbOpt{Small: true, Tip: "向右拆分 ⌘\\"})
			kit.Ib(c, "more", kit.IbOpt{Small: true})
		})
	})
}

func (a *App) etab(c *ui.Context, tb *ideTab) {
	t := a.tk
	on := tb.id == a.ide.active
	el := ui.ButtonBase(c).Key("etab-" + tb.id).PaddingX(10).Gap(6).AlignItems(ui.Center).Shrink(0).
		TextColor(t.Tx3).Cursor(ui.CursorDefault).Transition(kit.Fade)
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
	if el.Clicked() {
		a.ide.active = tb.id
		if tb.path != "" {
			a.ide.sel = tb.path
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
