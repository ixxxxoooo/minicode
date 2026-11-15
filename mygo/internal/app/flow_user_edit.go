package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

func (a *App) editingUser() bool { return a.userEd.id != "" }

func (a *App) userEditEscape() bool {
	if !a.editingUser() {
		return false
	}
	a.cancelUserEdit()
	return true
}

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
	col := ui.Column(c).Key("user-ed-"+u.ID).Background(t.Raised).Radius(t.RMd()).Padding(8, 10).Gap(6).Border(1, t.Ac)
	col.Children(func() {
		ed := ui.TextArea(c.Key("uedit"), &a.userEd.draft).Lines(3, 10).FontSize(a.FlowFontSize()).AutoFocus()
		ed.OnShortcut(0, ui.KeyEscape, func() { a.cancelUserEdit() })
		ui.Row(c).Gap(6).Justify(ui.End).Children(func() {
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
