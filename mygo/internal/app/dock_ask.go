package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 提问卡(设计稿 flow-queue「问答卡」):多个问题时顶部有步骤条,可直接跳到任意一题,
// 底部「上一题 / 下一题(最后一题是提交)」,← → 也能切换;回答完在会话流里留一张回答卡片。

// answered 第 i 题是否已有可提交的回答(选了选项且「其它」已写说明 / 自由输入非空 / 多选允许空)。
func (s *qaState) answered(q *core.QuestionRequest, i int) bool {
	if i < len(s.skip) && s.skip[i] {
		return true
	}
	qu := q.Questions[i]
	if len(qu.Options) == 0 {
		return strings.TrimSpace(s.freet[i]) != ""
	}
	if len(s.sel[i]) == 0 {
		return qu.MultiSelect // 多选可以一个都不选
	}
	if s.wantsFreeAt(qu, i) && strings.TrimSpace(s.freet[i]) == "" {
		return false
	}
	return true
}

// hasAnswer 第 i 题是否「真的答过」(步骤条上显示对勾;多选空选不算)。
func (s *qaState) hasAnswer(q *core.QuestionRequest, i int) bool {
	if i < len(s.skip) && s.skip[i] {
		return true
	}
	qu := q.Questions[i]
	if len(qu.Options) == 0 {
		return strings.TrimSpace(s.freet[i]) != ""
	}
	return len(s.sel[i]) > 0 && s.answered(q, i)
}

// firstOpen 第一道还不能提交的题;全部就绪返回 -1。
func (s *qaState) firstOpen(q *core.QuestionRequest) int {
	for i := range q.Questions {
		if !s.answered(q, i) {
			return i
		}
	}
	return -1
}

