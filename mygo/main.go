// MiniCode 的原生界面入口(MyGo,无 WebView)。
package main

import (
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/app"
	"github.com/ixxxxoooo/minicode/internal/backend"
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

func main() {
	dev := os.Getenv("MINI_DEV") != ""
	runMode := "real"
	if os.Getenv("MINI_DEMO") != "0" { // 真实后端尚未移植,目前只有演示后端
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
		opts.DevBadge = "演示模式"
	}
	a := app.New(opts)
	a.SetProjects(demo.Projects(), "fixbug")

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
		win.OnClosed(func() { diag.CleanExit("关闭窗口") })

		// 主线程卡死看门狗:每 500 ms 让主线程回一次执行
		diag.Watchdog(func(done func()) { mygo.RunOnMain(done) }, 400*time.Millisecond, 3*time.Second)

		speed := 1.0
		if v, err := strconv.ParseFloat(os.Getenv("MINI_DEMO_SLOW"), 64); err == nil && v > 0 {
			speed = v
		}
		a.SetSessionOpener(func(id string) backend.Backend { return demo.Session(id, speed) })
		a.Attach(demo.New(func() []demo.Step { return demo.FixBug(true) }, speed))
	})
	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) { diag.CleanExit("退出") })
	if err := mygo.App.Run(); err != nil {
		log.Error("运行失败", "err", err)
		os.Exit(1)
	}
}
