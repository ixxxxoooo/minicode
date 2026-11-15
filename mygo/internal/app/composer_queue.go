package app

import (
	"github.com/ixxxxoooo/minicode/internal/core"
)

// qStopAndSend 停止当前回合并在结束后自动发出排队中的一条。
func (a *App) qStopAndSend(i int) {
	if !a.q.stopAndSend(i) {
		return
	}
	log.Info("停止并发送", "队列", a.q.len(), "preempt", true)
	if a.st.Phase == core.PhaseRunning {
		a.send(core.CmdCancel{})
		return
	}
	if a.q.preempt != "" {
		a.flushPreempt()
	}
}

func (a *App) flushPreempt() {
	if a.q.preempt == "" {
		return
	}
	msg := a.q.preempt
	a.q.preempt = ""
	a.submitPrompt(msg, false)
}
