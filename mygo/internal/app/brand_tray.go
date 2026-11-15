package app

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

// trayStyleIndex 菜单栏图标样式下标(织纹 / 提示符 / 棋盘 / 行 / 花括号),越界按默认 0。
func (a *App) trayStyleIndex() int {
	if a.prefs == nil {
		return 0
	}
	if i := a.prefs.Int(pTrayStyle, 0); i >= 0 && i < len(trayStyles) {
		return i
	}
	return 0
}

// trayAssetName 当前托盘状态对应的素材文件名(样式 0 沿用 tray-<状态>.png,其余 tray-s<N>-<状态>.png)。
func (a *App) trayAssetName(sum traySum, frame int) string {
	mode := sum.mode
	if a.prefs != nil && !a.prefs.Bool(pTrayDyn, true) {
		mode = "idle" // 关掉「随会话状态变化」:永远是空闲图
	}
	pre := "tray"
	if i := a.trayStyleIndex(); i > 0 {
		pre = "tray-s" + strconv.Itoa(i)
	}
	if mode == "running" && !a.trayAnimStatic() && a.trayAnimMode() == 0 {
		return pre + "-running-" + strconv.Itoa(frame%3) + ".png" // 织线流动:三帧轮流点亮
	}
	return pre + "-" + mode + ".png"
}

// trayAnimMode 运行中动画:0 织线流动 / 1 呼吸 / 2 无。
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
	if sum.mode == "running" && !a.trayAnimStatic() && a.trayAnimMode() == 1 {
		// 呼吸:一个周期 8 帧,透明度在 45%~100% 之间起伏
		ph := 2 * math.Pi * float64(a.brand.trayFrame%8) / 8
		data = fadeTrayPNG(data, 0.45+0.55*(0.5+0.5*math.Cos(ph)))
	}
	if a.trayWantsColor() {
		if tinted, ok := tintTrayPNG(data, sum.mode, a.trayMenuBarDark()); ok {
			return tinted, false, nil
		}
	}
	return data, true, nil
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
		kind := "ok"
		if a.st.AwaitingUser() {
			kind = "wait"
		} else if a.st.Phase == core.PhaseRunning {
			kind = "run"
		}
		add(a.curSess, a.sessionTitle(a.curSess), a.projectName(), kind)
	}
	for id, lv := range a.live {
		kind := "run"
		if lv.st != nil && lv.st.AwaitingUser() {
			kind = "wait"
		}
		add(id, a.sessionTitle(id), brandPathBase(lv.id), kind)
	}
	// 待确认排最前,其次运行中,最后已完成;最多 5 个
	rank := map[string]int{"wait": 0, "run": 1, "ok": 2}
	sort.SliceStable(out, func(i, j int) bool { return rank[out[i].kind] < rank[out[j].kind] })
	if len(out) > 5 {
		out = out[:5]
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

// trayDotIcon 菜单里会话行前的状态点(蓝 = 运行中、橙 = 待确认、绿 = 刚完成,与应用内会话列表同一套),按种类缓存。
var trayDotCache = map[string]*mygo.MenuIcon{}

func trayDotIcon(kind string) *mygo.MenuIcon {
	if ic, ok := trayDotCache[kind]; ok {
		return ic
	}
	col := map[string]color.NRGBA{
		"run":  {0x0a, 0x84, 0xff, 0xff},
		"wait": {0xff, 0x9f, 0x0a, 0xff},
		"ok":   {0x30, 0xd1, 0x58, 0xff},
	}[kind]
	const n = 32 // 16pt @2x
	img := image.NewNRGBA(image.Rect(0, 0, n, n))
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			d := math.Hypot(float64(x)+.5-n/2, float64(y)+.5-n/2)
			al := math.Max(0, math.Min(1, 6.5-d)) // 半径 6.5px,边缘 1px 抗锯齿
			if al > 0 {
				c := col
				c.A = uint8(al * 255)
				img.SetNRGBA(x, y, c)
			}
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	ic, err := mygo.NewMenuIcon(buf.Bytes(), false)
	if err != nil {
		ic = nil
	}
	trayDotCache[kind] = ic
	return ic
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
	sess := a.brandSessions()
	for _, s := range sess {
		s := s
		items = append(items, &mygo.MenuItem{
			Label:    s.title,
			Subtitle: s.project,
			Icon:     trayDotIcon(s.kind),
			Click:    func(*mygo.MenuItem, *mygo.Window) { a.OpenSession(s.id); mygo.App.Focus() },
		})
	}
	if len(sess) == 0 {
		items = append(items, &mygo.MenuItem{Label: "没有进行中的会话", Disabled: true})
	}
	items = append(items,
		&mygo.MenuItem{Type: mygo.MenuItemSeparator},
		&mygo.MenuItem{Label: "新对话", Click: func(*mygo.MenuItem, *mygo.Window) { a.NewSession(); mygo.App.Focus() }},
		&mygo.MenuItem{Label: "打开 MiniCode", Click: func(*mygo.MenuItem, *mygo.Window) { mygo.App.Focus() }},
		&mygo.MenuItem{Label: "暂停通知", Type: mygo.MenuItemCheckbox, Checked: a.notifyPaused(),
			Click: func(it *mygo.MenuItem, _ *mygo.Window) {
				if a.prefs != nil {
					a.prefs.Set(pNotifyPause, it.Checked) // 复选项在回调前已翻转
				}
				brandLog.Info("托盘:切换暂停通知", "paused", it.Checked)
				a.syncBrand()
			}},
		&mygo.MenuItem{Type: mygo.MenuItemSeparator},
		&mygo.MenuItem{Label: "设置…", Click: func(*mygo.MenuItem, *mygo.Window) { a.openSettings("general"); mygo.App.Focus() }},
		&mygo.MenuItem{Label: "检查更新…", Click: func(*mygo.MenuItem, *mygo.Window) {
			a.openSettings("about")
			a.checkUpdateAsync()
			mygo.App.Focus()
		}},
		&mygo.MenuItem{Label: "退出 MiniCode", Role: mygo.RoleQuit},
	)
	return mygo.NewMenu(items)
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
	return sum
}

func (a *App) countRunningSessions() int {
	n := 0
	if a.st != nil && a.st.Phase == core.PhaseRunning {
		n++
	}
	for range a.live {
		n++
	}
	return n
}

func (a *App) scheduleTrayAnim() {
	if a.brand.tray == nil || a.trayAnimStatic() {
		return
	}
	sum := a.brandSummarize()
	if sum.mode != "running" {
		return
	}
	if a.brand.animStop != nil {
		return
	}
	a.brand.animStop = make(chan struct{})
	go func(stop <-chan struct{}) {
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
		a.syncBrand()
	}
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
