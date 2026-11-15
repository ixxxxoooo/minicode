package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 对话栏按「自己的宽度」分三档(设计稿的容器查询,不看窗口宽度):
//
//	宽 ≥ 640   Agent 形态、工作区收起:内容居中;输入框带完整工具栏
//	中 420–639 Agent 形态带工作区:头部同宽档,工具栏收紧
//	窄 < 420   IDE 右侧停靠 352:头部标题变下拉,补「新会话 / 历史 / 占满」,任务坞合并成「任务 N」
const (
	tierNarrow = iota
	tierMid
	tierWide
)

// chatTier 返回对话栏当前的档位。宽度取上一帧量到的值;第一帧没有量过时按形态估计。
func (a *App) chatTier() int {
	w := a.chatW
	if w <= 0 {
		w = 700
		if a.mode == ModeIDE && !a.chatMax {
			w = theme.WDock
		} else if a.rightOn {
			w = 480
		}
	}
	switch {
	case w < 420:
		return tierNarrow
	case w < 640:
		return tierMid
	}
	return tierWide
}

// chatHead 是对话栏头部。宽 / 中:标题 + 分支 + 历史 + 更多;窄:标题变下拉,补上「新会话」「占满」。
func (a *App) chatHead(c *ui.Context, title string) {
	t := a.tk
	narrow := a.chatTier() == tierNarrow
	run := a.st.Phase == core.PhaseRunning
	ui.Row(c).Height(theme.HTab).Padding(0, 8, 0, 14).Gap(6).Shrink(0).AlignItems(ui.Center).Children(func() {
		ttl := ui.ButtonBase(c).Key("chat-ttl").Height(24).Gap(5).AlignItems(ui.Center).Radius(t.RSm()).Shrink(1).MinWidth(0).
			Cursor(ui.CursorDefault).Transition(kit.Fade)
		if narrow {
			ttl.PaddingX(6).Margin(0, 0, 0, -6)
			if ttl.Hovered() || a.popup == "hist" {
				ttl.Background(t.Fill1)
			}
		}
		ttl.OnClick(func() {
			if narrow {
				a.togglePop("hist")
			}
		})
		ttl.Children(func() {
			if narrow && run {
				kit.Dot(c, kit.KAc)
			}
			ui.Text(c, title).FontWeight(700).SingleLine().Shrink(1)
			if narrow {
				kit.Ic(c, "chevD", kit.S12).TextColor(t.Tx3)
			}
		})
		a.histPopup(c, ttl)
		if !narrow && a.branch != "" {
			kit.Chip(c, a.branch, kit.ChipOpt{Icon: "branch", Tip: "切换分支"}).OnClick(func() { a.OpenQuick("branch", "") })
		}
		ui.Spacer(c).Grow(1)
		if narrow {
			kit.Ib(c, "plus", kit.IbOpt{Small: true, Tip: "新会话 ⌘N"}).OnClick(a.NewSession)
		}
		kit.Ib(c, "history", kit.IbOpt{Small: true, Tip: "历史会话 ⌘⇧H", On: a.popup == "hist"}).OnClick(func() { a.togglePop("hist") })
		if a.mode == ModeIDE { // 占满与退出同一个按钮(占满后图标换成「还原」,Esc 也能退出)
			ic, tip := "max", "对话占满 ⌘⇧M"
			if a.chatMax {
				ic, tip = "min", "还原对话栏 ⌘⇧M · Esc"
			}
			kit.Ib(c, ic, kit.IbOpt{Small: true, Tip: tip, On: a.chatMax}).OnClick(a.toggleChatMax)
		}
		kit.Ib(c, "more", kit.IbOpt{Small: true, Tip: "更多:复制会话 ID / 标题、历史会话"}).Menu(func(m *ui.Menu) { a.sessionMenu(m, a.curSess, a.st.Title, true) })
	})
}

// histPop 的搜索词。
type histState struct{ q string }

