package app

import (
	"strings"

	"github.com/ixxxxoooo/minicode/internal/core"
)

func (a *App) editingUser() bool { return len(a.userPinned) > 0 }

func (a *App) cancelUserEdit() { a.cancelTopUserEdit() }

func (a *App) submitUserEdit() {
	if len(a.userPinned) == 0 {
		return
	}
	a.submitPinnedUserEdit(a.userPinned[len(a.userPinned)-1].id)
}

func (a *App) submitPinnedUserEdit(id string) {
	draft := a.pinnedEditDraft(id)
	if draft == nil {
		return
	}
	txt := strings.TrimSpace(*draft)
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
	a.closePinnedUserEdit(id)
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
