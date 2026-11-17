package app

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// motionEntry 登记表里的一条动效(与 docs/design/js/motions-overview.js 同步)。
type motionEntry struct {
	Group, ID, Title, Trigger, Dur, Ease, Files string
	ReduceOK                                    bool
}

var motionCatalogEntries = []motionEntry{
	{Group: "主界面切换", ID: "morph", Title: "Agent ⇄ IDE 布局变形", Trigger: "点标题栏切换 / ⌘E", Dur: "440ms", Ease: "ease-morph", Files: "morph_belt.go · motion_layout.go", ReduceOK: true},
	{Group: "主界面切换", ID: "nav-seg", Title: "导航分段下划线", Trigger: "点侧栏顶部分段", Dur: "180ms", Ease: "ease-spring", Files: "side.go", ReduceOK: true},
	{Group: "会话与列表", ID: "flow-rise", Title: "会话流行落下", Trigger: "切换会话后视野内各行", Dur: "30ms/行,封顶 400ms", Ease: "ease-out", Files: "flow_rise.go", ReduceOK: true},
	{Group: "会话与列表", ID: "nav-drop", Title: "导航列表落下", Trigger: "换分段 / 形态 / 项目", Dur: "8ms/行", Ease: "ease-out", Files: "side_drop.go", ReduceOK: true},
	{Group: "弹层与菜单", ID: "pop-in", Title: "弹层升起", Trigger: "圆环 / 菜单 / 气泡", Dur: "120–180ms", Ease: "ease-out", Files: "kit/popshell.go", ReduceOK: true},
	{Group: "弹层与菜单", ID: "dialog", Title: "居中对话框", Trigger: "命令面板 / 新建 / 确认", Dur: "180ms", Ease: "ease-out", Files: "popups.go", ReduceOK: true},
	{Group: "弹层与菜单", ID: "drawer", Title: "侧滑抽屉", Trigger: "窄窗导航 / 工作区浮层", Dur: "260ms", Ease: "ease-out", Files: "overlays.go · kit.DrawerSlide", ReduceOK: true},
	{Group: "反馈状态", ID: "toast", Title: "提示条滑入", Trigger: "复制 / 保存", Dur: "260ms", Ease: "ease-out", Files: "kit.ToastSlide", ReduceOK: true},
	{Group: "反馈状态", ID: "skeleton", Title: "骨架 → 内容", Trigger: "用量页首次有数据", Dur: "700ms", Ease: "ease-io", Files: "usage_page.go · kit.Crossfade", ReduceOK: true},
	{Group: "反馈状态", ID: "shimmer", Title: "进行中扫光", Trigger: "状态行", Dur: "1.6s 循环", Ease: "linear", Files: "kit.RunText", ReduceOK: true},
	{Group: "反馈状态", ID: "shake", Title: "输入校验抖动", Trigger: "路径 / 表单错误", Dur: "320ms", Ease: "ease-io", Files: "kit.ShakeInput", ReduceOK: true},
	{Group: "编辑器与终端", ID: "pane-split", Title: "分栏宽度过渡", Trigger: "拖分栏", Dur: "440ms", Ease: "ease-io", Files: "layout.go", ReduceOK: true},
	{Group: "窗口布局动画", ID: "grow-y", Title: "柱状图生长", Trigger: "打开用量页", Dur: "260ms+", Ease: "ease-out", Files: "usage_page.go · kit.GrowY", ReduceOK: true},
}

func (a *App) openMotionCatalog() {
	log.Info("打开动效总览")
	a.motionCatOpen = true
}

func (a *App) closeMotionCatalog() {
	a.motionCatOpen = false
}