// histPopup 是历史会话弹层(本项目最近的会话 + 搜索 + 全部历史)。
// 它是「跳转」而不是第二份列表:会话列表的家仍是导航栏的「会话」。
func (a *App) histPopup(c *ui.Context, anchor ui.Element) {
	t := a.tk
	a.popOpen("hist", func(open *bool) {
		ui.PopoverBase(c, anchor, open, func(panel ui.Element) {
			a.popPanel(panel, 300).AttachTo(anchor, ui.AnchorTopLeft, ui.AnchorBottomLeft).Margin(4, 0, 0, 0)
			proj := a.curProject()
			ui.Row(c).Height(28).Margin(0, 0, 4, 0).PaddingX(8).Gap(6).AlignItems(ui.Center).Radius(t.RSm()).Background(t.Fill1).Children(func() {
				kit.Ic(c, "search", kit.S12).TextColor(t.Tx3)
				ui.TextInputBase(c.Key("hist-q"), &a.hist.q).Grow(1).MinWidth(0).FontSize(theme.FsBody).Placeholder("搜索 " + proj.Name + " 的会话").
					PlaceholderColor(t.Tx4).Background(ui.Transparent).BorderWidth(0).Padding(0).FocusRing(false)
			})
			kit.Cap(c, "本项目")
			q := strings.ToLower(strings.TrimSpace(a.hist.q))
			n := 0
			live, archived := sessionsSplit(proj)
			for _, s := range live {
				if q != "" && !strings.Contains(strings.ToLower(s.Title), q) {
					continue
				}
				if n >= 6 {
					break
				}
				n++
				on := s.ID == a.curSess
				r := kit.RowEl(c, on).Key("hist-" + s.ID).Gap(6)
				r.OnClick(func() { a.popup = ""; a.OpenSession(s.ID) })
				r.Children(func() {
					if on {
						kit.Dot(c, kit.KAc)
					} else {
						ui.Box(c).Width(6).Shrink(0)
					}
					ui.Text(c, s.Title).Grow(1).SingleLine()
					ui.Text(c, s.Label).FontSize(theme.FsMicro).TextColor(t.Tx4).SingleLine().Shrink(0)
				})
			}
			if n == 0 {
				ui.Text(c, "没有匹配的会话").TextColor(t.Tx3).Padding(8, 10)
			}
			if len(archived) > 0 {
				kit.Sep(c)
				kit.Cap(c, "已归档")
				na := 0
				for _, s := range archived {
					if q != "" && !strings.Contains(strings.ToLower(s.Title), q) {
						continue
					}
					if na >= 4 {
						break
					}
					na++
					r := kit.RowEl(c, false).Key("hist-arch-" + s.ID).Gap(6)
					r.OnClick(func() { a.popup = ""; a.OpenSession(s.ID) })
					r.Children(func() {
						ui.Text(c, s.Title).Grow(1).SingleLine().TextColor(t.Tx3)
					})
				}
			}
			kit.Sep(c)
			if a.mi(c, "hist-all", "history", "全部历史…", "⌘⇧H", false).Clicked() {
				a.popup = ""
				a.OpenQuick("history", "")
			}
			if a.mi(c, "hist-new", "plus", "新会话", "⌘N", false).Clicked() {
				a.popup = ""
				a.NewSession()
			}
		})
	})
}

// curProject 当前会话所在的项目(找不到取第一个)。
func (a *App) curProject() core.ProjectInfo {
	for _, p := range a.projects {
		for _, s := range p.Sessions {
			if s.ID == a.curSess {
				return p
			}
		}
	}
	if len(a.projects) > 0 {
		return a.projects[0]
	}
	return core.ProjectInfo{Name: "项目"}
}

// openTaskTarget 打开任务对应的工作区标签:子 Agent 开它的记录,其它开任务输出。
func (a *App) openTaskTarget(tk *core.BackgroundTask) {
	if tk.Kind == core.TaskAgent {
		var walk func(items []core.Item) *core.SubagentCard
		walk = func(items []core.Item) *core.SubagentCard {
			for _, it := range items {
				if s, ok := it.(*core.SubagentCard); ok {
					if s.TaskID == tk.TaskID || s.ID == tk.ToolUseID {
						return s
					}
					if r := walk(s.Children); r != nil {
						return r
					}
				}
			}
			return nil
		}
		if s := walk(a.st.Items); s != nil {
			a.OpenWorkspaceTab(vSub, s.ID)
			return
		}
	}
	a.OpenWorkspaceTab(vTask, tk.TaskID)
}
