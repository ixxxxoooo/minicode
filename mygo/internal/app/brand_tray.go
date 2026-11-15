package app

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

func (a *App) trayIconName(sum traySum, frame int) string {
	if !a.prefs.Bool(pTrayDyn, true) {
		return "tray-idle"
	}
	base := "tray-" + sum.mode
	if sum.mode == "running" && !a.trayAnimStatic() {
		return base + "-" + strconv.Itoa(frame%3)
	}
	if _, err := os.Stat(iconAssetPath(base + ".png")); err != nil {
		return "tray-idle"
	}
	return base
}

func (a *App) trayAnimStatic() bool {
	if a.prefs == nil {
		return false
	}
	if a.prefs.Int(pTrayAnim, 0) == 2 { // 无
		return true
	}
	return a11yReduceMotion() || kit.SystemReduceMotion()
}

func (a *App) trayIconPath(sum traySum) string {
	style := 0
	if a.prefs != nil {
		style = a.prefs.Int(pTrayStyle, 0)
	}
	if style > 0 {
		name := a.trayIconName(sum, a.brand.trayFrame)
		if p := iconAssetPath(name); p != "" {
			return p
		}
	}
	return iconAssetPath(a.trayIconName(sum, a.brand.trayFrame))
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
	for _, s := range a.brandSessions() {
		s := s
		items = append(items, &mygo.MenuItem{
			Label: s.title + " · " + s.project,
			Click: func(*mygo.MenuItem, *mygo.Window) { a.OpenSession(s.id); mygo.App.Focus() },
		})
	}
	if len(items) > 0 {
		items = append(items, &mygo.MenuItem{Type: mygo.MenuItemSeparator})
	}
	items = append(items,
		&mygo.MenuItem{Label: "显示 MiniCode", Click: func(*mygo.MenuItem, *mygo.Window) { mygo.App.Focus() }},
		&mygo.MenuItem{Label: "设置…", Click: func(*mygo.MenuItem, *mygo.Window) { a.openSettings("appearance") }},
		&mygo.MenuItem{Type: mygo.MenuItemSeparator},
		&mygo.MenuItem{Label: "退出", Role: mygo.RoleQuit},
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
	if sum.pending > 0 {
		sum.mode = "waiting"
		sum.title = "MiniCode · " + strconv.Itoa(sum.pending) + " 个待确认"
	} else if sum.running > 0 && sum.mode != "demo" && sum.mode != "update" {
		sum.mode = "running"
		sum.title = "MiniCode · " + strconv.Itoa(sum.running) + " 个运行中"
	} else if sum.mode == "idle" {
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
					a.brand.trayFrame = (a.brand.trayFrame + 1) % 3
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
	sum := a.brandSummarize()
	data, err := os.ReadFile(a.trayIconPath(sum))
	if err != nil {
		return
	}
	colored := a.prefs != nil && a.prefs.Bool(pTrayColor, true) && !a11yIncreaseContrast() && !a11yReduceTransparency()
	_ = a.brand.tray.SetIcon(data, !colored || sum.mode == "idle" || sum.mode == "running")
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
