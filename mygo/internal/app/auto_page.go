package app

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/automation"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 自动化整页(设计稿 pages.js):本周周历 + 卡片列表 + 右侧编辑抽屉。
// 清单与计划在 internal/automation;运行器接上 Claude 后端(t8)后「立即运行」才会真的开会话。

type autoState struct {
	sel    int // 选中的自动化 ID
	filter int // 0 全部 / 1 运行中 / 2 已暂停
	weekOf int // 相对本周的周数偏移

	// 抽屉里的编辑草稿:换选中项时从存储重新载入,点「保存」才写回
	draftID          int
	name, prompt, tm string
	days, perm, proj string
	errMsg           string
	focusName        bool
	runView          string // 正在查看的运行记录键
	savedPopAt       time.Time
	kindFoldAt       time.Time

	// 新建模态
	newDlg                      bool
	editDlg                     bool // 窄窗下的编辑浮层(宽窗用右侧抽屉);只在用户点了卡片 / 计划后才弹出
	newName, newPrompt, newTime string
	newDays, newPerm, newProj   string
	newErr                      string
	newErrAt                    string // 错误属于哪个字段:name / prompt / time
	newErrVal                   string // 出错时该字段的内容;内容变了错误就不再显示
}

var autoPerms = []string{"只读 · 计划模式", "逐项确认", "自动编辑"}

// SetAutoSel 选中一条自动化(测试 / 截图用)。
func (a *App) SetAutoSel(id int) { a.au.sel = id }

// autoPick 点了某条自动化:宽窗换右侧抽屉,窄窗弹出编辑浮层。
func (a *App) autoPick(id int) {
	a.au.sel = id
	if a.pageNarrowWide() {
		a.au.editDlg = true
		log.Info("窄窗打开自动化编辑浮层", "id", id)
	}
}

func (a *App) autoLoadDraft() {
	s := &a.au
	it, ok := a.auto.Get(s.sel)
	if !ok {
		if l := a.auto.List(); len(l) > 0 {
			it = l[0]
			s.sel = it.ID
		} else {
			s.draftID = 0
			return
		}
	}
	if s.draftID == it.ID {
		return
	}
	s.draftID, s.name, s.prompt, s.tm, s.days, s.perm, s.proj, s.errMsg = it.ID, it.Name, it.Prompt, it.Time, it.Days, it.Perm, it.Project, ""
}

func (a *App) autoPage(c *ui.Context) {
	items := a.auto.List()
	a.autoLoadDraft()
	a.pane(c).Grow(1).Children(func() {
		a.pageHead(c, "自动化", "让 Claude 按计划自动做事:定时检查、生成报告、巡检依赖。每次运行都会留下一个可回看的会话。", func() {
			if kit.Btn(c, "新建自动化", kit.BtnOpt{Variant: kit.BtnPri, Icon: "plus"}).Clicked() {
				a.openAutoNewDlg()
			}
		})
		narrow := a.pageNarrowWide()
		if narrow {
			ui.Scroll(c).Key("au-main-n").Grow(1).MinHeight(0).Children(func() {
				ui.Column(c).Padding(4, 20, 24, 20).Children(func() {
					a.autoWeek(c, items)
					ui.Box(c).Height(18)
					a.autoList(c, items)
				})
			})
		} else {
			a.pageSplit(c, "au", func() {
				a.autoWeek(c, items)
				ui.Box(c).Height(18)
				a.autoList(c, items)
			}, func() { a.autoDrawer(c) })
		}
	})
	a.autoModals(c)
}

// currentProjectName 是当前会话所在项目名(新建自动化的默认项目)。
func (a *App) currentProjectName() string {
	if n := a.projectName(); n != "" {
		return n
	}
	return a.projLabel()
}

