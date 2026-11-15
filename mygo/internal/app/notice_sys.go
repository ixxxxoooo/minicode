package app

import (
	"strings"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// maybeSystemNotify 应用在后台且偏好开启时,发一条 macOS 系统通知。
// Darwin+cgo 且在 .app 包内走 UserNotifications(点击 sess 回会话);否则 osascript(无点击回调)。
func (a *App) maybeSystemNotify(title, body, sessID string) {
	if a.appInForeground() || !a.prefs.Bool("general.notify", true) {
		return
	}
	sessID = strings.TrimSpace(sessID)
	t, b, sid := title, body, sessID
	sound := a.notifySoundOn()
	a.bg("system-notify", func() func() {
		return func() { a.notifyMacNative(t, b, sid, sound) }
	})
}

func escOSA(s string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s)
}

// afterEventNotify 回合结束 / 待审批 / 任务完成时的系统通知与通知中心。
func (a *App) afterEventNotify(ev core.Event) {
	switch e := ev.(type) {
	case core.EvTurnEnded:
		if e.Reason == core.StopEndTurn {
			a.onTurnFinished()
		}
	case core.EvAppend:
		if _, ok := e.Item.(*core.PermissionRequest); ok {
			a.onNeedsApproval("权限请求")
		}
		if _, ok := e.Item.(*core.QuestionRequest); ok {
			a.onNeedsApproval("需要回答问题")
		}
		if _, ok := e.Item.(*core.PlanApproval); ok {
			a.onNeedsApproval("计划待批准")
		}
	case core.EvPromoteToPermission:
		a.onNeedsApproval("权限请求")
	case core.EvTaskNotification:
		if e.Status == core.TaskCompleted {
			a.onTaskDone(e.TaskID, e.Summary)
		}
	}
}

func (a *App) onTurnFinished() {
	title := "回合结束"
	body := core.ShortTitle(a.st.Title)
	if body == "" {
		body = "可以查看结果"
	}
	a.postNotice("ok", title, body, nil)
	a.playTurnDing()
	a.maybeSystemNotify(title, body, a.curSess)
}

func (a *App) onNeedsApproval(what string) {
	if !a.prefs.Bool("general.notifyAsk", true) {
		return
	}
	act := []noticeAction{{"打开会话", func() { a.screen = ScreenWork }}}
	if a.appInForeground() {
		a.postNotice("warn", what, "点开会话处理", act)
		return
	}
	a.maybeSystemNotify(what, core.ShortTitle(a.st.Title), a.curSess)
	a.postNotice("warn", what, "会话在后台等待你", act)
}

func (a *App) onTaskDone(id, summary string) {
	tk := a.findTask(id)
	name := summary
	if tk != nil && tk.Description != "" {
		name = tk.Description
	}
	title := "后台任务完成"
	a.postNotice("ok", title, name, nil)
	a.maybeSystemNotify(title, name, a.curSess)
}

func (a *App) ingestCompactNotice(text string) {
	if strings.Contains(text, "正在压缩") {
		a.ctxCompacting = true
	}
	if strings.Contains(text, "已压缩") {
		a.ctxCompacting = false
	}
}
