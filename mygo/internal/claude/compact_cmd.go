package claude

import "github.com/ixxxxoooo/minicode/internal/core"

func (b *Backend) onCompact() {
	if !b.ensureProc() {
		return
	}
	uuid := newUUID()
	text := "/compact"
	log.Info("请求压缩上下文")
	b.emit(b.tr.UserPrompt(text, nil, uuid)...)
	if !b.turnActive {
		b.turnActive = true
		if b.tr != nil {
			b.tr.Interrupted = false
		}
		b.emit(core.EvTurnStarted{})
	}
	b.write(UserMessage(text, nil, false, uuid))
}
