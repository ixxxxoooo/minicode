// MiniCode 的原生界面入口(MyGo,无 WebView)。
package main

import (
	"context"
	_ "github.com/egoist/mygo/locales/zh-Hans" // 控件自带文案的简体中文译文
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/app"
	"github.com/ixxxxoooo/minicode/internal/automation"
	"github.com/ixxxxoooo/minicode/internal/backend"
	"github.com/ixxxxoooo/minicode/internal/claude"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/demo"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/prefs"
	"github.com/ixxxxoooo/minicode/internal/projects"
	"github.com/ixxxxoooo/minicode/internal/sessmeta"
)

// 构建时可用 -ldflags "-X main.version=..." 覆盖。
var log = diag.L("app")

var version = "0.2.0-mygo"

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// resolveDemo 决定运行模式。优先级:--demo / --real / MINI_DEMO=1|0 > 开发版记住的上次选择
// (~/.minicode/dev-mode)> 真实。开发版显式指定时会记住,下次不带开关沿用。
func resolveDemo(dev bool) bool {
	memo := filepath.Join(diag.Home(), "dev-mode")
	explicit := ""
	for _, a := range os.Args[1:] {
		switch a {
		case "--demo":
			explicit = "demo"
		case "--real":
			explicit = "real"
		}
	}
	switch os.Getenv("MINI_DEMO") {
	case "1":
		explicit = "demo"
	case "0":
		explicit = "real"
	}
	if explicit != "" {
		if dev {
			_ = os.MkdirAll(filepath.Dir(memo), 0o755)
			_ = os.WriteFile(memo, []byte(explicit), 0o644)
		}
		return explicit == "demo"
	}
	if dev {
		if b, err := os.ReadFile(memo); err == nil {
			return strings.TrimSpace(string(b)) == "demo"
		}
	}
	return false
}

// realSetup 是真实模式:读 ~/.claude/projects 建侧栏索引,按会话 id 造惰性 claude 后端。
type realSetup struct {
	mu     sync.Mutex
	ix     *claude.Index
	curCwd string       // 新会话的工作目录:最近打开的会话所在目录
	onCwd  func(string) // 打开会话后目录变了(切到别的项目)的回调,在主线程调用
	reg    *projects.Store
}

func (r *realSetup) refresh() *claude.Index {
	end := diag.SpanSlow("app", "扫描会话索引", 500*time.Millisecond)
	ix := claude.BuildIndex(claude.ProjectsDir(), time.Now())
	end()
	r.mu.Lock()
	r.ix = ix
	r.mu.Unlock()
	return ix
}

// open 为会话 id 造后端:已有会话 = 恢复并回放历史,空 id = 新会话。
func (r *realSetup) open(id string) backend.Backend {
	r.mu.Lock()
	cfg := claude.Config{Cwd: r.curCwd}
	if dir, ok := strings.CutPrefix(id, "new:"); ok { // 「在此项目新建会话」:换目录,不接旧会话
		if dir != "" {
			cfg.Cwd, r.curCwd = dir, dir
		}
		id = ""
	}
	if parts := strings.Split(id, ":"); len(parts) == 3 && parts[0] == "fork" {
		f := claude.ForkConfig{ParentSessionID: parts[1], NewSessionID: parts[2]}
		f.ApplyConfig(&cfg)
		id = parts[2]
	} else if loc, ok := r.ix.Locate(id); ok && id != "" {
		cfg.Cwd, cfg.SessionID, cfg.HistoryPath = loc.Cwd, id, loc.Path
		r.curCwd = loc.Cwd
	}
	cb := r.onCwd
	r.mu.Unlock()
	// 工作区的主文件夹启动会话时:其余文件夹用 --add-dir 授权,可选让 Claude 读取它们的 CLAUDE.md
	if w, ok := r.reg.ByPrimary(cfg.Cwd); ok {
		cfg.AddDirs = w.Extra()
		if w.LoadMD {
			cfg.Env = append(cfg.Env, core.EnvVar{Key: projects.LoadMDEnv, Value: "1"})
		}
	}
	log.Info("打开会话", "id", id, "cwd", cfg.Cwd)
	if cb != nil {
		cb(cfg.Cwd)
	}
	return claude.New(cfg)
}

