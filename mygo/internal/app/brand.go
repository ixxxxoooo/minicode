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
	dockDark  bool
	lastBadge string
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
		a.setDockIcon(dark)
	}
	a.syncDockBadge()
	a.syncTray(dark)
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
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c
		}
	}
	return ""
}

func (a *App) syncDockBadge() {
	if a.st == nil {
		return
	}
	n := traySummary(a.st).badge
	label := ""
	if n > 0 {
		if n > 99 {
			label = "99+"
		} else {
			label = strconv.Itoa(n)
		}
	}
	if label == a.brand.lastBadge {
		return
	}
	a.brand.lastBadge = label
	mygo.App.Dock.SetBadge(label)
}

type traySum struct {
	badge int
	mode  string // idle / running / waiting / unread / error / update / paused / demo
	title string
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
	if a.prefs != nil && !a.prefs.Bool("general.tray", true) {
		if a.brand.tray != nil {
			a.brand.tray.Destroy()
			a.brand.tray = nil
		}
		return
	}
	sum := traySummary(a.st)
	iconName := "tray-" + sum.mode + ".png"
	if _, err := os.Stat(iconAssetPath(iconName)); err != nil {
		iconName = map[string]string{
			"running": "tray-running.png",
			"waiting": "tray-waiting.png",
			"unread":  "tray-unread.png",
		}[sum.mode]
		if iconName == "" {
			iconName = "tray-idle.png"
		}
	}
	data, err := os.ReadFile(iconAssetPath(iconName))
	if err != nil {
		return
	}
	if a.brand.tray == nil {
		tr, err := mygo.NewTray(mygo.TrayOptions{
			ID:             "minicode.main",
			Icon:           data,
			IconIsTemplate: true,
			ToolTip:        sum.title,
			Menu:           a.trayMenu(),
		})
		if err != nil {
			brandLog.Warn("创建托盘失败", "err", err)
			return
		}
		a.brand.tray = tr
		a.brand.trayMode = sum.mode
		return
	}
	if sum.mode != a.brand.trayMode {
		_ = a.brand.tray.SetIcon(data, true)
		a.brand.trayMode = sum.mode
	}
	if sum.title != "" {
		a.brand.tray.SetToolTip(sum.title)
	}
}

func (a *App) trayMenu() *mygo.Menu {
	return mygo.NewMenu([]*mygo.MenuItem{
		{Label: "显示 MiniCode", Click: func(*mygo.MenuItem, *mygo.Window) { mygo.App.Focus() }},
		{Type: mygo.MenuItemSeparator},
		{Label: "退出", Role: mygo.RoleQuit},
	})
}
