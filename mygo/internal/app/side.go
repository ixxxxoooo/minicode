package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
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
	a.projects = p
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

func (a *App) sideWidth() float32 {
	if a.tk.Scheme.ID == "c" {
		return 212
	}
	return theme.WSide
}

// side 是导航栏:顶部分段 + 内容 + 底部页面入口。宽度变化带 260 ms 过渡,内容宽度固定所以只是被裁切。
func (a *App) side(c *ui.Context) {
	t := a.tk
	w := a.sideWidth()
	target := w
	if !a.sideOpen {
		target = 0
	}
	ui.Column(c).Key("side").Width(target).Shrink(0).ClipX().
		Opacity(map[bool]float32{true: 1, false: 0}[a.sideOpen]).
		Transition(ui.ElementTransition{Size: true, Duration: kit.DurSlow, Ease: kit.EaseIO}).Children(func() {
		ui.Column(c).Width(w).Grow(1).MinHeight(0).Children(func() {
			a.sideTop(c)
			ui.Column(c).Grow(1).MinHeight(0).PaddingX(6).Clip().Children(func() {
				switch a.NavID() {
				case "chat":
					a.sessions(c)
				case "files":
					a.filesView(c)
				default:
					ui.Text(c, navTitle(a.NavID())).TextColor(t.Tx3).Margin(8)
				}
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
			ui.Text(c, navTitle(a.NavID())).FontWeight(600).PaddingStart(6).Grow(1)
			kit.Ib(c, "plus", kit.IbOpt{Small: true, Tip: "新建"})
			return
		}
		items := make([]kit.SegItem, len(navs))
		on := 0
		for i, n := range navs {
			items[i] = kit.SegItem{Icon: n.icon, Tip: n.name}
			if n.id == a.NavID() {
				on = i
			}
		}
		if i := kit.Seg(c, items, on, kit.SegOpt{Icons: true, Fill: true, Key: "navseg"}); i >= 0 {
			a.nav = navs[i].id
		}
	})
}

func (a *App) sideFoot(c *ui.Context) {
	ui.Row(c).Gap(2).Padding(6, 8).Shrink(0).Children(func() {
		for _, p := range pages {
			kit.Ib(c, p.icon, kit.IbOpt{Tip: p.name, On: a.screen == p.screen}).OnClick(func() { a.screen = p.screen })
		}
		ui.Spacer(c).Grow(1)
		g := kit.Ib(c, "gear", kit.IbOpt{Tip: "账号与设置", On: a.popup == "acct@gear"})
		g.OnClick(func() { a.togglePop("acct@gear") })
		a.acctPopup(c, "gear", g)
		if a.devBadge != "" {
			// 开发版徽章(设计稿之外的开发辅助:主题色小标签)
			kit.Tag(c, a.devBadge, kit.KAc)
		}
	})
}

// sessions 是会话列表:方案 A / C 按项目分组,方案 B 只看当前项目并按时间分段。
func (a *App) sessions(c *ui.Context) {
	t := a.tk
	if t.Scheme.ID == "b" {
		a.sessionsB(c)
		return
	}
	ui.Scroll(c).Key("sess").Grow(1).Children(func() {
		ui.Column(c).Children(func() {
			if t.Scheme.ID != "c" {
				r := kit.RowEl(c, false).Margin(0, 0, 4, 0).Gap(6)
				r.OnClick(a.NewSession)
				r.Children(func() {
					kit.Ic(c, "plus", kit.S14).TextColor(t.Ac)
					ui.Text(c, "新对话").Grow(1).FontWeight(500).TextColor(t.Tx1)
					kit.Kbd(c, "⌘", "N")
				})
			}
			for _, p := range a.projects {
				a.projRow(c, p)
				if a.projOpen[p.ID] {
					for i, s := range p.Sessions {
						if i >= 4 {
							break
						}
						a.sessRow(c, s, true)
					}
				}
			}
		})
	})
}

func (a *App) projRow(c *ui.Context, p core.ProjectInfo) {
	t := a.tk
	open := a.projOpen[p.ID]
	r := kit.RowEl(c, false).Key("proj-" + p.ID).Gap(5).TextColor(t.Tx1).FontWeight(500)
	hov := r.Hovered()
	r.OnClick(func() { a.projOpen[p.ID] = !open })
	r.Children(func() {
		rot := float32(0)
		if open {
			rot = 90
		}
		car := kit.Ic(c, "chevR", kit.S12).TextColor(t.Tx4)
		car.Rotate(car.AnimateWith("rot", rot, kit.DurBase, kit.EaseOut))
		kit.Ic(c, "folder", kit.S12).TextColor(t.Tx3)
		ui.Text(c, p.Name).Grow(1).SingleLine()
		if hov {
			kit.Ib(c, "plus", kit.IbOpt{Small: true, Tip: "在此项目新建会话"}).Margin(0, -4, 0, 0)
		} else {
			ui.Text(c, itoa(p.Count)).FontSize(theme.FsMicro).FontWeight(400).TextColor(t.Tx4).FontFeatures("tnum")
		}
	})
}

func (a *App) sessRow(c *ui.Context, s core.SessionInfo, indent bool) {
	t := a.tk
	on := s.ID == a.curSess
	r := kit.RowEl(c, on).Key("sess-" + s.ID)
	if indent {
		r.PaddingStart(25)
	}
	hov := r.Hovered()
	r.OnClick(func() { a.OpenSession(s.ID) })
	r.Children(func() {
		if on && t.Scheme.ID == "a" {
			// 方案 A:选中行左侧一道强调色竖线
			ui.Box(c).Absolute().Left(0).Top(5).Bottom(5).Width(2).Radius(2).Background(t.Ac)
		}
		ui.Text(c, s.Title).Grow(1).SingleLine()
		if hov {
			kit.Ib(c, "more", kit.IbOpt{Small: true}).Margin(0, -4, 0, 0)
		} else {
			ui.Text(c, s.Label).FontSize(theme.FsMicro).TextColor(t.Tx4).FontFeatures("tnum").SingleLine()
		}
	})
}

// sessionsB 方案 B:当前项目选择条 + 今天 / 昨天 / 更早。
func (a *App) sessionsB(c *ui.Context) {
	t := a.tk
	ui.Scroll(c).Key("sessb").Grow(1).Children(func() {
		ui.Column(c).Children(func() {
			ui.Row(c).Gap(4).Padding(0, 0, 6, 0).Children(func() {
				kit.Chip(c, "router-demo", kit.ChipOpt{Icon: "folder", Line: true, Car: true}).Height(26).Grow(1)
				kit.Ib(c, "edit", kit.IbOpt{Tip: "新对话 ⌘N"}).Size(26, 26).OnClick(a.NewSession)
			})
			cur := ""
			for _, p := range a.projects {
				if p.ID != "router-demo" {
					continue
				}
				for _, ss := range p.Sessions {
					if ss.Group != cur {
						cur = ss.Group
						kit.Grp(c, cur)
					}
					ss.Label = ss.Clock
					a.sessRow(c, ss, false)
				}
			}
			_ = t
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