// startUsage 后台统计用量(启动后立刻扫一次,之后每分钟增量重扫),返回给用量页读的数据源。
func startUsage(rs *realSetup, win *mygo.Window) func(rng int) core.UsageReport {
	var sc claude.UsageScanner
	var recs atomic.Pointer[[]claude.UsageRec]
	title := func(id string) (string, string) {
		rs.mu.Lock()
		ix := rs.ix
		rs.mu.Unlock()
		if ix != nil {
			if loc, ok := ix.Locate(id); ok {
				return loc.Title, loc.Cwd
			}
		}
		return "", ""
	}
	go func() {
		defer diag.Recover("用量统计")
		for {
			end := diag.SpanSlow("app", "扫描用量", time.Second)
			r := sc.Scan(claude.ProjectsDir())
			end()
			recs.Store(&r)
			win.Update(func() {})
			time.Sleep(time.Minute)
		}
	}()
	cache := map[int]core.UsageReport{}
	var cached *[]claude.UsageRec
	return func(rng int) core.UsageReport {
		p := recs.Load()
		if p == nil {
			return core.UsageReport{}
		}
		if p != cached { // 有新数据:丢掉旧报告
			cache, cached = map[int]core.UsageReport{}, p
		}
		if r, ok := cache[rng]; ok {
			return r
		}
		r := claude.BuildReport(*p, rng, time.Now(), title)
		cache[rng] = r
		return r
	}
}

// initialCwd 新会话默认目录:MINI_PROJECT > 最近有活动的项目 > 当前目录(从访达启动时是 /,退回用户主目录)。
func initialCwd(ix *claude.Index) string {
	if p := os.Getenv("MINI_PROJECT"); p != "" {
		return p
	}
	if len(ix.Projects) > 0 {
		return ix.Projects[0].Path
	}
	if wd, err := os.Getwd(); err == nil && wd != "/" {
		return wd
	}
	h, _ := os.UserHomeDir()
	return h
}

