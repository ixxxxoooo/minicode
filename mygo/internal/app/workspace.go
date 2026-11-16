package app

import (
	"path"
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
			a.guard(c, "页面", "page:"+string(a.screen), guardCard, func() {
				if !a.page(c) {
					a.pagePlaceholder(c)
				}
			})
			return
		}
		a.wsPane(c, gap, true)
		a.guard(c, "对话", "chat", guardCard, func() { a.chatPane(c, gap) })
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
		p.Width(0).Shrink(0)
	}
	// 透明度 260ms ease-io(设计稿 .pane);形态切换里「展开中」的那一槽直接显示,内容由 morphWrap 分轨淡入。
	op := p.AnimateWith("op", map[bool]float32{true: 1, false: 0}[active], kit.DurSlow, kit.EaseIO)
	if active && a.morphing() {
		op = 1
	}
	p.Opacity(op)
	// 收起期间继续画内容(被宽度动画一点点裁掉);另一槽正承载工作区时不能画第二份,否则同一份编辑器状态会出现两处。
	keep := a.keepWhile("ws:"+key, active, kit.DurMorph) && !a.wsActive(!left)
	if !active && !keep {
		p.Children(func() {})
		return
	}
	if w := p.Bounds().W; w > 0 {
		a.wsW = w
	}
	p.Children(func() {
		inner := ui.Column(c).Grow(1).MinHeight(0)
		a.zoneMarkEditor(inner)
		if !left {
			inner.Width(a.wsRW())
			if active {
				a.resizeHandle(c, lyWs, "l", a.wsRW(), theme.WWs, 360, max(360, a.winW-a.sideW()-360), func(v float32) { a.ly.ws = v })
			}
		}
		inner.Children(func() { a.morphWrap(c, "ws", func() { a.workspace(c) }) })
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
	key := "tab:empty" // 一个标签出错只隔离这个标签的内容,标签条还在,可以切走或关闭
	if cur != nil {
		key = "tab:" + cur.id
	}
	a.guard(c, "编辑区", key, guardCard, func() { a.workspaceBody(c, cur) })
	// 底部面板始终在树里:收起时高度 260ms ease-io 归零(内容保留到动画结束),展开时从 0 长出来
	a.guard(c, "底部面板", "bottom", guardCard, func() { a.bottomPanel(c) })
}

// workspaceBody 是工作区里标签条以下、底部面板以上的内容。
func (a *App) workspaceBody(c *ui.Context, cur *ideTab) {
	switch {
	case cur == nil && a.ide.splitDir == 0:
		a.wsEmpty(c)
	case a.ide.splitDir != 0 && !cur.isVirt(): // 拆分后焦点组可能没有标签(cur 为 nil),两个组各自显示空态
		a.wsSplit(c, cur) // 面包屑在各组里自己画
	case cur.isVirt():
		a.virtView(c, cur)
	case cur.viewer != "":
		a.crumbs(c, cur)
		a.viewerTab(c, cur)
	case cur.err != "":
		a.crumbs(c, cur)
		a.errView(c, cur)
	case cur.diff:
		a.diffTab(c, cur)
	default:
		a.crumbs(c, cur)
		if cur.buf != nil && isMarkdownFile(cur.path) && !a.mdSource(cur) {
			a.mdView(c, cur)
		} else {
			a.codeView(c, cur)
		}
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
		// 标签条永远没有滚动条(设计稿 scroll-spec「标签条」):放不下就横向滚动(滚轮 / Shift + 滚轮 / 触控板),
		// 被遮住的一侧边缘渐隐,溢出时工具区多一个「全部标签」菜单;所以这里不接 kit 的滚动条策略。
		holder := ui.Row(c).Key("etabs-holder").Grow(1).MinWidth(0).AlignItems(ui.Stretch)
		var over, fadeL, fadeR bool
		holder.Children(func() {
			strip := ui.ScrollHorizontal(c).Key("etabs-scroll").TrackScroll(&d.tabSt).Grow(1).MinWidth(0).AlignItems(ui.Stretch).Scrollbars(false)
			w := strip.Bounds().W
			if w > 0 && d.tabSeen != d.active { // 当前标签变了:滚到可见(用上一帧量到的位置)
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
			var total float32
			for _, tb := range d.tabs {
				total += d.tabW[tb.id]
			}
			over = w > 0 && total > w+1
			fadeL = over && d.tabSt.X > 1
			fadeR = over && d.tabSt.X < total-w-1
			strip.Children(func() {
				ui.Row(c).AlignItems(ui.Stretch).Children(func() {
					for _, tb := range d.tabs {
						a.etabPane(c, tb, 0)
					}
				})
			})
			a.etabFades(c, fadeL, fadeR)
		})
		ui.Row(c).Gap(1).PaddingX(6).AlignItems(ui.Center).Shrink(0).Children(func() {
			if over {
				a.etabListBtn(c)
			}
			kit.Ib(c, "cols", kit.IbOpt{Small: true, Tip: "向右拆分 ⌘\\"}).OnClick(func() { a.splitEditor(1) })
			kit.Ib(c, "rows", kit.IbOpt{Small: true, Tip: "向下拆分"}).OnClick(func() { a.splitEditor(2) })
			kit.Ib(c, "more", kit.IbOpt{Small: true, Tip: "更多操作"}).Menu(a.tabStripMenu)
		})
	})
}

// etabFades 在标签条被遮住的一侧画 22px 渐隐(不吃指针),提示那边还有标签。
func (a *App) etabFades(c *ui.Context, left, right bool) {
	t := a.tk
	bg := t.BG
	if t.Scheme.ID == "a" {
		bg = t.Chrome
	}
	clear := bg.Alpha(0)
	if left {
		ui.Box(c).Key("etab-fade-l").Absolute().Left(0).Top(0).Bottom(0).Width(22).PassThrough().
			LinearGradient(ui.LinearGradient{Angle: 90, Stops: []ui.GradientStop{{Color: bg, At: 0}, {Color: clear, At: 1}}})
	}
	if right {
		ui.Box(c).Key("etab-fade-r").Absolute().Right(0).Top(0).Bottom(0).Width(22).PassThrough().
			LinearGradient(ui.LinearGradient{Angle: 270, Stops: []ui.GradientStop{{Color: bg, At: 0}, {Color: clear, At: 1}}})
	}
}

func (a *App) etab(c *ui.Context, tb *ideTab) { a.etabPane(c, tb, 0) }

func (a *App) etabPane(c *ui.Context, tb *ideTab, pane int8) {
	t := a.tk
	d := a.ide
	on := tb.id == *d.paneActive(pane)
	el := ui.ButtonBase(c).Key("etab-" + tb.id).PaddingX(10).Gap(6).AlignItems(ui.Center).Shrink(0).
		TextColor(t.Tx3).Cursor(ui.CursorDefault).Transition(kit.Fade)
	narrowWs := a.wsNarrow() // 窄档(<520):标签收紧到 130,非当前标签只留 70px 标题
	if narrowWs {
		el.PaddingX(8).MaxWidth(130)
	}
	if isFileTab(tb) {
		dragRef(el, tb.path, false) // 拖到对话栏 = 引用这个文件
	}
	if a.ide.splitDir != 0 {
		a.etabDrag(el, tb, pane)
	}
	el.ContextMenu(func(m *ui.Menu) { a.tabMenu(m, tb) })
	// 鼠标中键关闭标签(对标 VS Code / 浏览器)
	a.onMiddleClick(el, tb.id, func() {
		ideLog.Info("中键关闭标签", "tab", tb.id)
		a.closeTabOnPane(tb.id, pane)
	})
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
		el.Radius(t.RSm()).Height(theme.Px(24))
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
		if tb.virt == "" { // 文件 / 差异标签:与文件树一致的类型图标(认识的语言用徽标)
			a.fileIc(c, path.Base(tb.path))
		} else {
			ic := kit.Ic(c, a.tabIcon(tb), kit.S12)
			if tb.virt == vReview || tb.virt == vPlan {
				ic.TextColor(t.Ac)
			}
		}
		nm := ui.Text(c, a.tabTitle(tb)).SingleLine()
		if narrowWs && !on {
			nm.MaxWidth(70)
		}
		if tb.preview {
			nm.Italic()
		}
		if a.isExternalTab(tb) {
			a.externalChip(c, tb)
		}
		if tb.extDeleted && !tb.extOrphan {
			nm.TextColor(t.Tx4).Strikethrough()
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
			a.closeTabOnPane(tb.id, pane)
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
