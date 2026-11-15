package app

import (
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

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
	// 用户不在前台时发生的「回合出错 / 回合完成」:回到前台(或开始新回合)即清除,驱动托盘 error / unread 状态
	trayErr, trayUnread bool
	// 限时暂停通知的到期定时器
	pauseTimer *time.Timer
	// 访达图标上次同步的签名(开关 + 样式 + 外观),不变就不重复写
	finderSig string
	// 缺图标只警告一次,避免每次同步刷屏
	warnedMissing bool
}

func (a *App) initBrand() {
	if a.secondary {
		return
	}
	a.schedulePauseExpiry() // 重启后仍在暂停期内:到点自动恢复
	a.syncBrand()
}

func (a *App) syncBrand() {
	if a.secondary || testing.Testing() { // 单测里没有主循环,Dock / 托盘调用会永久阻塞
		return
	}
	if !mygo.App.IsReady() { // 应用就绪前创建托盘会直接 panic(New 里的 retheme 也会走到这);就绪后 InitBrand 会补一次
		return
	}
	dark := a.tk != nil && a.tk.Palette.Dark
	if a.brand.dockDark != dark {
		a.brand.dockDark = dark
		a.setDockIconFromPrefs(dark)
	}
	if a.prefs != nil {
		if mode := a.prefs.Int(pAppMode, 0); mode == 0 {
			if sysDark := systemIsDark(); sysDark != a.brand.dockDark {
				a.brand.dockDark = sysDark
				a.setDockIconFromPrefs(sysDark)
			}
		}
	}
	a.syncFinderIcon(dark)
	a.syncDockBadge()
	a.syncTray(dark)
	a.syncDockProgress()
}

func iconAssetPath(name string) string {
	candidates := []string{
		filepath.Join("assets", "icon", name),
		filepath.Join("..", "assets", "icon", name),
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		// .app 包:Contents/MacOS/minicode → Contents/Resources(从访达 / open 启动时工作目录是 /,只能靠这个找到)
		candidates = append(candidates, filepath.Join(dir, "..", "Resources", name))
		root := filepath.Join(dir, "..", "..", "..")
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
	data, template, err := a.trayIconData(sum)
	if err != nil {
		if !a.brand.warnedMissing {
			a.brand.warnedMissing = true
			brandLog.Warn("托盘图标未找到,菜单栏将不显示托盘", "name", a.trayAssetName(sum, 0), "err", err)
		}
		return
	}
	a11y := a.trayA11yLabel(sum)
	tip := a11y
	if sum.title != "" {
		tip = sum.title + " · " + a11y
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
		// 彩色徽标依赖菜单栏深浅:创建后按真实外观重画一次,之后随外观变化(壁纸 / 系统切换)自动重画
		a.refreshTrayIcon()
		tr.OnAppearanceChange(func() { a.refreshTrayIcon() })
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