func (a *App) autoWeek(c *ui.Context, items []automation.Automation) {
	t := a.tk
	s := &a.au
	now := a.now()
	mon := automation.MondayOf(now).AddDate(0, 0, 7*s.weekOf)
	week := automation.Week(items, mon, now)
	end := mon.AddDate(0, 0, 6)
	title := "本周"
	switch {
	case s.weekOf < 0:
		title = itoa(-s.weekOf) + " 周前"
	case s.weekOf > 0:
		title = itoa(s.weekOf) + " 周后"
	}
	rng := itoa(int(mon.Month())) + " 月 " + itoa(mon.Day()) + " 日 – "
	if end.Month() != mon.Month() {
		rng += itoa(int(end.Month())) + " 月 "
	}
	rng += itoa(end.Day()) + " 日"
	ui.Row(c).Gap(8).AlignItems(ui.Center).Padding(0, 0, 8, 0).Children(func() {
		ui.Text(c, title).FontWeight(500)
		ui.Text(c, rng).TextColor(t.Tx3)
		ui.Spacer(c).Grow(1)
		kit.Ib(c, "chevL", kit.IbOpt{Small: true, Tip: "上一周"}).OnClick(func() { s.weekOf-- })
		kit.Btn(c, "今天", kit.BtnOpt{Small: true}).OnClick(func() { s.weekOf = 0 })
		kit.Ib(c, "chevR", kit.IbOpt{Small: true, Tip: "下一周"}).OnClick(func() { s.weekOf++ })
	})
	names := automation.Weekdays()
	if a.pageNarrowWide() {
		ui.Column(c).Gap(6).Children(func() {
			for i := 0; i < 7; i++ {
				d := mon.AddDate(0, 0, i)
				today := s.weekOf == 0 && i == (int(now.Weekday()+6)%7)
				col := ui.Column(c.Key("wk-v-"+itoa(i))).Gap(4).Padding(8, 10).Background(t.Raised).Radius(t.RMd()).Border(1, t.Bd1)
				if today {
					col.BorderColor(t.Ac).Border(1.5, t.Ac)
				}
				col.Children(func() {
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						ui.Text(c, names[i]).FontWeight(600)
						ui.Text(c, itoa(int(d.Month()))+"月"+itoa(d.Day())+"日").FontSize(theme.FsCap).TextColor(t.Tx3)
					})
					if len(week[i]) == 0 {
						ui.Text(c, "没有计划").FontSize(theme.FsCap).TextColor(t.Tx4)
					}
					for _, sl := range week[i] {
						a.autoSlot(c, sl, i)
					}
				})
			}
		})
		return
	}
	ui.Row(c).Gap(6).AlignItems(ui.Stretch).Children(func() {
		for i := 0; i < 7; i++ {
			d := mon.AddDate(0, 0, i)
			today := s.weekOf == 0 && i == (int(now.Weekday()+6)%7)
			col := ui.Column(c.Key("wk-"+itoa(i))).Grow(1).Basis(0).MinWidth(0).MinHeight(104).Padding(6, 6).Gap(4).Background(t.Raised).Radius(t.RMd()).Border(1, t.Bd1)
			if today {
				col.BorderColor(t.Ac).Border(1.5, t.Ac)
			}
			col.Children(func() {
				ui.Row(c).AlignItems(ui.Center).Padding(0, 2).Children(func() {
					ui.Text(c, names[i]).FontSize(theme.FsCap).TextColor(t.Tx3).Grow(1)
					n := ui.Text(c, itoa(d.Day())).FontWeight(600).FontFeatures("tnum")
					if today {
						n.TextColor(t.AcTx)
					}
				})
				if len(week[i]) == 0 {
					ui.Text(c, "没有计划").FontSize(theme.FsCap).TextColor(t.Tx4).Padding(2, 4)
				}
				for _, sl := range week[i] {
					a.autoSlot(c, sl, i)
				}
			})
		}
	})
}

func (a *App) autoSlot(c *ui.Context, sl automation.Slot, day int) {
	t := a.tk
	b := ui.ButtonBase(c).Key("slot-" + itoa(day) + "-" + itoa(sl.ID)).Height(theme.Px(22)).PaddingX(5).Gap(5).Radius(t.RXs()).AlignItems(ui.Center).Justify(ui.Start).
		Cursor(ui.CursorPointer).FontSize(theme.FsMicro + .5).Background(t.AcSoft).TextColor(t.AcTx)
	switch {
	case sl.Off:
		b.Background(t.Fill1).TextColor(t.Tx4)
	case sl.Err:
		b.Background(t.ErrSoft).TextColor(t.Err)
	}
	b.OnClick(func() { a.autoPick(sl.ID) })
	b.Children(func() {
		ui.Text(c, sl.Time).FontFeatures("tnum").Shrink(0)
		n := ui.Text(c, sl.Name).SingleLine().MinWidth(0).Shrink(1)
		if sl.Off {
			n.Strikethrough()
		}
	})
}

