package app

import (
	"strings"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// @author ixxxxoooo

// sysNote 是一条要发给系统的通知(设计稿「通知横幅」):标题放项目名(带事件),正文一句话;
// 需要拍板的带「允许一次 / 拒绝」动作并标成时效性,出错也是时效性;图标是包内应用图标(系统自己取)。
type sysNote struct {
	title, body string
	sess        string // 所属会话 id:点击回到它,通知中心里同一会话叠成一组
	kind        int    // 0 普通 / 1 需要确认(带动作、时效性)/ 2 出错(时效性)
	permID      string // kind=1 时的权限请求 id
	icon        []byte // 当前所选样式的应用图标 PNG(发送时填入),作为通知附件,保证换了图标通知也跟着换
}

// 通知种类。
const (
	noteNormal = 0
	noteAsk    = 1
	noteErr    = 2
)

// projectLabel 是通知标题里的项目名;没有打开项目时用应用名。
func (a *App) projectLabel() string {
	if a.proj != "" {
		return baseName(a.proj)
	}
	return "MiniCode"
}

// maybeSystemNotify 应用在后台且偏好开启时,发一条 macOS 系统通知。
// Darwin+cgo 且在 .app 包内走 UserNotifications(点击回会话、带动作);否则 osascript(无点击回调)。
func (a *App) maybeSystemNotify(n sysNote) {
	if a.appInForeground() {
		return
	}
	a.sendSystemNotify(n)
}

// notifyTurnEnd 回合结束专用:general.notifyFront 开启(默认)时,前台也发系统通知并响提示音。
func (a *App) notifyTurnEnd(n sysNote) {
	if a.appInForeground() && !a.prefs.Bool("general.notifyFront", true) {
		return
	}
	a.sendSystemNotify(n)
}

// sendSystemNotify 检查通知开关 / 暂停状态后发出系统通知(声音由 general.sound 决定)。
func (a *App) sendSystemNotify(n sysNote) {
	if !a.prefs.Bool("general.notify", true) || a.notifyPaused() {
		return
	}
	n.sess = strings.TrimSpace(n.sess)
	sound := a.notifySoundOn()
	n.icon = a.dockIconData(a.tk != nil && a.tk.Palette.Dark)
	a.bg("system-notify", func() func() {
		return func() { a.notifyMacNative(n, sound) }
	})
}

func escOSA(s string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s)
}

// permNotifyBody 把权限请求写成一句话:「允许运行 cargo test?」。
func (a *App) permNotifyBody(r *core.PermissionRequest) string {
	if r == nil || r.Tool == nil {
		return "有一个操作等你批准"
	}
	detail := strings.TrimSpace(a.permDetail(r))
	if len([]rune(detail)) > 80 {
		detail = string([]rune(detail)[:80]) + "…"
	}
	if detail == "" {
		return permTitle(r)
	}
	switch r.Tool.Kind {
	case core.KindExecute:
		return "允许运行 " + detail + "?"
	case core.KindEdit, core.KindWrite:
		return "允许修改 " + detail + "?"
	case core.KindRead:
		return "允许读取 " + detail + "?"
	case core.KindFetch:
		return "允许访问 " + detail + "?"
	}
	return permTitle(r) + " " + detail
}

// afterEventNotify 回合结束 / 待审批 / 任务完成时的系统通知与通知中心。
// 历史回放区间的事件不会走到这里(见 afterEvent 的 replay 判断)。
func (a *App) afterEventNotify(ev core.Event) {
	switch e := ev.(type) {
	case core.EvTurnEnded:
		if e.Reason == core.StopEndTurn {
			a.onTurnFinished()
		}
		if e.Reason == core.StopError {
			if !a.appInForeground() {
				a.brand.trayErr = true // 托盘显示出错徽标,回到前台后清除
			}
			a.notifyTurnEnd(sysNote{title: a.projectLabel() + " · 回合出错", body: noteBody(a.st.Title, "点按查看原因"), sess: a.curSess, kind: noteErr})
		}
	case core.EvAppend:
		if r, ok := e.Item.(*core.PermissionRequest); ok {
			a.onNeedsApproval("权限请求", r)
		}
		if _, ok := e.Item.(*core.QuestionRequest); ok {
			a.onNeedsApproval("需要回答问题", nil)
		}
		if _, ok := e.Item.(*core.PlanApproval); ok {
			a.onNeedsApproval("计划待批准", nil)
		}
	case core.EvPromoteToPermission:
		a.onNeedsApproval("权限请求", e.Request)
	case core.EvPermissionResolved:
		clearMacNotification(e.ID) // 在应用里处理完了:撤掉系统横幅,免得上面的按钮还能点
	case core.EvTaskNotification:
		if e.Status == core.TaskCompleted {
			a.onTaskDone(e.TaskID, e.Summary)
		}
	}
}