func (a *App) motionCatalogOverlay(c *ui.Context) {
	key := "motion-catalog"
	a.dialogOpen(key, a.motionCatOpen, func(open *bool) { /* 退场阶段 hold 为 true,勿写回 motionCatOpen */ })
	if !a.motionCatOpen && a.dlgClosing != key {
		return
	}
	t := a.tk
	ui.DialogBase(c, &a.motionCatOpen, func(backdrop, panel ui.Element) {
		backdrop.Background(t.Scrim).OnClick(a.closeMotionCatalog)
		pw := min(float32(560), a.winW-48)
		panel.Width(pw).MaxHeight(a.winH - 80).Background(t.BG).Radius(t.RLg()).Clip().Column()
		for _, s := range t.Shadow3() {
			panel.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
		}
		panel.Border(1, t.Bd1)
		a.dialogIn(key, backdrop, panel)
		panel.Children(func() {
			ui.Row(c).Padding(14, 16, 8, 16).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "动效总览").FontWeight(600).FontSize(theme.Px(15)).Grow(1)
				motionLbl := "减少动效:关"
				if !kit.MotionOn() {
					motionLbl = "减少动效:开"
				}
				kit.Btn(c, motionLbl, kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(func() {
					on := !a.prefs.Bool(pMotion, true)
					a.prefs.Set(pMotion, on)
					a.applyPrefs()
				})
				kit.Ib(c, "x", kit.IbOpt{Tip: "关闭 Esc"}).OnClick(a.closeMotionCatalog)
			})
			a.box(c).Padding(0, 16, 10, 16).Children(func() {
				a.motionPreview(c)
			})
			kit.VScroll(c).Grow(1).MinHeight(180).MaxHeight(420).Children(func() {
				ui.Column(c).Padding(0, 8, 8, 8).Gap(2).Children(func() {
					last := ""
					for _, e := range motionCatalogEntries {
						if e.Group != last {
							last = e.Group
							ui.Text(c, e.Group).FontWeight(600).Padding(8, 8, 4, 8).TextColor(t.Tx2)
						}
						a.motionCatalogRow(c, e)
					}
				})
			})
		})
	})
}

func (a *App) motionPreview(c *ui.Context) {
	t := a.tk
	a.box(c).Height(72).Padding(8, 10).Children(func() {
		ui.Text(c, "预览区").FontSize(theme.FsCap).TextColor(t.Tx3)
		inner := ui.Box(c.Key("motion-demo")).Grow(1).Height(32).Radius(t.RSm()).Background(t.Fill1).Margin(4, 0, 0, 0)
		switch a.motionReplayKey {
		case "shake":
			var at time.Time
			if !a.motionReplayAt.IsZero() {
				at = a.motionReplayAt
			}
			kit.ShakeInput(inner, &at)
		case "grow-y":
			h := float32(80)
			if kit.MotionOn() && !a.motionReplayAt.IsZero() {
				frac := float32(kit.Since(a.motionReplayAt)) / float32(kit.DurSlow)
				if frac < 1 {
					h = frac * 80
					a.frameSoon()
				}
			}
			inner.HeightPercent(h).AlignSelf(ui.End)
		default:
			if a.motionReplayKey != "" {
				kit.PopUpIn(inner, a.motionReplayAt, a.frameSoon)
			}
		}
		inner.Children(func() {
			lbl := "点清单中的 ▶ 重放"
			if a.motionReplayKey != "" {
				lbl = "重放: " + a.motionReplayKey
			}
			ui.Text(c, lbl).TextColor(t.Tx3).FontSize(theme.FsCap)
		})
	})
}

func (a *App) motionCatalogRow(c *ui.Context, e motionEntry) {
	t := a.tk
	tip := fmt.Sprintf("%s · %s · %s · 减少动效归零=%v", e.Trigger, e.Dur, e.Ease, e.ReduceOK)
	ui.Row(c).Gap(6).Padding(4, 6).AlignItems(ui.Center).Tooltip(tip).Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Children(func() {
			ui.Text(c, e.Title).FontWeight(500).SingleLine()
			ui.Text(c, e.Files).FontSize(theme.FsMicro).TextColor(t.Tx4).SingleLine()
		})
		kit.Btn(c, "▶ 重放", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(func() {
			a.motionReplayKey = e.ID
			a.motionReplayAt = time.Now()
			a.frameSoon()
		})
	})
}
