// MiniCode 的原生界面入口(MyGo,无 WebView)。
package main

import (
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
	"github.com/ixxxxoooo/minicode/internal/backend"
	"github.com/ixxxxoooo/minicode/internal/claude"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/demo"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/prefs"
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
	if loc, ok := r.ix.Locate(id); ok && id != "" {
		cfg.Cwd, cfg.SessionID, cfg.HistoryPath = loc.Cwd, id, loc.Path
		r.curCwd = loc.Cwd
	}
	cb := r.onCwd
	r.mu.Unlock()
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
	demoMode := resolveDemo(dev)
	runMode := "real"
	if demoMode {
		runMode = "demo"
	}
	diag.Init(diag.Options{Version: version, Mode: runMode, Mirror: dev && os.Getenv("MINI_MIRROR") != ""})
	log.Info("启动", "version", version, "mode", runMode, "dev", dev)

	opts := app.Options{
		Palette: env("MINI_THEME", "paper"),
		Scheme:  env("MINI_SCHEME", "a"),
		Mode:    app.ModeAgent,
		Screen:  app.Screen(env("MINI_SCREEN", "work")),
		Prefs:   prefs.Open(filepath.Join(diag.Home(), "settings.json")),
	}
	if env("MINI_MODE", "agent") == "ide" {
		opts.Mode = app.ModeIDE
	}
	if dev {
		opts.DevBadge = map[bool]string{true: "演示模式", false: "开发版"}[demoMode]
	}
	a := app.New(opts)
	var rs *realSetup
	if demoMode {
		a.SetRepo(demo.Repo())
		a.SetProjects(demo.Projects(), "fixbug")
	} else {
		claude.SetCacheFile(filepath.Join(diag.Home(), "session-index-go.json"))
		rs = &realSetup{}
		ix := rs.refresh()
		rs.curCwd = initialCwd(ix)
		log.Info("真实模式", "projects", len(ix.Projects), "cwd", rs.curCwd)
		a.SetProjects(ix.Projects, "")
		a.SetProject(rs.curCwd)
		rs.onCwd = a.SetProject
	}

	mygo.App.WhenReady(func() {
		w, h := 1280, 800
		if v, err := strconv.Atoi(os.Getenv("MINI_WIN_W")); err == nil {
			w = v
		}
		if v, err := strconv.Atoi(os.Getenv("MINI_WIN_H")); err == nil {
			h = v
		}
		win := mygo.NewWindow(mygo.WindowOptions{
			Title:         "MiniCode",
			Width:         w,
			Height:        h,
			MinWidth:      900,
			MinHeight:     560,
			StateKey:      "main",
			TitleBarStyle: mygo.TitleBarHidden,
			Content:       ui.View(a.View),
		})
		a.SetWindow(win)
		win.OnClosed(func() { a.CloseTerminals(); diag.CleanExit("关闭窗口") })

		// 主线程卡死看门狗:每 500 ms 让主线程回一次执行
		diag.Watchdog(func(done func()) { mygo.RunOnMain(done) }, 400*time.Millisecond, 3*time.Second)

		speed := 1.0
		if v, err := strconv.ParseFloat(os.Getenv("MINI_DEMO_SLOW"), 64); err == nil && v > 0 {
			speed = v
		}
		if demoMode {
			a.SetSessionOpener(func(id string) backend.Backend {
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
		// 侧栏定时重扫:别处(终端里的 claude、本应用刚写盘的新会话)产生的会话也能出现
		go func() {
			defer diag.Recover("会话索引刷新")
			for range time.Tick(20 * time.Second) {
				ix := rs.refresh()
				win.Update(func() { a.RefreshProjects(ix.Projects) })
			}
		}()
	})
	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) { a.CloseTerminals(); diag.CleanExit("退出") })
	if err := mygo.App.Run(); err != nil {
		log.Error("运行失败", "err", err)
		os.Exit(1)
	}
}
