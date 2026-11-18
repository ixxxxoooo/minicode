package app

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

func noticeIcon(t *theme.Tokens, lv string) (string, ui.Color) {
	switch lv {
	case "err":
		return "errc", t.Err
	case "warn":
		return "alert", t.Warn
	case "ok":
		return "okc", t.Ok
	}
	return "info", t.Info
}

// noticeBell 标题栏铃铛(设计稿只有标题栏一处,状态栏没有);返回可挂载 noticePanel 的锚点。
func (a *App) noticeBell(c *ui.Context) ui.Element {
	a.noticeInit()
	n := 0
	if a.nt != nil {
		n = a.nt.unread
	}
	on := a.nt != nil && a.nt.panelOn
	w := ui.ButtonBase(c).Key("sb-bell").Height(18).PaddingX(5).Shrink(0).Cursor(ui.CursorPointer)
	w.OnClick(func() {
		if a.nt == nil {
			a.nt = &noticeStore{}
		}
		a.nt.panelOn = !a.nt.panelOn
		if a.nt.panelOn {
			a.markNoticesRead()
		}
	})
	if on || w.Hovered() {
		w.Background(a.tk.Fill1)
	}
	w.Children(func() {
		kit.Ic(c, "bell", kit.S12)
		if n > 0 {
			kit.Cnt(c, itoa(n), true).Margin(0, 0, 0, 2)
		}
	})
	return w
}

// noticePanel 通知历史(锚在铃铛下)。
func (a *App) noticePanel(c *ui.Context, anchor ui.Element) {
	if a.nt == nil || !a.nt.panelOn {
		return
	}
	t := a.tk
	a.popOpen("notice", func(open *bool) {
		*open = a.nt.panelOn
		ui.PopoverBase(c, anchor, open, func(panel ui.Element) {
			a.popPanel(panel, 320).Padding(0).Clip().AttachTo(anchor, ui.AnchorTopRight, ui.AnchorBottomRight).Margin(0, 0, 6, 0)
			ui.Row(c).Padding(8, 10).Gap(8).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(t.Bd1).Children(func() {
				ui.Text(c, "通知").FontWeight(500).Grow(1)
				if a.nt.unread > 0 {
					kit.Cnt(c, itoa(a.nt.unread), true)
				}
				kit.Btn(c, "全部清除", kit.BtnOpt{Variant: kit.BtnGhost, Small: true}).OnClick(func() {
					a.clearNotices()
					a.nt.panelOn = false
				})
			})
			if len(a.nt.items) == 0 {
				ui.Text(c, "没有通知").TextColor(t.Tx3).Padding(12, 10)
				return
			}
			kit.VScroll(c).Key("notice-list").MaxHeight(360).Children(func() {
				ui.Column(c).Padding(4).Children(func() {
					for _, it := range a.nt.items {
						a.noticeRow(c, it)
					}
				})
			})
		})
		if !*open {
			a.nt.panelOn = false
		}
	})
}

func (a *App) noticeRow(c *ui.Context, it noticeItem) {
	t := a.tk
	icon, col := noticeIcon(t, it.level)
	bg := t.Fill1
	if it.level == "warn" {
		bg = t.WarnSoft
	} else if it.level == "err" {
		bg = t.ErrSoft
	}
	r := ui.Column(c).Key("nt-"+itoa64(it.id)).Gap(4).Padding(8, 10).Radius(t.RSm()).Background(bg).Margin(0, 0, 4, 0)
	r.Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Start).Children(func() {
			kit.Ic(c, icon, kit.S12).TextColor(col).Margin(2, 0, 0, 0)
			ui.Column(c).Grow(1).MinWidth(0).Children(func() {
				ui.Text(c, it.title).FontWeight(500).SingleLine()
				if it.body != "" {
					ui.Text(c, it.body).FontSize(theme.FsCap).TextColor(t.Tx3).LineHeight(theme.LhCap)
				}
				if len(it.actions) > 0 {
					ui.Row(c).Gap(6).Margin(4, 0, 0, 0).Children(func() {
						for i, act := range it.actions {
							i, act := i, act
							v := kit.BtnGhost
							if i == 0 {
								v = kit.BtnPri
							}
							kit.Btn(c, act.label, kit.BtnOpt{Small: true, Variant: v}).OnClick(func() {
								act.run()
							})
						}
					})
				}
			})
			ui.Text(c, relTime(a.now(), it.at)).FontSize(theme.FsMicro).TextColor(t.Tx4)
			kit.Ib(c, "x", kit.IbOpt{Small: true, Tip: "删除"}).OnClick(func() { a.dismissNotice(it.id) })
		})
	})
}

// noticeToasts 右上角叠放(标题栏下方)(最后几条自动消失)。
func (a *App) noticeToasts(c *ui.Context) {
	if a.nt == nil || len(a.nt.toasts) == 0 {
		return
	}
	t := a.tk
	now := a.now()
	var live []noticeItem
	for _, it := range a.nt.toasts {
		// 带操作按钮的通知不自动消失(等用户点按钮或关掉),其余 6 秒后淡出
		if len(it.actions) > 0 || now.Sub(it.at) < 6*time.Second {
			live = append(live, it)
		}
	}
	a.nt.toasts = live
	if len(live) == 0 {
		return
	}
	ui.Column(c).Absolute().Right(12).Top(theme.HBar + 8).Gap(8).Width(theme.Px(300)).Children(func() {
		for _, it := range live {
			icon, col := noticeIcon(t, it.level)
			card := ui.Row(c).Key("toast-"+itoa64(it.id)).Gap(8).Padding(10, 12).Radius(t.RLg()).Background(t.Raised).Border(1, t.Bd1).AlignItems(ui.Start)
			for _, s := range t.Shadow2() {
				card.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
			}
			kit.Float(card)
			card = kit.ToastSlideKeep(card, it.at, a.frameSoon, len(it.actions) > 0)
			card.Children(func() {
				kit.Ic(c, icon, kit.S12).TextColor(col).Margin(2, 0, 0, 0)
				ui.Column(c).Grow(1).MinWidth(0).Children(func() {
					ui.Text(c, it.title).FontWeight(500).MaxLines(3)
					if it.body != "" {
						ui.Text(c, it.body).FontSize(theme.FsCap).TextColor(t.Tx3).MaxLines(2)
					}
					if len(it.actions) > 0 { // 与通知面板同样的操作按钮:第一个是主操作
						ui.Row(c).Gap(6).Margin(6, 0, 0, 0).Children(func() {
							for i, act := range it.actions {
								act, id := act, it.id
								v := kit.BtnGhost
								if i == 0 {
									v = kit.BtnPri
								}
								kit.Btn(c, act.label, kit.BtnOpt{Small: true, Variant: v}).OnClick(func() {
									act.run()
									a.dismissToast(id)
								})
							}
						})
					}
				})
				kit.Ib(c, "x", kit.IbOpt{Small: true, Tip: "关闭"}).OnClick(func() { a.dismissToast(it.id) })
			})
		}
	})
}

func itoa64(n int64) string { return itoa(int(n)) }

func relTime(now, at time.Time) string {
	d := now.Sub(at)
	switch {
	case d < time.Minute:
		return "刚刚"
	case d < time.Hour:
		return itoa(int(d/time.Minute)) + " 分钟前"
	case d < 24*time.Hour:
		return itoa(int(d/time.Hour)) + " 小时前"
	}
	return at.Format("01-02 15:04")
}
