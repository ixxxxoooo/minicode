package app

import (
	"strings"
	"unicode"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

// 用户消息卡:长消息折叠、吸顶、就地编辑;会话样式(flowStyle)。

const (
	pFlowStyle  = "appearance.flowStyle"
	pStickyUser = "chat.stickyUser"
	userFoldMax = 6
	userFoldExp = 24
)

type userEdit struct {
	id, draft string
}

// flowRounds 按「回合」渲染会话流(一轮 = 用户消息 + 后续条目直到下一条用户消息)。
func (a *App) flowRounds(c *ui.Context) {
	items := a.visibleItems()
	i := 0
	for i < len(items) {
		u, ok := items[i].(*core.User)
		if !ok {
			a.flowTail(c, items, i)
			i++
			continue
		}
		j := i + 1
		for j < len(items) {
			if _, ok := items[j].(*core.User); ok {
				break
			}
			j++
		}
		a.flowRound(c, u, items[i+1:j])
		i = j
	}
}

func (a *App) flowTail(c *ui.Context, items []core.Item, i int) {
	if isStepItem(items[i]) {
		j := i
		for j < len(items) && isStepItem(items[j]) {
			j++
		}
		a.stepGroup(c, items[i:j])
		return
	}
	switch v := items[i].(type) {
	case *core.Assistant:
		a.assistant(c, v)
	case *core.Notice:
		a.notice(c, v)
	case *core.TaskNote:
		a.taskNoteLine(c, v)
	case *core.PlanApproval:
		a.planCard(c, v)
	case *core.QuestionRequest:
		if v.Answered {
			a.askEcho(c, v)
		}
	}
}

func turnEndRound(rest []core.Item, i int, uid string) (string, bool) {
	if i+1 < len(rest) {
		return "", false
	}
	return uid, true
}

func (a *App) flowRound(c *ui.Context, u *core.User, rest []core.Item) {
	foldTurn := a.shouldFoldTurn(u.ID, rest, a.roundComplete(u.ID))
	var allSteps []core.Item
	if foldTurn {
		allSteps = turnStepItems(rest)
	}
	foldedSteps := false
	round := ui.Column(c).Key("round-" + u.ID).Gap(8).AlignItems(ui.Stretch)
	// 上一帧回合的顶边:提问卡的顶边比它低,说明正被吸顶(Bounds 是上一帧的矩形,第一帧为空)。
	roundTop := round.Bounds()
	round.Children(func() {
		a.userMsg(c, u, roundTop)
		for i := 0; i < len(rest); {
			last := i
			if isStepItem(rest[i]) {
				if foldTurn {
					if !foldedSteps {
						a.turnFoldGroup(c, u.ID, allSteps)
						foldedSteps = true
					}
					j := i
					for j < len(rest) && isStepItem(rest[j]) {
						j++
					}
					last = j - 1
				} else {
					j := i
					for j < len(rest) && isStepItem(rest[j]) {
						j++
					}
					a.stepGroup(c, rest[i:j])
					last = j - 1
				}
			} else {
				if !(foldTurn && foldedSteps && turnFoldedItem(rest[i])) {
					a.flowTail(c, rest, i)
				}
				last = i
			}
			if uid, ok := turnEndRound(rest, last, u.ID); ok {
				// 改动卡汇总的是整个会话的改动,只放在最后一轮,免得每一轮都显示同一份累计数字。
				if last == len(rest)-1 && a.st.Phase != core.PhaseRunning && rest[last] == a.st.Items[len(a.st.Items)-1] {
					a.changesCard(c)
				}
				a.turnFooterActions(c, uid)
			}
			i = last + 1
		}
	})
}

func estimatedUserLines(text string, col int) int {
	if col < 20 {
		col = 44
	}
	lines := 0
	for _, para := range strings.Split(text, "\n") {
		w := 0
		for _, r := range para {
			w += map[bool]int{true: 2, false: 1}[r > 127 || unicode.Is(unicode.Han, r)]
			if w >= col {
				lines++
				w = 0
			}
		}
		if w > 0 || para == "" {
			lines++
		}
	}
	if lines == 0 {
		lines = 1
	}
	return lines
}

func (a *App) userExpanded(id string) bool {
	return a.outFold != nil && a.outFold["user:"+id]
}

func (a *App) beginUserEdit(u *core.User) {
	a.userEd = userEdit{id: u.ID, draft: u.Text}
}

func (a *App) cancelUserEdit() { a.userEd = userEdit{} }

func (a *App) userMsg(c *ui.Context, u *core.User, roundBox ui.Rect) {
	t := a.tk
	style := a.prefs.Int(pFlowStyle, 0)
	sticky := a.prefs.Bool(pStickyUser, true)
	lines := estimatedUserLines(u.Text, 44)
	clamp := lines > userFoldMax
	exp := a.userExpanded(u.ID)
	editing := a.userEd.id == u.ID
	if editing {
		a.userEditCard(c, u)
		return
	}
	card := ui.ButtonBase(c).Key("user-"+u.ID).Background(t.Raised).Radius(t.RMd()).Padding(7, 11).AlignItems(ui.Stretch).Vertical()
	// 正被吸顶:上一帧提问卡的顶边比所属回合的顶边低(设计稿 .msg-u.stuck)。
	stuck := false
	if sticky {
		card = card.Sticky().Top(0).ZIndex(2)
		if b := card.Bounds(); b.H > 0 && roundBox.H > 0 && b.Y > roundBox.Y+0.5 {
			stuck = true
		}
	}
	// 吸顶的卡必须不透明,否则下面滚过来的内容会透出来叠成一团:
	// Fill1 之类的半透明填充要先叠在底色 BG 上。
	switch style {
	case 1:
		card.AlignSelf(ui.End).MaxWidthPercent(86).Background(t.AcSoft.Over(t.BG)).Radius(t.RLg(), t.RLg(), t.RXs(), t.RLg())
	case 2:
		card.AlignSelf(ui.Stretch).MaxWidthPercent(100).Background(t.Raised.Over(t.BG))
	default:
		card.Background(t.Raised.Over(t.BG))
		if t.Scheme.ID == "b" {
			card.AlignSelf(ui.End).MaxWidthPercent(86).Background(t.AcSoft.Over(t.BG)).Radius(t.RLg(), t.RLg(), t.RXs(), t.RLg())
		}
		if t.Scheme.ID == "c" {
			card.Background(t.Fill1.Over(t.BG)).Radius(0, t.RSm(), t.RSm(), 0)
		}
	}
	if stuck {
		// 固定在顶部时:卡片正下方一小道短阴影 + 一圈细边,与下面滚动的内容分开。
		// (不用大偏移的 Shadow2,也不再叠下沿渐隐,那两层叠起来会在整行下面拖出长暗带。)
		for _, s := range t.ShadowStuck() {
			card.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
		}
		card.Shadow(0, 0, 0, 1, t.Bd1)
	} else if style == 0 && t.Scheme.ID != "b" {
		for _, s := range t.Shadow1() {
			card.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
		}
	}
	hov := card.Hovered()
	card.OnClick(func() { a.beginUserEdit(u) }).Cursor(ui.CursorPointer)
	card.Children(func() {
		if clamp && exp && lines > userFoldExp {
			fs := a.FlowFontSize()
			ui.Scroll(c).MaxHeight(float32(userFoldExp) * fs * 1.55).Key("user-scroll-" + u.ID).Children(func() {
				a.userMsgBody(c, u)
			})
		} else {
			body := ui.Column(c)
			if clamp && !exp {
				body = body.MaxHeight(float32(userFoldMax)*a.FlowFontSize()*1.55 + 8).Clip()
			}
			body.Children(func() { a.userMsgBody(c, u) })
		}
		a.userMsgImages(c, u)
		if hov {
			a.userMsgHoverBar(c, u)
		}
		if clamp {
			ui.Row(c).Gap(6).Margin(4, 0, 0, 0).Children(func() {
				lbl := "展开"
				if exp {
					lbl = "收起"
				}
				kit.Btn(c, lbl, kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(func() {
					if a.outFold == nil {
						a.outFold = map[string]bool{}
					}
					a.outFold["user:"+u.ID] = !exp
				})
			})
		}
	})
}

func (a *App) userMsgBody(c *ui.Context, u *core.User) {
	t := a.tk
	var spans []ui.Span
	txt := u.Text
	last := 0
	for _, m := range userTok.FindAllStringSubmatchIndex(txt, -1) {
		if m[0] > last {
			spans = append(spans, ui.Span{Text: txt[last:m[0]]})
		}
		if m[2] >= 0 {
			spans = append(spans, ui.Span{Text: txt[m[2]:m[3]], Font: "monospace", Size: a.FlowMonoSize(), Background: t.Fill2, Radius: t.RXs(), PaddingX: 3})
		} else {
			spans = append(spans, ui.Span{Text: " " + baseName(txt[m[4]:m[5]]) + " ", Size: a.FlowCapFont(), Color: t.Info, Background: t.InfoSoft, Radius: t.RXs(), PaddingX: 4})
		}
		last = m[1]
	}
	if last < len(txt) {
		spans = append(spans, ui.Span{Text: txt[last:]})
	}
	ui.RichText(c, spans...).FontSize(a.FlowFontSize()).LineHeight(1.55).TextColor(t.Tx1)
}
