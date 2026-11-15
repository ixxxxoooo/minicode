package app

import (
	"os/exec"
	"strings"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// maybeSystemNotify 应用在后台且偏好开启时,发一条 macOS 系统通知(osascript)。
func (a *App) maybeSystemNotify(title, body, sessID string) {
	if a.appInForeground() || !a.prefs.Bool("general.notify", true) {
		return
	}
	sessID = strings.TrimSpace(sessID)
	script := `display notification "` + escOSA(body) + `" with title "` + escOSA(title) + `" subtitle "MiniCode"`
	if sessID != "" {
		script += ` sound name "Submarine"`
	}
	a.bg("osascript-notify", func() func() {
		err := exec.Command("osascript", "-e", script).Run()
		return func() {
			if err != nil {
				log.Warn("系统通知失败", "err", err)
			}
		}
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
	if a.mode == ModeIDE {
		a.noticeOK(title, body)
	}
	a.maybeSystemNotify(title, body, a.curSess)
}

func (a *App) onNeedsApproval(what string) {
	if !a.prefs.Bool("general.notifyAsk", true) {
		return
	}
	if a.appInForeground() && a.mode == ModeIDE {
		a.noticeWarn(what, "点开会话处理", []noticeAction{{"打开", func() { a.screen = ScreenWork }}})
		return
	}
	a.maybeSystemNotify(what, core.ShortTitle(a.st.Title), a.curSess)
	if a.mode == ModeIDE {
		a.noticeWarn(what, "会话在后台等待你", nil)
	}
}

func (a *App) onTaskDone(id, summary string) {
	tk := a.findTask(id)
	name := summary
	if tk != nil && tk.Description != "" {
		name = tk.Description
	}
	title := "后台任务完成"
	if a.mode == ModeIDE {
		a.noticeOK(title, name)
	}
	a.maybeSystemNotify(title, name, a.curSess)
}
