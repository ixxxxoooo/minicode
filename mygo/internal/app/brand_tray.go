package app

import (
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

// trayStyleIndex 菜单栏图标样式下标:与应用图标样式是同一个偏好(pAppStyle),越界按默认 0。
// 选一次样式,Dock / ⌘Tab / 访达 / 菜单栏同时换成同一种,两者不可能不一致。
func (a *App) trayStyleIndex() int { return a.iconStyleIndex() }

// iconStyleIndex 当前图标样式下标(应用图标与菜单栏图标共用),越界按默认 0。
func (a *App) iconStyleIndex() int {
	if a.prefs == nil {
		return 0
	}
	if i := a.prefs.Int(pAppStyle, 0); i >= 0 && i < len(appStyles) {
		return i
	}
	return 0
}

// trayEffMode 托盘实际显示的模式:关掉「随会话状态变化」时永远是空闲。
func (a *App) trayEffMode(mode string) string {
	if a.prefs != nil && !a.prefs.Bool(pTrayDyn, true) {
		return "idle"
	}
	return mode
}

// trayAssetFor 某样式、模式、动画帧对应的托盘素材名(样式 0 沿用 tray-<状态>.png,其余 tray-s<N>-<状态>.png);
// stepped 为真且模式是运行中时取「逐段点亮」的三帧之一。
func trayAssetFor(style int, mode string, frame int, stepped bool) string {
	pre := "tray"
	if style > 0 {
		pre = "tray-s" + strconv.Itoa(style)
	}
	if mode == "running" && stepped {
		return pre + "-running-" + strconv.Itoa(frame%3) + ".png"
	}
	return pre + "-" + mode + ".png"
}

// trayAssetName 当前托盘状态对应的素材文件名。
func (a *App) trayAssetName(sum traySum, frame int) string {
	mode := a.trayEffMode(sum.mode)
	return trayAssetFor(a.trayStyleIndex(), mode, frame, mode == "running" && !a.trayAnimStatic() && a.trayAnimMode() == 0)
}

// trayAnimMode 运行中动画:0 逐段点亮 / 1 呼吸 / 2 无。
func (a *App) trayAnimMode() int {
	if a.prefs == nil {
		return 0
	}
	return a.prefs.Int(pTrayAnim, 0)
}

func (a *App) trayAnimStatic() bool {
	if a.prefs != nil && a.trayAnimMode() == 2 { // 无
		return true
	}
	return a11yReduceMotion() || kit.SystemReduceMotion()
}

// trayBreathK 「呼吸」动画第 frame 帧的不透明度(一个周期 8 帧,在 45%~100% 之间起伏)。
func trayBreathK(frame int) float64 {
	ph := 2 * math.Pi * float64(frame%8) / 8
	return 0.45 + 0.55*(0.5+0.5*math.Cos(ph))
}

// trayRender 把单色模板图按模式 / 呼吸 / 彩色徽标处理成最终要画的图:breath < 0 表示不做呼吸;
// 有彩色徽标时返回非模板图(必须按菜单栏深浅 dark 上色),否则是模板图。真实托盘与设置里的动态预览共用这一处,所以两边一致。
func trayRender(data []byte, mode string, breath float64, color, dark bool) ([]byte, bool) {
	if breath >= 0 {
		data = fadeTrayPNG(data, breath)
	}
	if color {
		if tinted, ok := tintTrayPNG(data, mode, dark); ok {
			return tinted, false
		}
	}
	return data, true
}

// trayIconData 返回当前应显示的托盘图像与是否按模板图绘制。
// 优先用内嵌素材,找不到再退回磁盘上的 assets/icon(开发时直接改素材即时生效)。
func (a *App) trayIconData(sum traySum) (data []byte, template bool, err error) {
	name := a.trayAssetName(sum, a.brand.trayFrame)
	data, ok := embeddedIcon(name)
	if !ok {
		p := iconAssetPath(name)
		if p == "" {
			return nil, true, os.ErrNotExist
		}
		if data, err = os.ReadFile(p); err != nil {
			return nil, true, err
		}
	}
	mode := a.trayEffMode(sum.mode)
	breath := -1.0
	if mode == "running" && !a.trayAnimStatic() && a.trayAnimMode() == 1 {
		breath = trayBreathK(a.brand.trayFrame)
	}
	data, template = trayRender(data, mode, breath, a.trayWantsColor(), a.trayMenuBarDark())
	return data, template, nil
}

// trayWantsColor 是否使用彩色徽标:偏好开启,且没有「增强对比度 / 降低透明度」。
func (a *App) trayWantsColor() bool {
	return a.prefs != nil && a.prefs.Bool(pTrayColor, true) && !a11yIncreaseContrast() && !a11yReduceTransparency()
}

// trayMenuBarDark 菜单栏当前是否是深色(随壁纸 / 系统外观变,不等同于应用主题)。
func (a *App) trayMenuBarDark() bool {
	if a.brand.tray != nil {
		return a.brand.tray.Dark()
	}
	return systemIsDark()
}

func (a *App) trayCountTitle(sum traySum) string {
	if a.prefs == nil || !a.prefs.Bool(pTrayCount, true) {
		return ""
	}
	if sum.pending > 0 && (sum.mode == "waiting" || sum.mode == "running") {
		return strconv.Itoa(sum.pending)
	}
	if sum.running > 0 && sum.mode == "running" && sum.pending == 0 {
		return strconv.Itoa(sum.running)
	}
	return ""
}

func (a *App) trayA11yLabel(sum traySum) string {
	parts := []string{"MiniCode"}
	if sum.pending > 0 {
		parts = append(parts, strconv.Itoa(sum.pending)+" 个待确认")
	}
	if sum.running > 0 {
		parts = append(parts, strconv.Itoa(sum.running)+" 个运行中")
	}
	return strings.Join(parts, ",")
}

func (a *App) brandSessions() []traySess {
	var out []traySess
	add := func(id, title, proj string, kind string) {
		if id == "" {
			return
		}
		for _, x := range out {
			if x.id == id {
				return
			}
		}
		out = append(out, traySess{id: id, title: title, project: proj, kind: kind})
	}
	if a.st != nil && a.curSess != "" {
		kind := "idle" // 看过的会话不放标记;不在前台时刚完成的才是「未读」(蓝点)
		if a.brand.trayUnread {
			kind = "unread"
		}
		if a.st.AwaitingUser() {
			kind = "wait"
		} else if a.st.Phase == core.PhaseRunning {
			kind = "run"
		}
		add(a.curSess, a.sessionTitle(a.curSess), a.projectName(), kind)
	}
	for id, lv := range a.live {
		kind := ""
		switch {
		case lv.st != nil && lv.st.AwaitingUser():
			kind = "wait"
		case lv.st != nil && lv.st.Phase == core.PhaseRunning:
			kind = "run"
		case a.sessUnread[id]:
			kind = "unread" // 跑完留在后台、还没看过
		}
		if kind == "" {
			continue
		}
		add(id, a.sessionTitle(id), brandPathBase(lv.id), kind)
	}
	// 待确认排最前,其次运行中,再是未读,最后没有标记的;最多 6 个
	rank := map[string]int{"wait": 0, "run": 1, "unread": 2, "idle": 3}
	sort.SliceStable(out, func(i, j int) bool { return rank[out[i].kind] < rank[out[j].kind] })
	if len(out) > 6 {
		out = out[:6]
	}
	return out
}

type traySess struct {
	id, title, project, kind string
}

func (a *App) sessionTitle(id string) string {
	for _, p := range a.projects {
		for _, s := range p.Sessions {
			if s.ID == id {
				if s.Title != "" {
					return s.Title
				}
				break
			}
		}
	}
	if len(id) > 12 {
		return id[:12] + "…"
	}
	return id
}

func brandPathBase(id string) string {
	if i := strings.LastIndex(id, "/"); i >= 0 {
		return id[i+1:]
	}
	return id
}

func (a *App) trayMenuFull() *mygo.Menu {
	sum := a.brandSummarize()
	var items []*mygo.MenuItem
	head := sum.title
	if head == "" {
		head = a.trayA11yLabel(sum)
	}
	if head != "" {
		items = append(items, &mygo.MenuItem{Label: head, Disabled: true})
	}
	// 会话按项目分组:组名是灰色小标题(禁用项),行首只放一个小标记(运行 = 四点、待确认 = 橙点、未读 = 蓝点,其余留空对齐)
	sess := a.brandSessions()
	var groups []string
	byProj := map[string][]traySess{}
	for _, s := range sess {
		if _, ok := byProj[s.project]; !ok {
			groups = append(groups, s.project)
		}
		byProj[s.project] = append(byProj[s.project], s)
	}
	for _, g := range groups {
		items = append(items, &mygo.MenuItem{Type: mygo.MenuItemSeparator})
		items = append(items, &mygo.MenuItem{Label: trayClip(g, 20), Disabled: true})
		for _, s := range byProj[g] {
			s := s
			mk := trayMarkKind(s.kind)
			it := &mygo.MenuItem{
				Label:   trayClip(s.title, 24),
				ToolTip: s.title,
				Icon:    trayMarkIcon(mk),
				Click:   func(*mygo.MenuItem, *mygo.Window) { a.OpenSession(s.id); a.showMainWindow("托盘 / 通知") },
			}
			if it.Icon == nil || trayMarkAsText() {
				it.Icon = nil                          // 文字标记与图标二选一,避免重复
				it.Label = trayMarkText(mk) + it.Label // 图标建不出来(或实机不渲染)时退回文字标记
			}
			items = append(items, it)
		}
	}
	if len(sess) == 0 {
		items = append(items, &mygo.MenuItem{Type: mygo.MenuItemSeparator},
			&mygo.MenuItem{Label: "没有进行中的会话", Disabled: true})
	}
	items = append(items,
		&mygo.MenuItem{Type: mygo.MenuItemSeparator},
		&mygo.MenuItem{Label: "新对话", Click: func(*mygo.MenuItem, *mygo.Window) { a.NewSession(); a.showMainWindow("托盘 / 通知") }},
		&mygo.MenuItem{Label: "打开 MiniCode", Click: func(*mygo.MenuItem, *mygo.Window) { a.showMainWindow("托盘 / 通知") }},
		a.pauseMenu(),
		&mygo.MenuItem{Type: mygo.MenuItemSeparator},
		&mygo.MenuItem{Label: "设置…", Click: func(*mygo.MenuItem, *mygo.Window) { a.openSettings("general"); a.showMainWindow("托盘 / 通知") }},
		&mygo.MenuItem{Label: "检查更新…", Click: func(*mygo.MenuItem, *mygo.Window) {
			a.openSettings("about")
			a.checkUpdateAsync()
			a.showMainWindow("托盘 / 通知")
		}},
		&mygo.MenuItem{Label: "重启 MiniCode", Click: func(*mygo.MenuItem, *mygo.Window) { a.trayRestart() }},
		&mygo.MenuItem{Label: "退出 MiniCode", Role: mygo.RoleQuit},
	)
	return mygo.NewMenu(items)
}

// trayMarkKind 会话状态 → 标记图标种类(没有状态的行用透明占位,文字左缘才对得齐)。
func trayMarkKind(kind string) string {
	switch kind {
	case "wait", "run", "unread":
		return kind
	}
	return "blank"
}

// trayRestart 托盘「重启 MiniCode」:有正在运行的回合或未保存的文件时先回到窗口确认,否则直接重启。
// 不改「真实 / 演示」记忆,新进程继承当前环境与参数,模式不变。
func (a *App) trayRestart() {
	running := a.countRunningSessions()
	dirty := 0
	if a.ide != nil {
		for _, tb := range a.allFileTabs() {
			if tb.buf != nil && tb.dirty() {
				dirty++
			}
		}
	}
	setLog.Info("托盘请求重启", "running", running, "dirty", dirty)
	if running == 0 && dirty == 0 {
		a.restartApp("托盘菜单", false)
		return
	}
	a.showMainWindow("托盘 / 通知")
	var body string
	if running > 0 && dirty > 0 {
		body = strconv.Itoa(running) + " 个会话正在运行,另有 " + strconv.Itoa(dirty) + " 个文件未保存,重启会丢失这些内容。"
	} else if running > 0 {
		body = strconv.Itoa(running) + " 个会话正在运行,重启会中断它们(可稍后用「继续」接上)。"
	} else {
		body = strconv.Itoa(dirty) + " 个文件有未保存的修改,重启会丢失这些修改。"
	}
	a.confirmDlg("重启 MiniCode?", body, "重启", true, func() { a.restartApp("托盘菜单", false) })
}

func (a *App) brandSummarize() traySum {
	sum := traySummary(a.st)
	if a.demoWs() {
		sum.mode = "demo"
		sum.title = "MiniCode · 演示模式"
	}
	if a.updSt.Phase == updDownload || a.updSt.Phase == updReady || a.updSt.Phase == updUnpack {
		if sum.mode == "idle" || sum.mode == "running" {
			sum.mode = "update"
			sum.title = "MiniCode · 有更新"
		}
	}
	sum.running = a.countRunningSessions()
	sum.pending = sum.badge
	if a.prefs != nil {
		switch a.prefs.Int(pDockBadge, 1) {
		case 0:
			sum.badge = 0
		case 2:
			if sum.running > 0 {
				sum.badge = sum.running
			}
		}
	}
	// 出错 / 已完成只记「不在前台时发生的」:回到前台或新回合开始就算看过了
	if a.appInForeground() || sum.running > 0 {
		a.brand.trayErr, a.brand.trayUnread = false, false
	}
	// 设计稿优先级:待确认 > 出错 > 已完成 > 运行中 > 有更新 > 暂停 > 空闲(演示是叠加标记,只在没有更高状态时显示)
	switch {
	case sum.pending > 0:
		sum.mode = "waiting"
		sum.title = "MiniCode · " + strconv.Itoa(sum.pending) + " 个待确认"
	case a.brand.trayErr:
		sum.mode = "error"
		sum.title = "MiniCode · 会话出错"
	case a.brand.trayUnread:
		sum.mode = "unread"
		sum.title = "MiniCode · 有刚完成的会话"
	case sum.running > 0 && sum.mode != "demo" && sum.mode != "update":
		sum.mode = "running"
		sum.title = "MiniCode · " + strconv.Itoa(sum.running) + " 个运行中"
	case sum.mode == "idle" && a.notifyPaused():
		sum.mode = "paused"
		sum.title = "MiniCode · 通知已暂停"
	case sum.mode == "idle":
		sum.title = "MiniCode"
	}
	// 调试:MINI_TRAY=waiting|error|unread|update|paused|running|demo|idle 固定托盘状态,方便在真实菜单栏里逐态查看
	if m := os.Getenv("MINI_TRAY"); m != "" {
		switch m {
		case "waiting", "error", "unread", "update", "paused", "running", "demo", "idle":
			sum.mode = m
			sum.title = "MiniCode · 调试状态 " + m
			if m == "waiting" && sum.pending == 0 {
				sum.pending = 2
				sum.badge = 2
			}
		}
	}
	return sum
}

func (a *App) countRunningSessions() int {
	n := 0
	if a.st != nil && a.st.Phase == core.PhaseRunning {
		n++
	}
	for _, lv := range a.live { // 只数真正在跑的:回合结束后留在后台待回收的不算
		if lv.st != nil && lv.st.Phase == core.PhaseRunning {
			n++
		}
	}
	return n
}

func (a *App) scheduleTrayAnim() {
	if a.brand.tray == nil || a.trayAnimStatic() {
		return
	}
	sum := a.brandSummarize()
	if a.trayEffMode(sum.mode) != "running" {
		return
	}
	if a.brand.animStop != nil {
		return
	}
	a.brand.animStop = make(chan struct{})
	go func(stop <-chan struct{}) {
		defer diag.Recover("app.trayanim")
		tick := time.NewTicker(180 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-stop:
				return
			case <-tick.C:
				if a.win == nil {
					continue
				}
				a.win.Update(func() {
					a.brand.trayFrame = (a.brand.trayFrame + 1) % 24
					a.refreshTrayIcon()
				})
			}
		}
	}(a.brand.animStop)
}

