package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 左栏导航分段:会话 / 文件 / 搜索 / 源代码管理。
var navs = []struct{ id, icon, name string }{
	{"chat", "chat", "会话"}, {"files", "folder", "文件"}, {"search", "search", "搜索"}, {"scm", "branch", "源代码管理"},
}

// 整页入口:自动化 / 画布 / 浏览器 / 自定义。
var pages = []struct {
	icon, name string
	screen     Screen
}{
	{"timer", "自动化", ScreenAutomation}, {"canvas", "画布", ScreenCanvas}, {"globe", "浏览器", ScreenBrowser}, {"plug", "自定义", ScreenCustomize},
}

// SetProjects 设置侧栏的项目与会话。
func (a *App) SetProjects(p []core.ProjectInfo, current string) {
	a.rawProj = p
	a.projects = a.applyMeta(p)
	p = a.projects
	a.curSess = current
	for _, pr := range p {
		if pr.ID == "web-dashboard" { // 演示:设计稿里 web-dashboard 默认展开
			a.projOpen[pr.ID] = true
		}
		for _, s := range pr.Sessions {
			if s.ID == current {
				a.projOpen[pr.ID] = true
			}
		}
	}
}

// RefreshProjects 只更新项目与会话列表(定时重扫用),不动当前会话与展开状态。
func (a *App) RefreshProjects(p []core.ProjectInfo) {
	a.rawProj = p
	a.projects = a.applyMeta(p)
}

// SetProjectRefresher 注入立即重扫会话索引(自动化「查看会话」用)。
func (a *App) SetProjectRefresher(f func() []core.ProjectInfo) { a.projRefresh = f }

func (a *App) sideWidth() float32 {
	if a.tk.Scheme.ID == "c" {
		return 212
	}
	return theme.WSide
}

// side 是导航栏:顶部分段 + 内容 + 底部页面入口。宽度变化带 260 ms 过渡,内容宽度固定所以只是被裁切。
func (a *App) side(c *ui.Context) {
	t := a.tk
	w := a.sideW()
	// 窄窗改抽屉:并排列不能「一帧消失」(内容区会一下子变宽几百像素),而是和手动收起一样 260 ms 收到 0;
	// 抽屉里的导航内容在 overlayLayer 画,并排列只在还没收完时保留内容(抽屉已展开时不重复画)
	inline := a.sideOpen && !a.useSideDrawer()
	drawer := a.useSideDrawer()
	// 开合:宽度 260ms ease-io(sideTransition),透明度 120ms ease(设计稿 .side 的 transition)
	// 宽度是显式的逐帧布局宽度 = 开合因子(paneOpen,260ms ease-io)× 满开宽度:舞台与对话栏每一帧跟着它走;
	// 不用元素过渡(见 paneOpen 的说明)。本帧宽度记进 sideCur,右栏据此实时预判自己的宽度。
	col := ui.Column(c).Key("side").Shrink(0).ClipX().Transition(noMotion)
	cur := a.paneOpenFree(col, "w", inline, true) * w
	col.Width(cur)
	a.sideCur, a.sideCurOk = cur, true
	a.probe(pzSide, col)
	col.Opacity(col.AnimateWith("op", map[bool]float32{true: 1, false: 0}[inline], kit.DurFast, kit.EaseCSS))
	a.syncNavSwap()
	a.syncNavDrop()
	if drawer && (a.sideOpen || cur < 0.5) {
		col.Children(func() {})
		return
	}
	col.Children(func() {
		if inline {
			a.resizeHandle(c, lySide, "r", w, a.sideWidth(), 180, 420, func(v float32) { a.ly.side = v })
		}
		ui.Column(c).Width(w).Grow(1).MinHeight(0).Children(func() {
			a.sideTop(c)
			ui.Column(c).Grow(1).MinHeight(0).PaddingX(6).Clip().Children(func() {
				// 换分段 / 换形态:内容自左 8px 滑入并淡入(a-swap);外层 Clip 把滑出左缘的部分裁掉
				swap := ui.Column(c).Key("nav-swap").Grow(1).MinHeight(0)
				// 整块只淡入(行以外的输入框、标题等);列表行另有自上而下的逐行落下(navDrop,见 side_drop.go)
				if p := kit.SwapProgress(a.navSwapAt, a.frameSoon); p < 1 {
					swap.Opacity(p)
				}
				swap.Children(func() {
					a.autoReveal()
					a.morphWrap(c, "nav", func() {
						switch a.NavID() {
						case "chat":
							a.sessions(c)
						case "files":
							a.filesView(c)
						case "search":
							a.searchView(c)
						case "scm":
							a.scmView(c)
						default:
							ui.Text(c, navTitle(a.NavID())).TextColor(t.Tx3).Margin(8)
						}
					})
				})
			})
			if t.Scheme.ID != "c" {
				a.sideFoot(c)
			}
		})
	})
}