func (a *App) autoList(c *ui.Context, items []automation.Automation) {
	t := a.tk
	s := &a.au
	ui.Row(c).Gap(8).AlignItems(ui.Center).Padding(0, 0, 8, 0).Children(func() {
		ui.Text(c, "全部自动化").FontWeight(500)
		kit.Cnt(c, itoa(len(items)), false)
		ui.Spacer(c).Grow(1)
		if i := kit.Seg(c, []kit.SegItem{{Text: "全部"}, {Text: "运行中"}, {Text: "已暂停"}}, s.filter, kit.SegOpt{Key: "au-filter"}); i >= 0 {
			s.filter = i
		}
	})
	var shown []automation.Automation
	for _, it := range items {
		if s.filter == 0 || (s.filter == 1 && it.On) || (s.filter == 2 && !it.On) {
			shown = append(shown, it)
		}
	}
	if len(shown) == 0 {
		a.box(c).Padding(30, 12).Gap(6).AlignItems(ui.Center).Children(func() {
			kit.Ic(c, "timer", 20).TextColor(t.Tx4)
			ui.Text(c, map[int]string{1: "没有运行中的自动化", 2: "没有已暂停的自动化", 0: "还没有自动化"}[s.filter]).FontWeight(500)
			hint := "切到「全部」看看。"
			if s.filter == 0 {
				hint = "点右上角「新建自动化」,让 Claude 按计划自动做事。"
			}
			ui.Text(c, hint).FontSize(theme.FsCap).TextColor(t.Tx3)
		})
		return
	}
	cols := 2
	if a.pageNarrowWide() {
		cols = 1
	}
	for i := 0; i < len(shown); i += cols {
		ui.Row(c.Key("aurow-"+itoa(i))).Gap(10).AlignItems(ui.Stretch).Padding(0, 0, 10, 0).Children(func() {
			for j := i; j < i+cols; j++ {
				if j < len(shown) {
					a.autoCard(c, shown[j])
				} else if cols > 1 {
					ui.Box(c).Grow(1).Basis(0)
				}
			}
		})
	}
}

func (a *App) autoCard(c *ui.Context, it automation.Automation) {
	t := a.tk
	s := &a.au
	on := s.sel == it.ID
	card := ui.ButtonBase(c).Key("au-"+itoa(it.ID)).Grow(1).Basis(0).MinWidth(0).Padding(12, 14).Gap(10).Radius(t.RMd()).Background(t.Raised).Border(1, t.Bd1).Cursor(ui.CursorPointer).
		Justify(ui.Start).AlignItems(ui.Stretch)
	for _, sh := range t.Shadow1() {
		card.Shadow(sh.X, sh.Y, sh.Blur, sh.Spread, sh.Color)
	}
	if on {
		card.BorderColor(t.AcLine).Border(1, t.AcLine)
	} else if card.Hovered() {
		card.BorderColor(t.Bd3)
	}
	card.OnClick(func() { a.autoPick(it.ID) })
	last, st := it.Last(a.now())
	card.Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(10).Children(func() {
			ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
				iconBox(a, c, "timer")
				ui.Column(c).Grow(1).MinWidth(0).Children(func() {
					ui.Text(c, it.Name).FontWeight(500).SingleLine()
					ui.Text(c, it.Schedule()).FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine()
				})
				if kit.Tog(c, it.On, "au-tog-"+itoa(it.ID)) {
					a.auto.Toggle(it.ID)
				}
			})
			busy := a.autoBusy != nil && a.autoBusy[it.ID]
			ui.Row(c).Gap(8).AlignItems(ui.Center).FontSize(theme.FsCap).TextColor(t.Tx3).Children(func() {
				if busy {
					kit.Spin(c, t.Ac)
					ui.Text(c, "运行中…").TextColor(t.Ac)
				}
				kit.Chip(c, it.Project, kit.ChipOpt{Icon: "folder", Line: true})
				ui.Spacer(c).Grow(1)
				switch st {
				case 1:
					kit.Ic(c, "okc", kit.S12).TextColor(t.Ok)
				case 2:
					kit.Ic(c, "errc", kit.S12).TextColor(t.Err)
				}
				l := ui.Text(c, last).SingleLine().Shrink(1).MinWidth(0)
				if st == 2 {
					l.TextColor(t.Err)
				}
			})
		})
	})
}

