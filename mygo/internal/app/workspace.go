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

// stage 是舞台:按形态排布工作区与对话栏。工作区与对话栏各是一个带 Key 的元素,换形态时只是换了先后顺序
// (设计稿 .ws / .chat 的 order),所以内容、编辑器状态都不会重建;宽度变化走 440 ms 变形过渡。
func (a *App) stage(c *ui.Context) {
	gap := a.tk.Scheme.GapPane
	// 形态切换的传送带平移:面板仍是同一个 Row 里带 Key 的同一批元素(身份不变,输入框焦点 / 编辑器状态不丢),
	// 只是平移期间它们改成绝对定位 + 逐帧摆位(见 morph_belt.go),Row 裁切溢出。
	f := a.beltStep()
	row := ui.Row(c).Key("stage").Grow(1).MinWidth(0).AlignItems(ui.Stretch)
	if f.on {
		row.Clip()
	}
	a.stageFactors(row) // 开合因子先于两块面板算好(它们互相依赖)
	// 平移中也要每帧更新:舞台是 Grow(1) 的 Row,宽度与绝对定位的子面板无关;
	// 左栏 / 右栏在切换里开合时舞台宽度会跟着 440ms 过渡,终点几何每帧按最新宽度重算,才不会在最后一帧跳宽。
	if w := row.Bounds().W; w > 0 {
		a.stageW = w
	}
	a.probe(pzStage, row)
	row.Children(func() {
		a.belt = f
		defer func() { a.belt = beltFrame{} }()
		if f.on && f.ghOn {
			a.wsGhost(c, f.ghost) // 被挤出舞台的旧工作区:最先画,新工作区后画
		}
		if a.screen != ScreenWork && a.screen != ScreenWelcome {
			a.guard(c, "页面", "page:"+string(a.screen), guardCard, func() {
				if !a.page(c) {
					a.pagePlaceholder(c)
				}
			})
			return
		}
		chat := func() { a.guard(c, "对话", "chat", guardCard, func() { a.chatPane(c, gap) }) }
		if a.mode == ModeIDE { // IDE:工作区在主位(左),对话栏停靠在右
			a.wsPane(c, gap)
			chat()
		} else { // Agent:对话在主位,工作区在右侧
			chat()
			a.wsPane(c, gap)
		}
	})
}

// wsOccupies 工作区此刻是否占位(Agent = 右栏开着且不是浮层;IDE = 没被对话占满)。
func (a *App) wsOccupies() bool {
	if a.mode == ModeIDE {
		return !a.chatMax
	}
	return !a.useWsOverlay() && a.rightOn
}

// wsActive 工作区此刻是否在左(IDE 主区,left = true)或右(Agent 右栏)占位。
func (a *App) wsActive(left bool) bool {
	return a.wsOccupies() && (a.mode == ModeIDE) == left
}

// wsPane 是工作区:IDE 形态在主位(占满),Agent 形态在右侧(默认 560,收起时宽度归零)。
func (a *App) wsPane(c *ui.Context, gap float32) {
	active := a.wsOccupies()
	ide := a.mode == ModeIDE
	// 宽度 / 间距 = 开合因子(stageFactors)× 满开值,不用元素过渡(见 paneOpen 的说明)
	p := a.pane(c).Key("ws").ClipX().Transition(noMotion)
	if a.tk.Glass() {
		p.Material(kit.GlassMat(kit.ClaritySidebar)) // 工作区比对话更通透:壁纸的色晕能透出一点
	}
	switch {
	case a.belt.on:
		// 形态切换的传送带平移:新工作区绝对定位 + 固定终态宽度,自与对话栏相邻的一侧移进来(morph_belt.go)
		r := a.belt.ws
		p.Absolute().Left(r.x).Top(0).Bottom(0).Width(max(0, r.w)).Shrink(0).Transition(noMotion)
		active = a.belt.wsOn
	case ide:
		// IDE:工作区永远是「占满剩余」(设计稿 .ws { width: calc(100% - --w-dock - gap) } 的等价写法):
		// 对话栏的宽度才是动画值(stageFactors 的 fDock / fMax),工作区每一帧拿到的就是舞台减去对话栏后的剩余,
		// 两块面板加起来恒等于舞台宽度——开合、最大化、缩放窗口、收放导航栏任意叠加都不会出现空白或重叠。
		// 对话栏最大化走完后工作区归零(内容再保留 440ms 随宽度被裁掉,见 keepWhile)。
		if a.fMax >= .999 {
			p.Width(0).Shrink(0)
		} else {
			p.Grow(1).Basis(0)
		}
	default:
		// Agent:右侧工作区 = 开合因子 × 满开宽度,左缘外边距同因子渐变(设计稿 .pane 的 width / margin 过渡)
		p.Width(a.fWs*a.wsRW()).Shrink(0).Margin(0, 0, 0, a.fWs*gap)
	}
	// 透明度 260ms ease-io(设计稿 .pane);宽度走 440ms 变形曲线(上面的 Transition)。平移中直接到位。
	opDur := kit.DurSlow
	if a.belt.on {
		opDur = time.Millisecond
	}
	p.Opacity(p.AnimateWith("op", map[bool]float32{true: 1, false: 0}[active], opDur, kit.EaseIO))
	a.probe(pzWs, p)
	// 收起期间继续画内容,被宽度动画一点点裁掉(设计稿里内容一直在 DOM 里)
	if !a.keepWhile("ws", active, kit.DurMorph) {
		p.Children(func() {})
		return
	}
	if w := p.Bounds().W; w > 0 {
		a.wsW = w
	}
	p.Children(func() {
		inner := ui.Column(c).Grow(1).MinHeight(0)
		a.zoneMarkEditor(inner)
		if !ide {
			if a.belt.on { // 平移中内容宽度跟着面板(被对话栏挤窄时逐帧收窄),不是终点宽度
				inner.Width(max(0, a.belt.ws.w))
			} else {
				inner.Width(a.wsRW())
			}
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
	el := kit.Since(a.flashAt)
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
	if tb.pinned {
		el.PaddingX(6)
		if !on {
			el.MaxWidth(34).Tooltip(a.tabTitle(tb))
		}
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
		showName := !tb.pinned || on
		if showName {
			nm := ui.Text(c, a.tabTitle(tb)).SingleLine()
			if narrowWs && !on {
				nm.MaxWidth(70)
			}
			if tb.preview {
				nm.Italic()
			}
			if tb.extDeleted && !tb.extOrphan {
				nm.TextColor(t.Tx4).Strikethrough()
			}
		}
		if a.isExternalTab(tb) {
			a.externalChip(c, tb)
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
