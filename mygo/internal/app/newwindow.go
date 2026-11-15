package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/egoist/mygo"
	"github.com/ixxxxoooo/minicode/internal/diag"
)

var nwLog = diag.L("newwindow")

const kRecentProjects = "general.recentProjects"

// ParseArgs 解析进程参数,返回是否从属新窗口、初始项目路径与虚拟标签说明(plan:ref)。
func ParseArgs() (secondary bool, project, virt string) {
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--new-window":
			secondary = true
		case "--project":
			if i+1 < len(args) {
				i++
				project = args[i]
			}
		case "--virt":
			if i+1 < len(args) {
				i++
				virt = args[i]
			}
		}
	}
	if virt == "" {
		virt = os.Getenv("MINI_VIRT")
	}
	return secondary, project, virt
}

// SpawnNewWindow 再起一个本应用进程(⇧⌘N)。
func (a *App) SpawnNewWindow(project string) {
	exe, err := os.Executable()
	if err != nil {
		nwLog.Warn("无法取得可执行文件路径", "err", err)
		return
	}
	args := []string{"--new-window"}
	if project != "" {
		args = append(args, "--project", project)
	}
	cmd := exec.Command(exe, args...)
	cmd.Env = os.Environ()
	if a.win != nil {
		b := a.win.Bounds()
		off := 32
		cmd.Env = append(cmd.Env,
			"MINI_WIN_W="+strconv.Itoa(b.Width),
			"MINI_WIN_H="+strconv.Itoa(b.Height),
			"MINI_WIN_X="+strconv.Itoa(b.X+off),
			"MINI_WIN_Y="+strconv.Itoa(b.Y+off),
		)
	}
	nwLog.Info("打开新窗口", "project", project)
	if err := cmd.Start(); err != nil {
		nwLog.Warn("启动新窗口失败", "err", err)
		a.toast = "无法打开新窗口"
	}
}

func (a *App) noteRecentProject(dir string) {
	if a.secondary || dir == "" || a.prefs == nil {
		return
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		abs = dir
	}
	raw := a.prefs.Str(kRecentProjects, "")
	var list []string
	for _, p := range strings.Split(raw, "\n") {
		p = strings.TrimSpace(p)
		if p != "" && p != abs {
			list = append(list, p)
		}
	}
	list = append([]string{abs}, list...)
	if len(list) > 12 {
		list = list[:12]
	}
	a.prefs.Set(kRecentProjects, strings.Join(list, "\n"))
}

func (a *App) recentProjects() []string {
	if a.prefs == nil {
		return nil
	}
	raw := a.prefs.Str(kRecentProjects, "")
	var out []string
	for _, p := range strings.Split(raw, "\n") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// OpenRecentProject 打开最近项目之一。
func (a *App) OpenRecentProject(path string) {
	if path == "" {
		return
	}
	nwLog.Info("打开最近项目", "path", path)
	if a.reg != nil {
		if _, err := a.reg.AddProject(path); err != nil {
			nwLog.Warn("添加最近项目失败", "err", err)
		}
	}
	a.SetProject(path)
	if a.mode == ModeIDE {
		a.nav, a.sideOpen = "files", true
	}
}

// installAppMenu 注册系统菜单(文件 → 新窗口 / 最近项目)。
func (a *App) installAppMenu() {
	if a.secondary {
		return
	}
	recent := func() []*mygo.MenuItem {
		var items []*mygo.MenuItem
		for _, p := range a.recentProjects() {
			p := p
			items = append(items, &mygo.MenuItem{
				Label:    filepath.Base(p),
				Subtitle: p,
				Click:    func(*mygo.MenuItem, *mygo.Window) { a.OpenRecentProject(p) },
			})
		}
		if len(items) == 0 {
			items = append(items, &mygo.MenuItem{Label: "尚无最近项目", Disabled: true})
		}
		return items
	}
	mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
		{Label: "MiniCode", Submenu: []*mygo.MenuItem{{Role: mygo.RoleAbout}}},
		{Label: "文件", Submenu: []*mygo.MenuItem{
			{Label: "新建窗口", Accelerator: "CmdOrCtrl+Shift+N", Click: func(*mygo.MenuItem, *mygo.Window) {
				a.SpawnNewWindow(a.proj)
			}},
			{Type: mygo.MenuItemSeparator},
			{Label: "打开最近的项目", Submenu: recent()},
		}},
	}))
}

// InstallAppMenu 供 main 在窗口创建后调用:安装应用菜单(含新建窗口)。
func (a *App) InstallAppMenu() { a.installAppMenu() }

// InitBrand 供 main 调用:装配 Dock 图标与菜单栏托盘。
func (a *App) InitBrand() { a.initBrand() }

// SaveLayMemToPrefs 供 main 在窗口关闭时调用:把各形态的面板开合写入偏好。
func (a *App) SaveLayMemToPrefs() { a.saveLayMemToPrefs() }