func navTitle(id string) string {
	for _, n := range navs {
		if n.id == id {
			return n.name
		}
	}
	return ""
}

func (a *App) sideTop(c *ui.Context) {
	t := a.tk
	r := ui.Row(c).Gap(4).Padding(2, 8, 6, 8).Shrink(0).AlignItems(ui.Center)
	if t.Scheme.ID == "c" {
		r.Height(theme.HTab).Padding(0, 6, 0, 8)
	}
	r.Children(func() {
		if t.Scheme.ID == "c" {
			ui.Text(c, navTitle(a.NavID())).FontWeight(500).PaddingStart(6).Grow(1)
			a.addMenuButton(c)
			return
		}
		items := make([]kit.SegItem, len(navs))
		on := 0
		nChg, conflict := a.scmBadge()
		for i, n := range navs {
			items[i] = kit.SegItem{Icon: n.icon, Tip: n.name}
			if n.id == "scm" { // 有未提交改动时,图标右上角亮一个点,不用切过去也知道
				items[i].Tip = scmTip(nChg, conflict)
				items[i].Dot, items[i].DotKind = nChg > 0, scmDotKind(conflict)
			}
			if n.id == a.NavID() {
				on = i
			}
		}
		if i := kit.Seg(c, items, on, kit.SegOpt{Icons: true, Fill: true, Key: "navseg"}); i >= 0 {
			old := a.nav
			a.nav = navs[i].id
			if old != a.nav {
				diag.LogUISwitch("nav:"+old, "nav:"+a.nav, 0)
			}
			a.leavePage()
			a.focusSearchOnEnter(navs[i].id)
		}
	})
}

func (a *App) sideFoot(c *ui.Context) {
	ui.Row(c).Gap(2).Padding(6, 8).Shrink(0).Children(func() {
		for _, p := range pages {
			kit.Ib(c, p.icon, kit.IbOpt{Tip: p.name, On: a.screen == p.screen}).OnClick(func() { a.togglePage(p.screen) })
		}
		ui.Spacer(c).Grow(1)
		g := kit.Ib(c, "gear", kit.IbOpt{Tip: "账号与设置", On: a.popup == "acct@gear"})
		g.OnClick(func() { a.togglePop("acct@gear") })
		a.acctPopup(c, "gear", g)
		// 开发版 / 演示模式徽章只在右下角状态栏显示一处,侧栏底部不再重复。
	})
}

// sessions 是会话列表:方案 A / C 按项目分组,方案 B 只看当前项目并按时间分段。
func (a *App) sessions(c *ui.Context) {
	t := a.tk
	if t.Scheme.ID == "b" {
		a.sessionsB(c)
		return
	}
	// 「新对话」与「项目与工作区」标题行(含「+」)固定在上方,只有下面的列表滚动。
	ui.Column(c).Grow(1).MinHeight(0).Children(func() {
		if t.Scheme.ID != "c" {
			r := kit.RowEl(c, false).Margin(0, 0, 4, 0).Gap(6).Shrink(0)
			r.OnClick(a.NewSession)
			r.Children(func() {
				kit.Ic(c, "plus", kit.S14).TextColor(t.Ac)
				ui.Text(c, "新对话").Grow(1).FontWeight(500).TextColor(t.Tx1)
				kit.Kbd(c, "⌘", "N")
			})
		}
		a.projectsHeader(c, t.Scheme.ID != "c")
		kit.VScroll(c).Key("sess").Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).Children(func() { a.projectsList(c) })
		})
	})
}

