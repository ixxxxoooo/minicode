package claude

import (
	"github.com/ixxxxoooo/minicode/internal/core"
)

// pendingUserEdit 等待 resume_session_at 应答后要发的正文。
type pendingUserEdit struct {
	text, uuid string
}

// ResumeSessionAtReq 尝试从某条用户消息处恢复会话(协议子类型,部分 claude 版本可能不支持)。
func ResumeSessionAtReq(id, msgUUID string) obj {
	return controlRequest(id, obj{"subtype": "resume_session_at", "user_message_id": msgUUID})
}

func (b *Backend) onEditUser(c core.CmdEditUser) {
	if b.turnActive {
		b.emit(notice(core.NoticeWarning, "回合进行中,先结束或中断后再编辑上一条消息"))
		return
	}
	if c.MsgUUID == "" {
		b.emit(notice(core.NoticeWarning, "这条消息没有会话检查点 id,无法从该处重发;请用「分叉会话」或只编辑本地演示"))
		// 演示 / 无 uuid 时仍允许当普通新消息发出
		if c.Text != "" {
			b.onPrompt(core.CmdPrompt{Text: c.Text})
		}
		return
	}
	b.pendingEdit = &pendingUserEdit{text: c.Text, uuid: c.MsgUUID}
	if !b.ensureProc() {
		b.pendingEdit = nil
		return
	}
	b.request(req{kind: rkResumeAt, uuid: c.MsgUUID}, func(id string) obj { return ResumeSessionAtReq(id, c.MsgUUID) })
	log.Info("编辑用户消息", "user", c.UserID, "msg", c.MsgUUID)
}

func (b *Backend) finishResumeAt(ok bool, errText string) {
	p := b.pendingEdit
	b.pendingEdit = nil
	if p == nil {
		return
	}
	if !ok {
		log.Warn("resume_session_at 失败", "err", errText)
		b.emit(notice(core.NoticeWarning, "无法从该消息处恢复会话("+errText+");已仅更新本地时间线,下一条将照常追加"))
	}
	if p.text == "" {
		return
	}
	b.onPrompt(core.CmdPrompt{Text: p.text})
}