func (a *App) autoDrawer(c *ui.Context) {
	t := a.tk
	s := &a.au
	it, ok := a.auto.Get(s.sel)
	if !ok {
		ui.Text(c, "没有选中的自动化").TextColor(t.Tx4)
		return
	}
	lbl := func(text string) { ui.Text(c, text).FontSize(theme.FsCap).FontWeight(500).TextColor(t.Tx3) }
	ui.Row(c).AlignItems(ui.Center).Children(func() {
		ui.Text(c, "编辑自动化").FontWeight(500).Grow(1)
		kit.Ib(c, "x", kit.IbOpt{Small: true, Tip: "关闭"})
	})
	field := func(key string, v *string, h float32) ui.Element {
		return ui.TextInput(c.Key(key), v).Grow(1).Height(h).FontSize(theme.FsBody)
	}
	ui.Column(c).Gap(4).Children(func() {
		lbl("名称")
		in := field("au-name", &s.name, 26)
		if s.focusName {
			in.AutoFocus()
			s.focusName = false
		}
	})
	ui.Column(c).Gap(4).Children(func() {
		lbl("提示词")
		ui.TextArea(c.Key("au-prompt"), &s.prompt).Height(92).FontSize(theme.FsBody)
	})
	ui.Column(c).Gap(4).Children(func() {
		lbl("计划")
		kinds := []string{"每天", "每周", "每月", "Cron"}
		cur := int(automation.KindOf(s.days))
		if i := kit.Seg(c, []kit.SegItem{{Text: kinds[0]}, {Text: kinds[1]}, {Text: kinds[2]}, {Text: kinds[3]}}, cur, kit.SegOpt{Fill: true, Key: "au-kind"}); i >= 0 && i != cur {
			s.days = []string{"每天", "周一", "每月 1 日", "0 9 * * 1"}[i]
			s.kindFoldAt = time.Now()
		}
		cronOpen := automation.KindOf(s.days) == automation.KCron
		cronBox := ui.Column(c)
		kit.FoldHeight(cronBox, cronOpen, 54, s.kindFoldAt, a.frameSoon)
		cronBox.Children(func() {
			ui.TextInput(c.Key("au-cron"), &s.days).FontSize(theme.FsBody).Placeholder("分 时 日 月 周,如 0 9 * * 1")
			ui.Text(c, "五段标准 cron;时间与 Asia/Shanghai 本地一致").FontSize(theme.FsCap).TextColor(t.Tx3)
		})
		timeBox := ui.Column(c)
		kit.FoldHeight(timeBox, !cronOpen, 34, s.kindFoldAt, a.frameSoon)
		timeBox.Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				a.autoDayPicker(c)
				field("au-time", &s.tm, 24).Width(70).Grow(0).Shrink(0)
				ui.Text(c, "时区 Asia/Shanghai").FontSize(theme.FsCap).TextColor(t.Tx3)
			})
		})
	})
	ui.Column(c).Gap(4).Children(func() {
		lbl("项目与权限")
		ui.Column(c).Gap(6).Children(func() {
			a.autoProjPick(c, &s.proj, "au-proj")
			a.autoPermPick(c, &s.perm)
		})
	})
	if nx, ok := it.NextRun(a.now()); ok {
		ui.Text(c, "下次运行 · "+automation.WhenNext(a.now(), nx)).FontSize(theme.FsCap).TextColor(t.Tx3)
	}
	ui.Column(c).Gap(4).Children(func() {
		lbl("最近运行")
		a.box(c).Padding(2, 4).Children(func() {
			if len(it.Runs) == 0 {
				ui.Text(c, "还没有运行记录").FontSize(theme.FsCap).TextColor(t.Tx4).Padding(8)
			}
			for i, r := range it.Runs {
				if i >= 5 {
					break
				}
				rk := r.Key
				if rk == "" {
					rk = fmt.Sprintf("auto:%d:%d", it.ID, r.At.Unix())
				}
				row := kit.RowEl(c, false).Key("run-" + itoa(it.ID) + "-" + itoa(i)).Height(theme.Px(28))
				row.OnClick(func() { a.openAutoRun(r, rk) })
				row.Children(func() {
					if r.OK {
						kit.Ic(c, "okc", kit.S12).TextColor(t.Ok)
					} else {
						kit.Ic(c, "errc", kit.S12).TextColor(t.Err)
					}
					ui.Text(c, automation.When(a.now(), r.At)).Grow(1)
					if sid := autoRunSessionID(r); sid != "" {
						ui.Text(c, "查看会话").FontSize(theme.FsCap).TextColor(t.Ac)
					} else if why := autoNoSessionReason(r); why != "" {
						ui.Text(c, "无法查看会话").FontSize(theme.FsCap).TextColor(t.Tx4).Tooltip(why)
					} else {
						ui.Text(c, map[bool]string{true: "成功", false: "失败"}[r.OK]).TextColor(t.Tx3)
					}
					kit.Ic(c, "chevR", kit.S12).TextColor(t.Tx4)
				})
			}
		})
	})
	if s.errMsg != "" {
		ui.Text(c, s.errMsg).FontSize(theme.FsCap).TextColor(t.Err)
	}
	ui.Spacer(c).Grow(1)
	busy := a.autoBusy != nil && a.autoBusy[it.ID]
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		if busy {
			ui.Box(c).Grow(1).MaxWidth(120).Children(func() { kit.IndetMeter(c) })
		}
		runBtn := kit.Btn(c, map[bool]string{true: "运行中…", false: "立即运行"}[busy], kit.BtnOpt{Variant: kit.BtnSoft, Icon: "play", Loading: busy})
		if busy {
			runBtn.OnClick(func() { s.runView = liveKey(it.ID) })
		} else {
			runBtn.OnClick(func() {
				a.runAutomationNow(it)
				log.Info("立即运行自动化", "id", it.ID, "name", it.Name)
			})
		}
		if busy {
			kit.Btn(c, "取消", kit.BtnOpt{Variant: kit.BtnGhost, Small: true}).OnClick(func() { a.cancelAutoRun(it.ID) })
		}
		kit.Btn(c, "删除", kit.BtnOpt{Variant: kit.BtnDangerGhost, Icon: "trash"}).OnClick(func() {
			// 删除不可恢复(运行记录一并丢掉),先确认;已产生的会话在 Claude 那边,不受影响
			a.confirmDlg("删除「"+it.Name+"」?", "会同时删除它的运行记录,不可恢复。它已产生的会话不会被删除。", "删除", true, func() {
				a.auto.Delete(it.ID)
				s.draftID = 0
				a.toast = "已删除「" + it.Name + "」"
				log.Info("删除自动化", "id", it.ID, "name", it.Name)
			})
		})
		ui.Spacer(c).Grow(1)
		save := kit.Btn(c, "保存", kit.BtnOpt{Variant: kit.BtnPri})
		save.OnClick(func() { a.autoSave(it) })
		if !s.savedPopAt.IsZero() {
			ok := kit.Ic(c, "okc", kit.S12).TextColor(t.Ok)
			kit.OkPop(ok, s.savedPopAt, a.frameSoon)
		}
	})
}

