package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

func (a *App) editingUser() bool { return a.userEd.id != "" }

func (a *App) submitUserEdit() {
	id := a.userEd.id
	txt := strings.TrimSpace(a.userEd.draft)
	if id == "" {
		return
	}
	if txt == "" {
		a.toast = "消息不能为空"
		return
	}
	uuid := a.st.UserUUIDs[id]
	be := ""
	if a.be != nil {
		be = a.be.Name()
	}
	if be == "claude" && uuid == "" {
		a.toast = "这条消息没有检查点记录,无法从该处重发"
		return
	}
	if be == "claude" && a.st.Phase == core.PhaseRunning {
		a.toast = "回合进行中,先结束或中断后再编辑"
		return
	}
	if _, ok := a.st.TruncateFromUser(id); !ok {
		return
	}
	a.cancelUserEdit()
	switch be {
	case "claude":
		a.send(core.CmdEditUser{UserID: id, Text: txt, MsgUUID: uuid})
	case "demo", "":
		a.send(core.CmdPrompt{Text: txt})
	default:
		a.send(core.CmdPrompt{Text: txt})
	}
	log.Info("编辑并发送用户消息", "id", id, "backend", be)
}

func (a *App) userEditCard(c *ui.Context, u *core.User) {
	t := a.tk
	// 只有卡片外圈一道描边;里面的输入区不画框、不画聚焦环,文字直接写在卡片上(设计稿 .msg-u.editing)。
	col := ui.Column(c).Key("user-ed-"+u.ID).Background(t.Raised.Over(t.BG)).Radius(t.RMd()).Padding(9, 11, 7, 11).Gap(6).AlignItems(ui.Stretch).Border(1, t.AcLine)
	for _, s := range t.Shadow1() {
		col.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
	}
	col.Children(func() {
		ed := ui.TextArea(c.Key("uedit"), &a.userEd.draft).Lines(3, 10).FontSize(a.FlowFontSize()).LineHeight(a.FlowReadLH()).
			Background(ui.Transparent).BorderWidth(0).Padding(0).FocusRing(false).TextColor(t.Tx1).AutoFocus()
		ed.OnShortcut(0, ui.KeyEscape, func() { a.cancelUserEdit() })
		ui.Box(c).Height(1).Background(t.Bd1) // 与操作行之间的分隔线(设计稿 .ed-acts)
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "发送后会回退这一轮之后的文件改动").FontSize(theme.FsMicro).TextColor(t.Tx3).Grow(1).SingleLine()
			kit.Btn(c, "取消", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(a.cancelUserEdit)
			canReal := a.be == nil || a.be.Name() != "claude" || a.st.UserUUIDs[u.ID] != ""
			sub := kit.Btn(c, "发送", kit.BtnOpt{Small: true, Variant: kit.BtnPri, Icon: "send"})
			if canReal {
				sub.OnClick(a.submitUserEdit)
			} else {
				sub.Opacity(0.45).Cursor(ui.CursorDefault)
			}
		})
		hint := ""
		if a.be != nil && a.be.Name() == "claude" && a.st.UserUUIDs[u.ID] == "" {
			hint = "无检查点:无法 resume_session_at;请用「分叉会话」从该回合重试"
		}
		if hint != "" {
			ui.Text(c, hint).FontSize(theme.FsMicro).TextColor(t.Tx4)
		}
	})
}