// projRow 项目行:点行 = 选中这个项目(聊天栏与 IDE 一起切过来)并展开;已选中时再点 = 展开 / 收起;没有行首箭头,图标本身表达展开(folder ⇄ folderOpen)。
func (a *App) projRow(c *ui.Context, p core.ProjectInfo) {
	t := a.tk
	open := a.projOpen[p.ID]
	on := a.ctxIsProject(p)
	r := a.navDrop(kit.RowEl(c, false).Key("proj-" + p.ID)).Gap(5).TextColor(t.Tx1).FontWeight(500)
	hov := r.Hovered()
	r.OnClick(func() {
		if !on {
			a.selectProject(p)
			return
		}
		a.projOpen[p.ID] = !open
	})
	r.ContextMenu(func(m *ui.Menu) { a.projMenu(m, p) })
	r.Children(func() {
		// 没有行首箭头:图标本身表达展开(展开 = 翻开的文件夹),整行可点
		name := "folder"
		if open {
			name = "folderOpen"
		}
		ic := kit.Ic(c, name, kit.S16).TextColor(t.Tx3)
		if on {
			ic.TextColor(t.Ac)
		}
		ui.Text(c, p.Name).Grow(1).SingleLine()
		if hov {
			kit.Ib(c, "plus", kit.IbOpt{Small: true, Tip: "在此项目新建会话"}).Margin(0, -4, 0, 0).OnClick(func() { a.NewSessionIn(p) })
		} else if a.registry().HasProject(p.Path) && !a.isRepoDir(p.Path) {
			kit.Tag(c, "非仓库")
		}
		// 项目行末尾不再显示会话数量:数字对找会话没有帮助,还让每行右侧参差不齐
	})
}

func (a *App) sessRow(c *ui.Context, s core.SessionInfo, indent bool) {
	t := a.tk
	on := s.ID == a.curSess
	r := a.navDrop(kit.RowEl(c, on).Key("sess-" + s.ID))
	if t0, hide := a.sessHide[s.ID]; hide {
		kit.CollapseHide(r, t0, t.Scheme.RowH, a.frameSoon)
	}
	if indent {
		// 整行(含选中高亮与左侧竖条)用外边距缩进,而不是在行内左边留一块空白:
		// 会话标题落在项目名右侧约 14px,高亮从项目图标之后开始。
		r.Margin(0, 0, 0, 18)
	}
	hov := r.Hovered()
	if a.ren.id == s.ID { // 行内重命名:整行换成输入框
		r.Children(func() { a.renameInput(c, s.ID) })
		return
	}
	ws := a.sessCtxWs // 工作区下的会话:点它先进入该工作区(IDE 跟着换成多文件夹)
	r.OnClick(func() {
		if ws != "" && a.activeWsID() != ws {
			a.ActivateWorkspace(ws)
		}
		a.OpenSession(s.ID)
	})
	r.ContextMenu(func(m *ui.Menu) { a.sessionMenu(m, s.ID, s.Title, false) })
	r.Children(func() {
		if on {
			kit.SelBar(c) // 选中行左侧一道强调色竖线(所有布局方案统一)
		}
		state := a.sessState(s.ID)
		ui.Row(c).Width(14).Height(14).Justify(ui.Center).AlignItems(ui.Center).Shrink(0).Margin(0, -2, 0, 0).Children(func() {
			switch state {
			case "run":
				a.glyph(c, t.Ac)
			case "wait":
				ui.Box(c).Size(6, 6).Radius(3).Background(t.Warn)
			case "unread":
				ui.Box(c).Size(6, 6).Radius(3).Background(t.Ac)
			}
		})
		if s.Pinned {
			kit.Ic(c, "pin", kit.S12).TextColor(t.Ac).Tooltip("已置顶")
		}
		title := ui.Text(c, s.Title).Grow(1).SingleLine()
		if s.Archived {
			title.TextColor(t.Tx3)
		}
		if hov {
			kit.Ib(c, "more", kit.IbOpt{Small: true, Tip: "更多操作"}).Margin(0, -4, 0, 0).Menu(func(m *ui.Menu) { a.sessionMenu(m, s.ID, s.Title, false) })
		} else {
			lbl := s.Label
			switch state {
			case "run":
				lbl = "运行中"
			case "wait":
				lbl = "待确认"
			}
			ui.Text(c, lbl).FontSize(theme.FsCap).TextColor(t.Tx3).FontFeatures("tnum").SingleLine()
		}
	})
}