// noteBody 取通知正文:优先用会话标题,没有就用兜底文案。
func noteBody(title, fallback string) string {
	if b := core.ShortTitle(title); b != "" {
		return b
	}
	return fallback
}

func (a *App) onTurnFinished() {
	body := noteBody(a.st.Title, "可以查看结果")
	a.postNotice("ok", "回合结束", body, nil)
	if !a.appInForeground() {
		a.brand.trayUnread = true // 托盘显示「已完成」徽标,回到前台后清除
	}
	a.playTurnDing()
	a.notifyTurnEnd(sysNote{title: a.projectLabel(), body: "已完成:" + body, sess: a.curSess})
}

func (a *App) onNeedsApproval(what string, r *core.PermissionRequest) {
	if !a.prefs.Bool("general.notifyAsk", true) {
		return
	}
	act := []noticeAction{{"打开会话", func() { a.screen = ScreenWork; a.revealChat() }}}
	if a.appInForeground() {
		if a.screen == ScreenWork && !a.chatVisible() {
			// IDE 形态里对话栏收着:权限卡 / 问答卡在对话栏里,看不见。只提示、不自动展开(不打断编辑器里的操作),
			// 「打开会话」才展开
			a.postNotice("warn", what, "对话栏已收起,点「打开会话」查看并处理", act)
			return
		}
		if a.screen == ScreenWork {
			// 正在看会话:权限卡 / 问答卡就在输入框上方,不再弹浮层遮挡它
			a.postNoticeQuiet("warn", what, "已在会话中显示", act)
			return
		}
		a.postNotice("warn", what, "点开会话处理", act)
		return
	}
	n := sysNote{title: a.projectLabel() + " · 需要确认", body: noteBody(a.st.Title, what), sess: a.curSess}
	switch {
	case r != nil:
		n.body, n.kind, n.permID = a.permNotifyBody(r), noteAsk, r.ID
	case what == "需要回答问题":
		n.title = a.projectLabel() + " · 需要你回答"
	case what == "计划待批准":
		n.title = a.projectLabel() + " · 计划待批准"
	}
	a.maybeSystemNotify(n)
	a.postNotice("warn", what, "会话在后台等待你", act)
}

func (a *App) onTaskDone(id, summary string) {
	tk := a.findTask(id)
	name := summary
	if tk != nil && tk.Description != "" {
		name = tk.Description
	}
	a.postNotice("ok", "后台任务完成", name, nil)
	a.maybeSystemNotify(sysNote{title: a.projectLabel() + " · 后台任务完成", body: name, sess: a.curSess})
}

// NotifyAction 处理系统通知上的「允许一次 / 拒绝」按钮。
// 只在这条请求仍在当前会话里等待时才执行;别的会话(或已经处理过)只回到那个会话,不替用户盲目放行。
func (a *App) NotifyAction(sess, permID, action string) {
	log.Info("通知动作", "action", action, "sess", sess, "perm", permID)
	var req *core.PermissionRequest
	if a.st != nil && (sess == "" || sess == a.curSess) {
		for _, it := range a.st.Items {
			if r, ok := it.(*core.PermissionRequest); ok && r.ID == permID && r.Resolved == "" {
				req = r
			}
		}
	}
	if req == nil {
		a.flash("这个请求已经处理过,或不在当前会话")
		if sess != "" && sess != a.curSess {
			a.OpenSessionFromNotify(sess)
		}
		return
	}
	opt := ""
	for _, o := range req.Options {
		if action == "allow" && o.Kind == core.AllowOnce && opt == "" {
			opt = o.ID
		}
		if action == "deny" && o.Kind.Rejects() && opt == "" {
			opt = o.ID
		}
	}
	if opt == "" {
		return
	}
	a.send(core.CmdResolvePermission{ID: req.ID, Option: opt})
}

// notifyDenied 系统通知被用户在「系统设置 → 通知」里关掉了:通知中心记一条并给跳转按钮(每次运行只提示一次)。
func (a *App) notifyDenied() {
	if a.notifyDeniedShown {
		return
	}
	a.notifyDeniedShown = true
	log.Warn("系统通知未授权")
	a.postNotice("warn", "系统通知已关闭", "回合结束、需要确认等提醒发不出来。在系统设置里打开 MiniCode 的通知权限。",
		[]noticeAction{{"打开系统设置", func() { a.openURL("x-apple.systempreferences:com.apple.Notifications-Settings.extension") }}})
}

func (a *App) ingestCompactNotice(text string) {
	if strings.Contains(text, "正在压缩") {
		a.ctxCompacting = true
	}
	if strings.Contains(text, "已压缩") {
		a.ctxCompacting = false
	}
}