func (a *App) stopTrayAnim() {
	if a.brand.animStop != nil {
		close(a.brand.animStop)
		a.brand.animStop = nil
	}
}

func (a *App) refreshTrayIcon() {
	if a.brand.tray == nil {
		return
	}
	data, template, err := a.trayIconData(a.brandSummarize())
	if err != nil {
		return
	}
	_ = a.brand.tray.SetIcon(data, template)
}

func (a *App) brandAfterEvent(ev core.Event) {
	switch ev.(type) {
	case core.EvAppend, core.EvPermissionResolved, core.EvTurnEnded, core.EvTaskStarted,
		core.EvTaskNotification, core.EvReady, core.EvPromoteToPermission:
		if _, ok := ev.(core.EvPromoteToPermission); ok {
			a.maybeDockBounce()
		}
		a.brandSyncSoon()
	}
}

// brandSyncMinGap 事件触发的托盘 / Dock 同步的最小间隔。
const brandSyncMinGap = 400 * time.Millisecond

// brandSyncSoon 合并事件触发的 syncBrand。
//
// 历史回放 / 流式输出会在很短时间内连发几百个 Append,原先每个事件都整套重建托盘菜单与图标,
// 打开长会话时界面线程被卡 400 ms 以上(日志「主线程停滞」与「归约事件 ev=Append」成串出现)。
// 现在:距上次同步已超过间隔就立刻同步(前沿,状态变化仍然及时);间隔内的后续事件只排一次尾沿同步,
// 在间隔结束时按最终状态补一次。托盘只看汇总状态,所以不会丢信息。
func (a *App) brandSyncSoon() {
	if a.win == nil { // 测试 / 启动阶段没有窗口:没有定时器可用,直接同步(syncBrand 自己会跳过)
		a.syncBrand()
		return
	}
	if a.brand.syncPending {
		return
	}
	wait := brandSyncMinGap - time.Since(a.brand.lastSync)
	if wait <= 0 {
		a.brand.lastSync = time.Now()
		a.syncBrand()
		return
	}
	a.brand.syncPending = true
	win := a.win
	time.AfterFunc(wait, func() {
		defer diag.Recover("托盘同步")
		win.Update(func() {
			a.brand.syncPending = false
			a.brand.lastSync = time.Now()
			a.syncBrand()
		})
	})
}

