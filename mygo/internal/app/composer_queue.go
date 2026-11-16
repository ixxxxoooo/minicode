package app

import (
	"strings"
	"time"

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

// steerSend 立即引导:消息马上发给 Claude(priority: next),但先显示在输入框上方的待发送区,
// 被接收后才进会话流(设计稿 flow-queue)。空闲时没有可引导的回合,按普通发送。
func (a *App) steerSend(text string) {
	body, rels := splitDraftImages(text)
	body = strings.TrimSpace(body)
	if body == "" && len(rels) == 0 {
		return
	}
	if a.st.Phase != core.PhaseRunning {
		a.submitPrompt(text, false)
		return
	}
	show := body
	if show == "" {
		show = "(图片)"
	}
	a.q.addSteer(body, show, len(a.st.Items), len(rels), time.Now())
	log.Info("引导", "len", len(body), "待接收", len(a.q.steers))
	a.submitPrompt(text, true)
}

// syncSteers 每帧对照会话流更新引导状态;还有引导在等就继续要帧(最短停留到点后才放进会话流)。
func (a *App) syncSteers() {
	if len(a.q.steers) == 0 {
		return
	}
	a.q.syncSteers(a.st.Items, a.st.Phase == core.PhaseRunning, time.Now())
	if len(a.q.steers) > 0 {
		a.frameSoon()
	}
}

// visibleItems 会话流要画的条目:尚未被接收的引导,其回显的用户消息先藏起来(它在待发送区里)。
func (a *App) visibleItems() []core.Item {
	if len(a.q.steers) == 0 {
		return a.st.Items
	}
	out := make([]core.Item, 0, len(a.st.Items))
	for _, it := range a.st.Items {
		if u, ok := it.(*core.User); ok && a.q.held(u.ID) {
			continue
		}
		out = append(out, it)
	}
	return out
}

// busyKeys 回合进行中 ⏎ / ⌘⏎ 各自的含义(随「回合进行中再发消息」偏好对调)。
func (a *App) busyKeys() (enter, cmdEnter string) {
	if a.prefs.Int(pBusy, 0) == 1 {
		return "引导", "排队"
	}
	return "排队", "引导"
}