func main() {
	dev := os.Getenv("MINI_DEV") != ""
	secondary, nwProject := app.ParseArgs()
	demoMode := resolveDemo(dev)
	runMode := "real"
	if demoMode {
		runMode = "demo"
	}
	diag.Init(diag.Options{Version: version, Mode: runMode, Mirror: dev && os.Getenv("MINI_MIRROR") != ""})
	log.Info("启动", "version", version, "mode", runMode, "dev", dev)

	prefStore := prefs.Open(filepath.Join(diag.Home(), "settings.json"))
	startMode := app.ModeAgent
	switch env("MINI_MODE", "") {
	case "ide":
		startMode = app.ModeIDE
	case "agent":
		startMode = app.ModeAgent
	default:
		startMode = app.ResolveStartupMode(app.NewPrefsReader(prefStore.Int), false)
	}
	opts := app.Options{
		Palette:   env("MINI_THEME", "paper"),
		Scheme:    env("MINI_SCHEME", "a"),
		Mode:      startMode,
		Screen:    app.Screen(env("MINI_SCREEN", "work")),
		Prefs:     prefStore,
		Secondary: secondary,
	}
	if dev {
		opts.DevBadge = map[bool]string{true: "演示模式", false: "开发版"}[demoMode]
	}
	a := app.New(opts)
	var rs *realSetup
	if demoMode {
		a.SetRepo(demo.Repo())
		a.SetProjects(demo.Projects(), "fixbug")
		a.SeedDemoRegistry()
		if os.Getenv("MINI_WS") != "" { // 调试 / 截图:启动即进入演示工作区
			a.ActivateWorkspace("ws-fullstack")
		}
	} else {
		claude.SetCacheFile(filepath.Join(diag.Home(), "session-index-go.json"))
		regPath := filepath.Join(diag.Home(), "workspaces.json")
		var reg *projects.Store
		if secondary {
			reg = projects.NewMem()
		} else {
			reg = projects.Open(regPath)
		}
		a.SetRegistry(reg)
		rs = &realSetup{reg: reg}
		a.SetSessMeta(sessmeta.Open(filepath.Join(diag.Home(), "session-meta.json")))
		a.SetSessionDeleter(app.SessionDeleter(func(id string) (string, bool) {
			rs.mu.Lock()
			defer rs.mu.Unlock()
			if rs.ix == nil {
				return "", false
			}
			loc, ok := rs.ix.Locate(id)
			return loc.Cwd, ok
		}))
		ix := rs.refresh()
		if nwProject != "" {
			rs.curCwd = nwProject
		} else {
			rs.curCwd = initialCwd(ix)
		}
		log.Info("真实模式", "projects", len(ix.Projects), "cwd", rs.curCwd, "secondary", secondary)
		a.SetProjects(ix.Projects, "")
		a.SetProject(rs.curCwd)
		rs.onCwd = a.SetProject
	}

	mygo.App.WhenReady(func() {
		mygo.App.SetLocale("zh-Hans") // 界面是中文:控件自带文案(差异折叠条、菜单、对话框)也按简体中文
		w, h := 1280, 800
		if v, err := strconv.Atoi(os.Getenv("MINI_WIN_W")); err == nil {
			w = v
		}
		if v, err := strconv.Atoi(os.Getenv("MINI_WIN_H")); err == nil {
			h = v
		}
		sk := "main"
		if secondary {
			sk = "win:" + strconv.Itoa(os.Getpid())
		}
		wx, wy := 0, 0
		if v, err := strconv.Atoi(os.Getenv("MINI_WIN_X")); err == nil {
			wx = v
		}
		if v, err := strconv.Atoi(os.Getenv("MINI_WIN_Y")); err == nil {
			wy = v
		}
		win := mygo.NewWindow(mygo.WindowOptions{
			Title:         "MiniCode",
			Width:         w,
			Height:        h,
			X:             wx,
			Y:             wy,
			MinWidth:      720,
			MinHeight:     560,
			StateKey:      sk,
			TitleBarStyle: mygo.TitleBarHidden,
			// 红绿灯与 36 高的标题栏垂直居中(按钮高 14:顶边距 = (36−14)/2)
			TrafficLightPosition: &mygo.Point{X: 14, Y: 11},
			Content:              ui.View(a.View),
		})
		a.SetWindow(win)
		a.InstallAppMenu()
		a.InitBrand()
		win.OnClosed(func() {
			a.SaveLayMemToPrefs()
			a.CloseTerminals()
			diag.CleanExit("关闭窗口")
		})

		// 主线程卡死看门狗:每 500 ms 让主线程回一次执行
		diag.Watchdog(func(done func()) { mygo.RunOnMain(done) }, 400*time.Millisecond, 3*time.Second)

		speed := 1.0
		if v, err := strconv.ParseFloat(os.Getenv("MINI_DEMO_SLOW"), 64); err == nil && v > 0 {
			speed = v
		}
		if secondary && !demoMode {
			a.SetSessionOpener(rs.open)
			a.Attach(rs.open(""))
			return
		}
		if demoMode {
			a.SetSessionOpener(func(id string) backend.Backend {
				if strings.HasPrefix(id, "fork:") {
					return demo.ForkSession(id, speed)
				}
				if strings.HasPrefix(id, "new:") {
					id = ""
				}
				return demo.Session(id, speed)
			})
			a.Attach(demo.New(func() []demo.Step { return demo.FixBug(true) }, speed))
			return
		}
		a.SetSessionOpener(rs.open)
		a.Attach(rs.open(""))
		a.SetUsageSource(startUsage(rs, win))
		// 自动化:持久化到 ~/.minicode/automations.json,每半分钟检查一次到点的并用 claude -p 运行
		if home, err := os.UserHomeDir(); err == nil {
			a.SetAutomations(automation.OpenStore(filepath.Join(home, ".minicode", "automations.json")))
		}
		a.SetAutomationRunnerCtx(func(ctx context.Context, it automation.Automation, dir string, onChunk func(string)) (string, error) {
			defer diag.Recover("自动化运行")
			end := diag.SpanSlow("app", "运行自动化", 30*time.Second, "name", it.Name)
			defer end()
			return claude.RunPromptStreamCtx(ctx, dir, it.Prompt, claude.PermModeOf(it.Perm), 15*time.Minute, onChunk)
		})
		a.SetTextGenerator(func(dir, prompt string) (string, error) {
			end := diag.SpanSlow("app", "AI 提交说明", 20*time.Second)
			defer end()
			return claude.RunPromptModel(dir, prompt, "plan", "haiku", 60*time.Second)
		})
		go func() {
			defer diag.Recover("自动化定时")
			for range time.Tick(30 * time.Second) {
				win.Update(func() { a.AutomationTick(time.Now()) })
			}
		}()
		// 侧栏定时重扫:别处(终端里的 claude、本应用刚写盘的新会话)产生的会话也能出现
		go func() {
			defer diag.Recover("会话索引刷新")
			for range time.Tick(20 * time.Second) {
				win.Update(func() { a.SetProjScanning(true) })
				ix := rs.refresh()
				win.Update(func() {
					a.RefreshProjects(ix.Projects)
					a.SetProjScanning(false)
				})
			}
		}()
	})
	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) { a.CloseTerminals(); diag.CleanExit("退出") })
	if err := mygo.App.Run(); err != nil {
		log.Error("运行失败", "err", err)
		os.Exit(1)
	}
}