func (a *App) maybeDockBounce() {
	if a.prefs == nil || a.win == nil {
		return
	}
	mode := a.prefs.Int(pDockBnce, 1)
	if mode == 0 {
		return
	}
	if a11yReduceMotion() || kit.SystemReduceMotion() {
		return
	}
	critical := mode >= 2
	if a.brand.bounceID > 0 {
		dockCancelAttention(a.brand.bounceID)
	}
	a.brand.bounceID = dockRequestAttention(critical)
}

// trayClip 把菜单里的会话标题截到 n 个字符(按字符,不按字节),过长加省略号,免得托盘菜单被长标题撑得很宽。
func trayClip(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n]) + "…"
}

// pauseMenu 「暂停通知 ▸」二级:未暂停 = 1 小时 / 到明天 9:00 / 一直暂停;已暂停 = 恢复通知。
func (a *App) pauseMenu() *mygo.MenuItem {
	do := func(d time.Duration) func(*mygo.MenuItem, *mygo.Window) {
		return func(*mygo.MenuItem, *mygo.Window) { a.pauseNotifyFor(d) }
	}
	if a.notifyPaused() {
		return &mygo.MenuItem{Label: a.pauseLabel(), Submenu: []*mygo.MenuItem{
			{Label: "恢复通知", Click: do(-1)},
		}}
	}
	now := a.now()
	tomorrow := time.Date(now.Year(), now.Month(), now.Day()+1, 9, 0, 0, 0, now.Location())
	return &mygo.MenuItem{Label: a.pauseLabel(), Submenu: []*mygo.MenuItem{
		{Label: "暂停 1 小时", Click: do(time.Hour)},
		{Label: "暂停到明天 9:00", Click: do(tomorrow.Sub(now))},
		{Label: "一直暂停", Click: do(0)},
	}}
}
