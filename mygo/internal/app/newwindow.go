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
		if p == "" {
			continue
		}
		// 目录已被删除(临时目录等)就不再列在「最近项目」里。
		if st, err := os.Stat(p); err != nil || !st.IsDir() {
			continue
		}
		out = append(out, p)
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

// installAppMenu 注册系统菜单(文件 → 新窗口 / 最近项目 / 关闭)。
// 从属窗口(⇧⌘N 起的进程)也装:否则它用的是库的默认菜单,⌘W 直接关窗口,不会先关标签;
// 它不能再开新窗口、不写最近项目,这两项不放。
func (a *App) installAppMenu() {
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
	// 快捷键由应用内按用户设置分发(可改键),所以这里的自定义项不挂 Accelerator,避免与改键后的绑定冲突;
	// 系统标准项(编辑 / 窗口 / 退出 / 隐藏…)用 Role,自带本地化文案与快捷键。
	settings := func(tab string) func(*mygo.MenuItem, *mygo.Window) {
		return func(*mygo.MenuItem, *mygo.Window) { a.openSettings(tab) }
	}
	fileMenu := []*mygo.MenuItem{
		{Label: "新建会话", Click: func(*mygo.MenuItem, *mygo.Window) { a.NewSession() }},
	}
	if !a.secondary {
		fileMenu = append(fileMenu,
			&mygo.MenuItem{Label: "新建窗口", Accelerator: "CmdOrCtrl+Shift+N", Click: func(*mygo.MenuItem, *mygo.Window) {
				a.SpawnNewWindow(a.proj)
			}},
			&mygo.MenuItem{Type: mygo.MenuItemSeparator},
			&mygo.MenuItem{Label: "打开最近的项目", Submenu: recent()},
		)
	}
	fileMenu = append(fileMenu,
		&mygo.MenuItem{Type: mygo.MenuItemSeparator},
		// 不用 RoleClose(它直接关窗口):⌘W 先看你在哪——在编辑器 / 终端的标签里只关标签,否则才关窗口(close_ctx.go)。
		// ⌘W 是系统约定的键位,不进可改键的快捷键表,所以挂在菜单上。
		&mygo.MenuItem{Label: "关闭", Accelerator: "CmdOrCtrl+W", Click: func(*mygo.MenuItem, *mygo.Window) { a.closeHere() }},
	)
	mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
		{Label: "MiniCode", Submenu: []*mygo.MenuItem{
			{Role: mygo.RoleAbout},
			{Type: mygo.MenuItemSeparator},
			{Label: "设置…", Click: func(*mygo.MenuItem, *mygo.Window) {
				tab := a.setTab
				if tab == "" {
					tab = "general"
				}
				a.openSettings(tab)
			}},
			{Type: mygo.MenuItemSeparator},
			{Role: mygo.RoleServices},
			{Type: mygo.MenuItemSeparator},
			{Role: mygo.RoleHide},
			{Role: mygo.RoleHideOthers},
			{Role: mygo.RoleUnhide},
			{Type: mygo.MenuItemSeparator},
			{Role: mygo.RoleQuit},
		}},
		{Label: "文件", Submenu: fileMenu},
		{Role: mygo.RoleEditMenu},
		{Label: "视图", Submenu: []*mygo.MenuItem{
			{Label: "Agent 形态", Click: func(*mygo.MenuItem, *mygo.Window) { a.SetMode(ModeAgent) }},
			{Label: "IDE 形态", Click: func(*mygo.MenuItem, *mygo.Window) { a.SetMode(ModeIDE) }},
			{Type: mygo.MenuItemSeparator},
			{Label: "命令面板…", Click: func(*mygo.MenuItem, *mygo.Window) { a.openPalette() }},
			{Type: mygo.MenuItemSeparator},
			{Role: mygo.RoleToggleFullScreen},
		}},
		{Role: mygo.RoleWindowMenu},
		{Label: "帮助", Role: mygo.RoleHelp, Submenu: []*mygo.MenuItem{
			{Label: "快捷键", Click: settings("shortcuts")},
			{Label: "检查更新", Click: settings("about")},
			{Label: "隐私与诊断", Click: settings("privacy")},
		}},
	}))
}

// InstallAppMenu 供 main 在窗口创建后调用:安装应用菜单(含新建窗口)。
func (a *App) InstallAppMenu() { a.installAppMenu() }

// InitBrand 供 main 调用:装配 Dock 图标与菜单栏托盘。
func (a *App) InitBrand() { a.initBrand() }

// SaveLayMemToPrefs 供 main 在窗口关闭时调用:把各形态的面板开合写入偏好。
func (a *App) SaveLayMemToPrefs() { a.saveLayMemToPrefs() }
