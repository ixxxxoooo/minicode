package app

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/egoist/mygo"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/diag"
)

var brandLog = diag.L("brand")

// brand 管理 Dock 图标、角标与菜单栏托盘(非从属进程)。
type brand struct {
	mu        sync.Mutex
	tray      *mygo.Tray
	trayMode  string
	trayFrame int
	animStop  chan struct{}
	dockDark  bool
	lastBadge string
	lastA11y  string
	bounceID  int
}

func (a *App) initBrand() {
	if a.secondary {
		return
	}
	a.syncBrand()
}

func (a *App) syncBrand() {
	if a.secondary {
		return
	}
	dark := a.tk != nil && a.tk.Palette.Dark
	if a.brand.dockDark != dark {
		a.brand.dockDark = dark
		a.setDockIconFromPrefs(dark)
	}
	if a.prefs != nil {
		if mode := a.prefs.Int(pAppMode, 1); mode == 0 {
			if sysDark := systemIsDark(); sysDark != a.brand.dockDark {
				a.brand.dockDark = sysDark
				a.setDockIconFromPrefs(sysDark)
			}
		}
	}
	a.syncDockBadge()
	a.syncTray(dark)
	a.syncDockProgress()
}

func (a *App) setDockIcon(dark bool) {
	p := iconAssetPath(map[bool]string{true: "MiniCode-dark.icns", false: "MiniCode.icns"}[dark])
	if p == "" {
		// 退回 PNG
		p = iconAssetPath(map[bool]string{true: "app-dark.png", false: "app-light.png"}[dark])
	}
	data, err := os.ReadFile(p)
	if err != nil {
		brandLog.Debug("读取 Dock 图标失败", "path", p, "err", err)
		return
	}
	if strings.HasSuffix(p, ".icns") {
		if png, ok := loadIcnsAsPNG(p); ok {
			data = png
		}
	}
	if len(data) > 0 {
		if err := mygo.App.Dock.SetIcon(data); err != nil {
			brandLog.Warn("设置 Dock 图标失败", "err", err)
		}
	}
}

func iconAssetPath(name string) string {
	candidates := []string{
		filepath.Join("assets", "icon", name),
		filepath.Join("..", "assets", "icon", name),
	}
	if exe, err := os.Executable(); err == nil {
		root := filepath.Join(filepath.Dir(exe), "..", "..", "..")
		candidates = append(candidates, filepath.Join(root, "assets", "icon", name))
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "assets", "icon", name))
		candidates = append(candidates, filepath.Join(wd, "..", "assets", "icon", name))
		candidates = append(candidates, filepath.Join(wd, "..", "..", "assets", "icon", name))
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c
		}
	}
	return ""
}

type traySum struct {
	badge, pending, running int
	mode                    string // idle / running / waiting / unread / error / update / paused / demo
	title                   string
}

func traySummary(st *core.State) traySum {
	var sum traySum
	sum.mode = "idle"
	for _, it := range st.Items {
		switch x := it.(type) {
		case *core.PermissionRequest:
			if x.Resolved == "" {
				sum.badge++
			}
		case *core.QuestionRequest:
			if !x.Answered {
				sum.badge++
			}
		}
	}
	if st.RunningTaskCount() > 0 {
		sum.mode = "running"
		sum.title = "MiniCode,运行中"
	}
	if sum.badge > 0 {
		sum.mode = "waiting"
		sum.title = "MiniCode," + strconv.Itoa(sum.badge) + " 个待确认"
	}
	return sum
}

func (a *App) syncTray(dark bool) {
	_ = dark
	if a.prefs != nil && !a.prefs.Bool("general.tray", true) {
		a.stopTrayAnim()
		if a.brand.tray != nil {
			a.brand.tray.Destroy()
			a.brand.tray = nil
		}
		return
	}
	sum := a.brandSummarize()
	path := a.trayIconPath(sum)
	data, err := os.ReadFile(path)
	if err != nil {
		brandLog.Debug("托盘图标未找到", "path", path, "err", err)
		return
	}
	a11y := a.trayA11yLabel(sum)
	tip := a11y
	if sum.title != "" {
		tip = sum.title + " · " + a11y
	}
	template := a.prefs == nil || !a.prefs.Bool(pTrayColor, true) || a11yIncreaseContrast() || a11yReduceTransparency()
	if sum.mode != "idle" && sum.mode != "running" {
		template = true
	}
	clickMenu := a.prefs == nil || a.prefs.Int(pTrayClick, 0) == 0
	if a.brand.tray == nil {
		tr, err := mygo.NewTray(mygo.TrayOptions{
			ID:             "minicode.main",
			Icon:           data,
			IconIsTemplate: template,
			Title:          a.trayCountTitle(sum),
			ToolTip:        tip,
			Menu:           a.trayMenuFull(),
		})
		if err != nil {
			brandLog.Warn("创建托盘失败", "err", err)
			return
		}
		if !clickMenu {
			tr.OnClick(func() { mygo.App.Focus() })
		}
		a.brand.tray = tr
		a.brand.trayMode = sum.mode
		if a11y != a.brand.lastA11y {
			a.brand.lastA11y = a11y
			postA11yAnnouncement(a11y)
		}
		a.scheduleTrayAnim()
		return
	}
	if sum.mode != a.brand.trayMode {
		a.stopTrayAnim()
		_ = a.brand.tray.SetIcon(data, template)
		a.brand.trayMode = sum.mode
		a.scheduleTrayAnim()
	} else {
		a.refreshTrayIcon()
	}
	a.brand.tray.SetTitle(a.trayCountTitle(sum))
	a.brand.tray.SetToolTip(tip)
	if clickMenu {
		a.brand.tray.SetMenu(a.trayMenuFull())
	}
	if a11y != a.brand.lastA11y {
		a.brand.lastA11y = a11y
		postA11yAnnouncement(a11y)
	}
}
