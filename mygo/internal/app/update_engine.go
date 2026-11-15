package app

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ixxxxoooo/minicode/internal/update"
)

func (a *App) initUpdater(demo bool) {
	if a.upd != nil {
		return
	}
	cur := os.Getenv("MINI_VERSION")
	if cur == "" {
		cur = "0.2.1"
	}
	var fetch update.Fetcher = update.GitHubAppcastFetcher{}
	if demo || os.Getenv("MINI_UPDATE_DEMO") == "1" {
		fetch = update.DemoFetcher{Version: "0.2.2"}
	}
	a.upd = &update.Runner{
		Fetch:      fetch,
		CurrentVer: cur,
		OnProgress: a.onUpdateProgress,
		InstallFunc: func(staging string) error {
			log.Info("更新解包完成", "dir", staging)
			return nil
		},
	}
}

func (a *App) onUpdateProgress(p update.Progress) {
	if p.Err != "" {
		log.Warn("更新失败", "err", p.Err)
		ph := updFail
		if strings.Contains(p.Err, "HTTP") || strings.Contains(p.Err, "network") {
			ph = updNetErr
		}
		a.updSt = updState{Phase: ph, Version: p.Version}
		a.update = a.updLabel()
		return
	}
	switch p.Phase {
	case update.PhaseDownload:
		a.updSt = updState{Phase: updDownload, Percent: p.Percent, Version: p.Version}
		if p.Percent < 8 {
			a.updSt.Phase = updFound
		}
	case update.PhaseVerify:
		a.updSt = updState{Phase: updVerify, Version: p.Version}
	case update.PhaseUnpack:
		a.updSt = updState{Phase: updUnpack, Version: p.Version}
	case update.PhaseReady:
		a.updSt = updState{Phase: updReady, Version: p.Version}
		a.postNotice("ok", "更新已就绪", "点「重启更新」应用", nil)
	default:
		a.updSt = updState{Phase: updChecking, Version: p.Version}
	}
	a.update = a.updLabel()
}

func (a *App) checkUpdateAsync() {
	if a.upd == nil {
		demo := a.be != nil && a.be.Name() == "demo"
		a.initUpdater(demo)
	}
	a.updSt = updState{Phase: updChecking}
	a.bg("update-check", func() func() {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		a.upd.Start(ctx)
		return nil
	})
}

func (a *App) applyUpdateRestart() {
	if a.updSt.Phase != updReady {
		a.toast = "还没有可安装的更新"
		return
	}
	log.Info("重启更新", "ver", a.updSt.Version)
	if os.Getenv("MINI_DEV") == "1" {
		a.bg("dev-restart", func() func() {
			root := filepath.Join(os.Getenv("HOME"), "MiniCode")
			if wd, err := os.Getwd(); err == nil && filepath.Base(wd) == "mygo" {
				root = filepath.Dir(wd)
			}
			err := exec.Command(filepath.Join(root, "scripts/dev.sh")).Run()
			return func() {
				if err != nil {
					a.toast = "重启失败,请手动运行 scripts/dev.sh"
					log.Warn("开发版重启失败", "err", err)
					return
				}
				os.Exit(0)
			}
		})
		return
	}
	a.toast = "正式版将在退出后替换应用包并重启(当前为占位)"
}
