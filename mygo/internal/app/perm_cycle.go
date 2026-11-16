package app

import (
	"os"
	"strings"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// cyclePermissionMode 循环切换权限模式(⇧Tab / 命令面板)。
func (a *App) cyclePermissionMode() {
	if a.st == nil || a.be == nil {
		a.toast = "还没有活动会话"
		return
	}
	next := a.st.Mode.Next()
	for a.NoBypassPermissions() && next == core.ModeBypassPermissions {
		next = next.Next()
	}
	if next == core.ModeBypassPermissions && a.NoBypassPermissions() {
		a.toast = "已禁止绕过权限模式"
		return
	}
	a.send(core.CmdSetMode{Mode: next})
	a.st.Mode = next
	a.rememberPermMode(next)
	a.toast = "权限模式:" + next.Label()
	log.Info("循环权限模式", "mode", next.CLI())
}

// rememberPermMode 记住在输入框里选的权限模式,新会话沿用(与模型 / 思考强度同一套做法)。
// 「跳过权限确认」风险高,不记:下次新会话仍回到上一次记住的安全模式。
func (a *App) rememberPermMode(m core.PermissionMode) {
	if a.prefs == nil || m == core.ModeBypassPermissions {
		return
	}
	if idx := permModeIndex(m); a.prefs.Int(pPermMode, 0) != idx || !a.prefs.Has(pPermMode) {
		a.prefs.Set(pPermMode, idx)
		log.Info("记住权限模式", "mode", m.CLI())
	}
}

func (a *App) hasActiveSession() bool {
	if a.st == nil || a.be == nil {
		return false
	}
	id := a.curSess
	return id != "" && !strings.HasPrefix(id, "new:")
}

// paletteOpenPlan 打开最新计划标签(无计划时提示)。
func (a *App) paletteOpenPlan() {
	if p := a.latestPlan(); p != nil {
		a.OpenWorkspaceTab(vPlan, p.ID)
		return
	}
	a.toast = "本轮还没有计划"
}

// paletteOpenTask 打开第一个后台任务输出标签。
func (a *App) paletteOpenTask() {
	if a.st == nil || len(a.st.Tasks) == 0 {
		a.toast = "没有可查看的后台任务"
		return
	}
	id := ""
	for _, tk := range a.st.Tasks {
		if tk.Status == core.TaskRunning {
			id = tk.TaskID
			break
		}
	}
	if id == "" {
		id = a.st.Tasks[0].TaskID
	}
	a.OpenWorkspaceTab(vTask, id)
}

// askSwitchRunMode 切换真实 / 演示并重启(开发版写入 dev-mode)。
func (a *App) askSwitchRunMode() {
	toDemo := !a.runDemo
	lbl := map[bool]string{true: "演示", false: "真实"}[toDemo]
	hint := ""
	if os.Getenv("MINI_DEV") != "1" {
		hint = "正式包请用设置 → 关于切换。"
	}
	body := "会重启本应用并改用" + lbl + "后端。"
	if hint != "" {
		body += " " + hint
	}
	a.confirmDlg("切换到"+lbl+"模式", body, "重启切换", false, func() {
		if toDemo {
			a.prefs.Set("adv.runmode", 1)
		} else {
			a.prefs.Set("adv.runmode", 0)
		}
		a.restartForSettings("运行模式")
	})
}
