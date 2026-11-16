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

func (a *App) userMsg(c *ui.Context, u *core.User, rowIdx int) {
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
	// 正被吸顶(设计稿 .msg-u.stuck):用户消息是列表的分节标题,库把它钉在列表顶部。
	// 判断(上一帧的数据):标题行已滚出视口(首个可见行在它之后),或它贴着列表顶边且内容确实滚过。
	stuck := false
	if sticky {
		stuck = a.userStuck(card.Bounds(), rowIdx)
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
	// 与背景分开:默认轻阴影+细描边;吸顶时加深。
	if stuck {
		for _, s := range t.ShadowStuck() {
			card.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
		}
		card.Shadow(0, 0, 0, 1, t.Bd1)
	} else {
		for _, s := range t.Shadow1() {
			card.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
		}
		card.Shadow(0, 0, 0, 1, t.Bd1.Alpha(.7))
	}
	hov := card.Hovered()
	card.OnClick(func() { a.userCardClick(u, stuck) }).Cursor(ui.CursorPointer)
	card.Children(func() {
		if clamp && exp && lines > userFoldExp {
			fs := a.FlowFontSize()
			kit.VScroll(c).MaxHeight(float32(userFoldExp) * fs * a.FlowReadLH()).Key("user-scroll-" + u.ID).Children(func() {
				a.userMsgBody(c, u, false)
			})
		} else {
			body := ui.Column(c)
			if clamp && !exp {
				body = body.MaxHeight(float32(userFoldMax)*a.FlowFontSize()*a.FlowReadLH() + 8).Clip()
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
		ui.RichText(c, ui.Span{Text: txt}).FontSize(a.FlowFontSize()).LineHeight(a.FlowReadLH()).TextColor(t.Tx1)
		return
	}
	ui.Row(c).Wrap().Gap(4).AlignItems(ui.Center).Children(func() {
		last := 0
		for _, m := range userTok.FindAllStringSubmatchIndex(txt, -1) {
			if m[0] > last {
				ui.Text(c, txt[last:m[0]]).FontSize(a.FlowFontSize()).LineHeight(a.FlowReadLH()).TextColor(t.Tx1)
			}
			if m[2] >= 0 {
				code := txt[m[2]:m[3]]
				fg, bg := a.inlineCodeColors()
				ui.Text(c, code).Font("monospace").FontSize(a.FlowMonoSize()).LineHeight(a.FlowReadLH()).TextColor(fg).
					Background(bg).Radius(t.RXs()).PaddingX(5)
			} else {
				a.userRefChip(c, txt[m[4]:m[5]])
			}
			last = m[1]
		}
		if last < len(txt) {
			ui.Text(c, txt[last:]).FontSize(a.FlowFontSize()).LineHeight(a.FlowReadLH()).TextColor(t.Tx1)
		}
	})
}
