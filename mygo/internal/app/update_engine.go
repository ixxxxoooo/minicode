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
		cur = Version
	}
	var fetch update.Fetcher = update.GitHubAppcastFetcher{}
	if demo || os.Getenv("MINI_UPDATE_DEMO") == "1" {
		fetch = update.DemoFetcher{Version: "0.2.2"}
	}
	target := update.BundlePath()
	a.upd = &update.Runner{
		Fetch:      fetch,
		CurrentVer: cur,
		OnProgress: a.onUpdateProgress,
		InstallFunc: func(staging, pkgPath string) error {
			script := filepath.Join(a.upd.WorkDir(), "replace-minicode.sh")
			if err := update.WriteReplaceScript(staging, target, script); err != nil {
				return err
			}
			log.Info("更新解包完成", "dir", staging, "pkg", pkgPath, "script", script)
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
	if p.UpToDate {
		a.updSt = updState{Phase: updLatest, Version: p.Version}
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
	if a.win != nil {
		a.win.Update(func() { a.syncBrand() })
	}
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
	script := filepath.Join(a.upd.WorkDir(), "replace-minicode.sh")
	if _, err := os.Stat(script); err != nil {
		a.toast = "找不到更新脚本,请重新检查更新"
		log.Warn("更新脚本缺失", "path", script, "err", err)
		return
	}
	a.bg("apply-update", func() func() {
		cmd := exec.Command("/bin/sh", script)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Start()
		return func() {
			if err != nil {
				a.toast = "无法启动更新脚本"
				log.Warn("启动更新脚本失败", "err", err)
				return
			}
			log.Info("退出以应用更新")
			os.Exit(0)
		}
	})
}
