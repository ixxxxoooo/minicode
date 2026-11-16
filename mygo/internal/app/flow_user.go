package app

import (
	"regexp"
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

// userLinesCap 是 estimatedUserLines 的上限:折叠 / 展开只关心「是否超过 6 / 24 行」,
// 数到这里就停,避免几万字的粘贴日志每帧从头数到尾。
const userLinesCap = 100

// estimatedUserLines 估算文字排成多少行(按 col 个半角宽度折行),最多数到 userLinesCap。
func estimatedUserLines(text string, col int) int {
	if col < 20 {
		col = 44
	}
	lines := 0
	for rest := text; ; {
		para := rest
		nl := strings.IndexByte(rest, '\n')
		if nl >= 0 {
			para = rest[:nl]
		}
		w := 0
		for _, r := range para {
			if r > 127 || unicode.Is(unicode.Han, r) {
				w += 2
			} else {
				w++
			}
			if w >= col {
				lines++
				w = 0
			}
		}
		if w > 0 || para == "" {
			lines++
		}
		if lines >= userLinesCap {
			return userLinesCap
		}
		if nl < 0 {
			break
		}
		rest = rest[nl+1:]
	}
	if lines == 0 {
		lines = 1
	}
	return lines
}

// userPreviewRunes 是折叠态送进排版的最大字符数:最宽的内容列(860px)一行约 110~140 个半角字符,
// 1000 个字符足够排满折叠的 6 行,视觉与整篇排版后再裁剪一致。
const userPreviewRunes = 1000

// userFoldPreview 返回折叠态用户卡需要的文字前缀:前 userFoldMax 个物理行、且不超过 userPreviewRunes 个字符。
// 折叠只是把正文裁成 6 行高,但整段文字仍会被排版;一条 3 万字的粘贴日志在每个过渡帧(宽度逐帧变化)
// 都要整篇重新断行,实测每帧约 41 ms,连续切换形态时叠成 1 秒以上的停滞。
func userFoldPreview(txt string) string {
	if len(txt) <= userPreviewRunes { // 字节数不超过上限,字符数一定不超过
		return txt
	}
	n, nl := 0, 0
	for i, r := range txt {
		if n >= userPreviewRunes {
			return txt[:i]
		}
		if r == '\n' {
			if nl++; nl >= userFoldMax {
				return txt[:i]
			}
		}
		n++
	}
	return txt
}

func (a *App) userExpanded(id string) bool {
	return a.outFold != nil && a.outFold["user:"+id]
}

func (a *App) beginUserEdit(u *core.User) {
	a.userEd = userEdit{id: u.ID, draft: userDisplayText(u.Text)}
}

func (a *App) cancelUserEdit() { a.userEd = userEdit{} }

func (a *App) userMsg(c *ui.Context, u *core.User, roundBox ui.Rect) {
	t := a.tk
	style := a.prefs.Int(pFlowStyle, 0)
	sticky := a.prefs.Bool(pStickyUser, true)
	lines := estimatedUserLines(userDisplayText(u.Text), 44)
	clamp := lines > userFoldMax
	exp := a.userExpanded(u.ID)
	editing := a.userEd.id == u.ID
	if editing {
		a.userEditCard(c, u)
		return
	}
	card := ui.ButtonBase(c).Key("user-"+u.ID).Background(t.Raised).Radius(t.RMd()).Padding(7, 11).AlignItems(ui.Stretch).Column()
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
	bg := t.Raised.Over(t.BG) // 卡片底色,悬停动作条要用同色盖住下面的文字
	switch style {
	case 1:
		bg = t.AcSoft.Over(t.BG)
		card.AlignSelf(ui.End).MaxWidthPercent(86).Background(t.AcSoft.Over(t.BG)).Radius(t.RLg(), t.RLg(), t.RXs(), t.RLg())
	case 2:
		card.AlignSelf(ui.Stretch).MaxWidthPercent(100).Background(t.Raised.Over(t.BG))
	default:
		card.Background(t.Raised.Over(t.BG))
		if t.Scheme.ID == "b" {
			bg = t.AcSoft.Over(t.BG)
			card.AlignSelf(ui.End).MaxWidthPercent(86).Background(t.AcSoft.Over(t.BG)).Radius(t.RLg(), t.RLg(), t.RXs(), t.RLg())
		}
		if t.Scheme.ID == "c" {
			bg = t.Fill1.Over(t.BG)
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
	card.OnClick(func() { a.userCardClick(u, stuck) }).Cursor(ui.CursorPointer)
	card.Children(func() {
		if clamp && exp && lines > userFoldExp {
			fs := a.FlowFontSize()
			ui.Scroll(c).MaxHeight(float32(userFoldExp) * fs * 1.55).Key("user-scroll-" + u.ID).Children(func() {
				a.userMsgBody(c, u, false)
			})
		} else {
			body := ui.Column(c)
			if clamp && !exp {
				body = body.MaxHeight(float32(userFoldMax)*a.FlowFontSize()*1.55 + 8).Clip()
			}
			body.Children(func() { a.userMsgBody(c, u, clamp && !exp) })
		}
		a.userMsgImages(c, u)
		if hov {
			a.userMsgHoverBar(c, u, bg)
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
	a.userFloat(c, u, card) // 吸顶时点击:向下悬浮展开完整正文
}

// userImgMark 是 Claude 在用户文字里为图片插入的占位标记(如「[Image #16]」),
// 图片本身已经以缩略图显示在文字下方,标记只会让文字显得杂乱。
var userImgMark = regexp.MustCompile(`\[Image #\d+\]\s*`)

// userDisplayText 返回会话流里展示的用户文字:去掉图片占位标记并修剪首尾空白。
// 只用于显示,原文(u.Text)保持不变,编辑 / 回退 / 复制仍用原文。
func userDisplayText(s string) string {
	if !strings.Contains(s, "[Image #") {
		return s
	}
	return strings.TrimSpace(userImgMark.ReplaceAllString(s, ""))
}

// userMsgBody 画用户消息正文。folded = true 是折叠态:只排版前缀(见 userFoldPreview),外面的卡片负责裁成 6 行高。
func (a *App) userMsgBody(c *ui.Context, u *core.User, folded bool) {
	t := a.tk
	txt := userDisplayText(u.Text)
	if folded {
		txt = userFoldPreview(txt)
	}
	if txt == "" && len(u.Images) > 0 {
		return // 只有图片:不留空文字行
	}
	if !userTok.MatchString(txt) {
		ui.RichText(c, ui.Span{Text: txt}).FontSize(a.FlowFontSize()).LineHeight(1.55).TextColor(t.Tx1)
		return
	}
	ui.Row(c).Wrap().Gap(4).AlignItems(ui.Center).Children(func() {
		last := 0
		for _, m := range userTok.FindAllStringSubmatchIndex(txt, -1) {
			if m[0] > last {
				ui.Text(c, txt[last:m[0]]).FontSize(a.FlowFontSize()).LineHeight(1.55).TextColor(t.Tx1)
			}
			if m[2] >= 0 {
				code := txt[m[2]:m[3]]
				fg, bg := a.inlineCodeColors()
				ui.Text(c, code).Font("monospace").FontSize(a.FlowMonoSize()).LineHeight(1.55).TextColor(fg).
					Background(bg).Radius(t.RXs()).PaddingX(5)
			} else {
				a.userRefChip(c, txt[m[4]:m[5]])
			}
			last = m[1]
		}
		if last < len(txt) {
			ui.Text(c, txt[last:]).FontSize(a.FlowFontSize()).LineHeight(1.55).TextColor(t.Tx1)
		}
	})
}