// sessionsB 方案 B:当前项目 / 工作区选择条(固定,带「新对话」与「+」添加入口)+ 今天 / 昨天 / 更早(只有这部分滚动)。
func (a *App) sessionsB(c *ui.Context) {
	name, icon, proj := a.sideBCtx()
	ui.Column(c).Grow(1).MinHeight(0).Children(func() {
		ui.Row(c).Gap(4).Padding(0, 0, 6, 0).Shrink(0).AlignItems(ui.Center).Children(func() {
			chip := kit.Chip(c, name, kit.ChipOpt{Icon: icon, Line: true, Car: true, On: a.popup == "projpick"}).Height(theme.Px(26)).Grow(1).MinWidth(0)
			chip.OnClick(func() { a.togglePop("projpick") })
			a.projPickPopup(c, chip)
			kit.Ib(c, "edit", kit.IbOpt{Tip: "新对话 ⌘N"}).Size(26, 26).OnClick(a.NewSession)
			a.addMenuButton(c)
		})
		kit.VScroll(c).Key("sessb").Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).Children(func() {
				cur := ""
				live, arch := sessionsSplit(proj) // 置顶的排最前
				if len(live) == 0 && len(arch) == 0 {
					ui.Text(c, "还没有会话").FontSize(theme.FsCap).TextColor(a.tk.Tx4).Padding(8)
				}
				for _, ss := range live {
					g := ss.Group
					if ss.Pinned {
						g = "置顶"
					}
					if g != cur {
						cur = g
						a.navDrop(kit.Grp(c, cur))
					}
					ss.Label = ss.Clock
					a.sessRow(c, ss, false)
				}
				a.archiveFold(c, proj.ID, arch, 8, false) // 归档的收在底部,可展开、可取消归档
			})
		})
	})
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// sideTransition 是导航栏开合的宽度过渡(拖动调整宽度期间关掉)。
func (a *App) sideTransition() ui.ElementTransition {
	if a.instantLayout() {
		return noMotion
	}
	return ui.ElementTransition{Size: true, Duration: kit.DurSlow, Ease: kit.EaseIO}
}

// leavePage 离开整页功能(自动化 / 画布 / 浏览器 / 自定义 / 用量)回到工作台;设置页与已在工作台时不动。
// 点导航分段(会话 / 文件 / 搜索 / 源代码管理)是「去看那一块」,所以也要先出整页,否则点了没反应。
func (a *App) leavePage() {
	for _, p := range pages {
		if a.screen == p.screen {
			a.screen = ScreenWork
			return
		}
	}
	if a.screen == ScreenUsage {
		a.screen = ScreenWork
	}
}

// togglePage 点整页入口图标:已在该页再点一次 = 回到工作台(给整页一个出口),否则进入。
func (a *App) togglePage(s Screen) {
	if a.screen == s {
		a.screen = ScreenWork
		return
	}
	a.screen = s
}