func (a *App) askCard(c *ui.Context, q *core.QuestionRequest) {
	t := a.tk
	n := len(q.Questions)
	if n == 0 {
		return
	}
	s := a.qaOf(q)
	if s.cur >= n {
		s.cur = n - 1
	}
	qu := q.Questions[s.cur]
	last := s.cur == n-1

	submit := func() {
		if i := s.firstOpen(q); i >= 0 { // 前面有题没答:跳回去并提示
			s.cur = i
			a.toast = "第 " + itoa(i+1) + " 题还没有回答"
			return
		}
		log.Info("提交回答", "题数", n)
		a.send(core.CmdAnswerQuestion{ID: q.ID, Answers: s.answers(q)})
		delete(a.qa, q.ID)
	}
	goTo := func(i int) {
		if i >= 0 && i < n {
			s.cur = i
		}
	}
	advance := func() {
		if !s.answered(q, s.cur) {
			if len(qu.Options) > 0 && !s.wantsFreeInput(qu) {
				a.toast = "请先选择一个选项,或点「跳过」"
			} else {
				a.toast = "请先写下你的回答,或点「跳过」"
			}
			return
		}
		if last {
			submit()
		} else {
			s.cur++
		}
	}
	skip := func() {
		s.sel[s.cur], s.freet[s.cur], s.skip[s.cur] = nil, "", true
		if last {
			submit()
		} else {
			s.cur++
		}
	}
	active := !a.paletteOn && a.screen == ScreenWork
	if active && c.Shortcut(0, ui.KeyEnter) {
		advance()
	} else if active && c.Shortcut(0, ui.KeyEscape) {
		log.Info("跳过提问", "题", s.cur+1, "共", n)
		skip()
	}
	// ← → 切题:只在没有文字输入框抢方向键时生效
	if active && n > 1 && !s.wantsFreeInput(qu) {
		if c.Shortcut(0, ui.KeyLeft) {
			goTo(s.cur - 1)
		} else if c.Shortcut(0, ui.KeyRight) {
			goTo(s.cur + 1)
		}
	}
	a.cardShell(c, "dock-ask").Children(func() {
		ui.Row(c).Gap(7).AlignItems(ui.Center).Children(func() {
			kit.Ic(c, "info", kit.S12).TextColor(t.AcTx)
			title := qu.Question
			if n > 1 {
				title = "需要你确认 " + itoa(n) + " 个问题"
			}
			ui.Text(c, title).FontWeight(500).Shrink(1)
			ui.Spacer(c).Grow(1)
			a.dockBadge(c)
			if n > 1 {
				ui.Text(c, itoa(s.cur+1)+" / "+itoa(n)).FontSize(theme.FsCap).TextColor(t.Tx3).FontFeatures("tnum")
			}
		})
		if n > 1 { // 步骤条:点任一步直接跳过去
			ui.Row(c).Gap(4).Wrap().Children(func() {
				for i, x := range q.Questions {
					i, x := i, x
					cur := i == s.cur
					chip := ui.ButtonBase(c).Key("qa-step" + itoa(i)).Height(theme.Px(22)).PaddingX(8).Gap(5).Radius(11).AlignItems(ui.Center).
						Cursor(ui.CursorPointer).Transition(kit.Fade).MaxWidth(170)
					switch {
					case cur:
						chip.Background(t.AcSoft).Border(1.5, t.Ac)
					case chip.Hovered():
						chip.Background(t.Fill1).Border(1, t.Bd2)
					default:
						chip.Border(1, t.Bd2)
					}
					chip.OnClick(func() { goTo(i) })
					chip.Children(func() {
						if s.hasAnswer(q, i) {
							kit.Ic(c, "check", kit.S12).TextColor(t.Ok)
						} else {
							ui.Text(c, itoa(i+1)).FontSize(theme.Px(11.5)).TextColor(t.Tx3).FontFeatures("tnum")
						}
						lb := strings.TrimSpace(x.Header)
						if lb == "" {
							lb = x.Question
						}
						col := t.Tx2
						if cur {
							col = t.Tx1
						}
						ui.Text(c, lb).FontSize(theme.Px(11.5)).TextColor(col).SingleLine().Shrink(1)
					})
				}
			})
			ui.Text(c, qu.Question).FontWeight(500).LineHeight(1.45)
		}
		if qu.MultiSelect && len(qu.Options) > 0 {
			ui.Text(c, "可多选").FontSize(theme.FsCap).TextColor(t.Tx3)
		}
		if len(qu.Options) == 0 {
			ui.TextInputBase(c.Key("ask-free"+itoa(s.cur)), &s.freet[s.cur]).Height(theme.Px(28)).PaddingX(9).Radius(t.RSm()).Border(1, t.Bd2).Placeholder("输入你的回答…").PlaceholderColor(t.Tx4)
		}
		ui.Column(c).Gap(4).Children(func() {
			for i, o := range qu.Options {
				on := hasInt(s.sel[s.cur], i)
				i := i
				row := ui.ButtonBase(c).Key("opt" + itoa(s.cur) + "-" + itoa(i)).Justify(ui.Start).Height(theme.Px(28)).PaddingX(9).Gap(8).Radius(t.RSm()).AlignItems(ui.Center).Cursor(ui.CursorPointer).
					Transition(kit.Fade)
				switch {
				case on:
					row.Background(t.AcSoft).Border(1.5, t.Ac)
				case row.Hovered():
					row.Background(t.Fill1).Border(1, t.Bd2)
				default:
					row.Border(1, t.Bd2)
				}
				row.OnClick(func() { s.skip[s.cur] = false; s.toggleOption(qu, i) })
				row.Children(func() {
					st := 0
					if on {
						st = 2
					}
					kit.Chk(c, st)
					ui.Text(c, o.Label).FontWeight(500).SingleLine().Shrink(0)
					if o.Description != "" {
						ui.Text(c, o.Description).FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine().Shrink(1)
					}
				})
			}
		})
		if s.wantsFreeInput(qu) && len(qu.Options) > 0 {
			ui.TextInputBase(c.Key("ask-other"+itoa(s.cur)), &s.freet[s.cur]).Height(theme.Px(28)).PaddingX(9).Radius(t.RSm()).Border(1, t.Bd2).Placeholder("其它说明…").PlaceholderColor(t.Tx4)
		}
		ui.Row(c).Gap(6).AlignItems(ui.Center).Wrap().Children(func() {
			if n > 1 {
				kit.Btn(c, "上一题", kit.BtnOpt{Variant: kit.BtnGhost, Small: true, Icon: "arrowL", Disabled: s.cur == 0}).OnClick(func() { goTo(s.cur - 1) })
			}
			ui.Spacer(c).Grow(1)
			if a.mode == ModeAgent && a.chatTier() != tierNarrow { // 窄档不放键位提示
				ui.Row(c).Gap(4).AlignItems(ui.Center).TextColor(t.Tx3).FontSize(theme.FsCap).Children(func() {
					if n > 1 && !s.wantsFreeInput(qu) {
						kit.Kbd(c, "←")
						kit.Kbd(c, "→")
						ui.Text(c, "切换")
					}
					kit.Kbd(c, "⏎")
					if last {
						ui.Text(c, "提交")
					} else {
						ui.Text(c, "下一题")
					}
					kit.Kbd(c, "esc")
					ui.Text(c, "跳过")
				})
			}
			kit.Btn(c, "跳过", kit.BtnOpt{Variant: kit.BtnGhost, Small: true}).OnClick(skip)
			label := "下一题"
			if last {
				label = "提交"
			}
			kit.Btn(c, label, kit.BtnOpt{Variant: kit.BtnPri, Small: true}).OnClick(advance)
		})
	})
}

