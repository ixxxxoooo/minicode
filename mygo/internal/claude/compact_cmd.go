package claude

import "github.com/ixxxxoooo/minicode/internal/core"

func (b *Backend) onCompact() {
	if !b.turnActive { // 先让界面进入运行态并出现压缩卡,不等进程启动(冷启动要几百毫秒)
		b.turnActive = true
		b.emit(core.EvTurnStarted{})
	}
	b.emit(core.EvCompact{State: core.CompactRunning})
	if !b.ensureProc() {
		b.emit(core.EvCompact{State: core.CompactFailed, Err: "claude 进程没有启动"})
		return
	}
	if b.tr != nil {
		b.tr.CompactManual = true
	}
	// 不再追加一条假的「/compact」用户消息:压缩是会话流里独立的一张卡(设计稿 flow-compact 页)。
	uuid := newUUID()
	log.Info("请求压缩上下文")
	b.write(UserMessage("/compact", nil, false, uuid))
}