// autoSave 把草稿写回存储(校验名称与时间)。
func (a *App) autoSave(it automation.Automation) {
	s := &a.au
	it.Name, it.Prompt, it.Time, it.Days, it.Perm, it.Project = s.name, s.prompt, s.tm, s.days, s.perm, s.proj
	if err := a.auto.Update(it); err != nil {
		s.errMsg = err.Error()
		return
	}
	s.errMsg = ""
	a.toast = "已保存「" + it.Name + "」"
	a.au.savedPopAt = time.Now()
	log.Info("保存自动化", "id", it.ID)
}

// autoDayPicker 是计划里的「哪天」芯片:点一下循环切换可选值(每天 ↔ 工作日 / 周一…周日 / 每月 N 日)。
func (a *App) autoDayPicker(c *ui.Context) {
	s := &a.au
	k := automation.KindOf(s.days)
	label := s.days
	chip := kit.Chip(c, label, kit.ChipOpt{Line: true, Car: k != automation.KCron, Tip: "点击切换"})
	chip.OnClick(func() {
		switch k {
		case automation.KDaily:
			s.days = map[string]string{"每天": "工作日", "工作日": "每天"}[s.days]
		case automation.KWeekly:
			names := automation.Weekdays()
			for i, n := range names {
				if n == s.days {
					s.days = names[(i+1)%7]
				}
			}
		case automation.KMonthly:
			var n int
			for _, ch := range s.days {
				if ch >= '0' && ch <= '9' {
					n = n*10 + int(ch-'0')
				}
			}
			s.days = "每月 " + itoa(n%28+1) + " 日"
		}
	})
}