// askEcho 是已回答的提问在会话流里的回答卡片(设计稿 .qa-done):每题一行问题,回答是强调淡底的小胶囊;
// 多选 = 多个胶囊,「其它」的自由输入 = 方角胶囊,跳过 = 灰色「已跳过」。
func (a *App) askEcho(c *ui.Context, q *core.QuestionRequest) {
	t := a.tk
	skipped := 0
	for i := range q.Questions {
		if i >= len(q.Answers) || len(q.Answers[i]) == 0 {
			skipped++
		}
	}
	lbl := "已回答 " + itoa(len(q.Questions)) + " 个问题"
	box := ui.Column(c).Key("qa-done-"+q.ID).Background(t.Raised).Radius(t.RMd()).Padding(8, 11).Gap(7).Label(lbl)
	for _, s := range t.Shadow1() {
		box.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
	}
	kit.Float(box)
	kit.CtxMenu(box, func(m *ui.Menu) { a.askEchoMenu(m, q) })
	box.Children(func() {
		ui.Row(c).Gap(7).AlignItems(ui.Center).Children(func() {
			kit.Ic(c, "check", kit.S12).TextColor(t.Ok)
			ui.Text(c, "已回答 "+itoa(len(q.Questions))+" 个问题").FontWeight(600).Shrink(1)
			ui.Spacer(c).Grow(1)
			if skipped > 0 && len(q.Questions) > 1 {
				kit.Tag(c, itoa(skipped)+" 个已跳过")
			}
		})
		for i, qu := range q.Questions {
			ans := []string(nil)
			if i < len(q.Answers) {
				ans = q.Answers[i]
			}
			ui.Column(c).Gap(3).Children(func() {
				ui.Row(c).Gap(5).Children(func() {
					if len(q.Questions) > 1 {
						ui.Text(c, itoa(i+1)).TextColor(t.Tx4).FontFeatures("tnum").Shrink(0)
					}
					ui.Text(c, qu.Question).TextColor(t.Tx2).Shrink(1)
				})
				ui.Row(c).Gap(4).Wrap().Padding(0, 0, 0, 16).Children(func() {
					if len(ans) == 0 {
						ui.Row(c).Background(t.Fill1).Radius(10).PaddingX(8).MinHeight(20).AlignItems(ui.Center).Children(func() {
							ui.Text(c, "已跳过").TextColor(t.Tx3)
						})
						return
					}
					for _, x := range ans {
						rad := float32(10)
						if isFreeAnswer(qu, x) {
							rad = t.RSm()
						}
						ui.Row(c).Background(t.AcSoft).Radius(rad).PaddingX(8).MinHeight(20).AlignItems(ui.Center).Children(func() {
							ui.Text(c, x).TextColor(t.Tx1)
						})
					}
				})
			})
		}
	})
}

// isFreeAnswer 回答不在候选项里 = 用户自己写的(自由输入 / 「其它」说明)。
func isFreeAnswer(qu core.Question, ans string) bool {
	for _, o := range qu.Options {
		if o.Label == ans {
			return false
		}
	}
	return true
}
