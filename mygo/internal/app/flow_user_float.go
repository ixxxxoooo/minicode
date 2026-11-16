package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 吸顶用户消息的「向下悬浮展开」(设计稿 .msg-u.stuck 点击):
// 吸顶时卡片只露出折叠的几行,点一下就在它正下方浮出完整正文——浮层不占位、不挤动下面的会话,
// 正文可以鼠标拖选 / 双击选词 / ⌘C 复制;点浮层外面或按 Esc 收起。「编辑」入口放在浮层底部。
// 没有吸顶时点卡片仍然是就地编辑。

func userFloatKey(id string) string { return "ufl:" + id }

// userFloatOpen 这条消息的悬浮层是否开着(或正在退场)。
func (a *App) userFloatOpen(id string) bool {
	k := userFloatKey(id)
	return a.popup == k || a.popClosing == k
}

// userCardClick 是用户消息卡的点击:吸顶(或浮层已开)时开 / 关悬浮层,否则就地编辑。
func (a *App) userCardClick(u *core.User, stuck bool) {
	k := userFloatKey(u.ID)
	if stuck || a.popup == k {
		a.togglePop(k)
		a.frameSoon()
		log.Info("吸顶用户消息悬浮展开", "user", u.ID, "open", a.popup == k)
		return
	}
	a.beginUserEdit(u)
}

// userFloat 在 anchor(吸顶的用户消息卡)正下方画悬浮层;没开时什么也不画。
func (a *App) userFloat(c *ui.Context, u *core.User, anchor ui.Element) {
	if !a.userFloatOpen(u.ID) {
		return
	}
	t := a.tk
	w := float32(360)
	if b := anchor.Bounds(); b.W > 0 {
		w = max(b.W, 280)
	}
	a.popOpen(userFloatKey(u.ID), func(open *bool) {
		ui.PopoverBase(c, anchor, open, func(panel ui.Element) {
			panel = a.popPanel(panel, w).Padding(8, 11, 6, 11).Gap(6).AlignItems(ui.Stretch)
			panel.Margin(4, 0, 0, 0)
			panel.Children(func() {
				lines := estimatedUserLines(userDisplayText(u.Text), 44)
				maxH := float32(userFoldExp) * a.FlowFontSize() * 1.55
				body := func() {
					ui.Column(c).Key("ufl-body-" + u.ID).Selectable().Children(func() { a.userMsgBody(c, u, false) })
				}
				if lines > userFoldExp {
					ui.Scroll(c).Key("ufl-scroll-" + u.ID).MaxHeight(maxH).Children(body)
				} else {
					body()
				}
				ui.Box(c).Height(1).Background(t.Bd1)
				ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "拖选可复制 · Esc 收起").FontSize(theme.FsMicro).TextColor(t.Tx4).Grow(1).SingleLine()
					kit.Btn(c, "复制", kit.BtnOpt{Small: true, Variant: kit.BtnGhost, Icon: "copy"}).OnClick(func() {
						a.copyOut(userDisplayText(u.Text), "消息")
					})
					kit.Btn(c, "编辑", kit.BtnOpt{Small: true, Variant: kit.BtnGhost, Icon: "edit"}).OnClick(func() {
						a.popup = ""
						a.beginUserEdit(u)
					})
				})
			})
		})
	})
}
