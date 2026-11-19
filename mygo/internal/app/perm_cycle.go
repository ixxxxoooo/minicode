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
// 六种模式都记(含自动 / 不再询问 / 跳过权限确认),换会话、重启后新会话都用最后一次选的;
// 设置里的「默认权限模式」只在从没选过时才起作用。禁止绕过权限时 DefaultPermMode 会回退,不会带出绕过。
func (a *App) rememberPermMode(m core.PermissionMode) {
	if a.prefs == nil {
		return
	}
	if a.prefs.Str(pPermLast, "") != m.CLI() {
		a.prefs.Set(pPermLast, m.CLI())
		log.Info("记住上次权限模式", "mode", m.CLI())
	}
	// 设置页里的下拉只有三档:自动 / 不再询问 / 绕过不写它,免得悄悄改掉「默认权限模式」
	if m == core.ModeBypassPermissions || m == core.ModeAuto || m == core.ModeDontAsk {
		return
	}
	if idx := permModeIndex(m); a.prefs.Int(pPermMode, 0) != idx || !a.prefs.Has(pPermMode) {
		a.prefs.Set(pPermMode, idx)
		log.Info("记住权限模式", "mode", m.CLI())
	}
}

// syncGlobalPermMode 让当前会话的权限模式对齐全局选择(DefaultPermMode = 最后一次选的模式)。
// 权限模式是全局偏好,不是会话属性:新会话、恢复历史会话(--resume 握手上报的是会话文件里存的旧模式)、
// 接回后台会话(沿用它离开时的模式)都要对齐,否则切换会话后选择就丢了。
// 演示后端的会话自带脚本化的模式(用来演示各种状态),只对新会话对齐。
func (a *App) syncGlobalPermMode(isNew bool) {
	if a.st == nil || a.be == nil || a.prefs == nil {
		return
	}
	if !isNew && a.be.Name() == "demo" {
		return
	}
	want := a.DefaultPermMode()
	if a.st.Mode == want {
		return
	}
	log.Info("权限模式对齐全局选择", "from", a.st.Mode.CLI(), "to", want.CLI(), "new", isNew)
	a.send(core.CmdSetMode{Mode: want})
	a.st.Mode = want
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
