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
	gh := update.GitHubAppcastFetcher{}
	if a.prefs != nil && a.prefs.Int(pAboutChan, 0) == 1 {
		gh.Channel = "beta"
	}
	var fetch update.Fetcher = gh
	if demo || os.Getenv("MINI_UPDATE_DEMO") == "1" {
		fetch = update.DemoFetcher{Version: "0.2.2"}
	}
	target := update.BundlePath()
	a.upd = &update.Runner{
		Fetch:      fetch,
		CurrentVer: cur,
		OnProgress: func(p update.Progress) {
			// 引擎在后台线程回调:状态一律回主线程改,错误态也能立刻重绘
			if a.win != nil {
				a.win.Update(func() { a.onUpdateProgress(p) })
			} else {
				a.onUpdateProgress(p)
			}
		},
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
		log.Warn("更新失败", "stage", p.ErrStage, "err", p.Err)
		a.updSt = failedUpdState(p)
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
		a.upd.Run(ctx) // 同步跑完再 cancel;用 Start 会在返回瞬间把自己取消
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

// retryUpdate 更新失败后的「重试」:重新检查更新(演示与真实共用)。
func (a *App) retryUpdate() {
	a.checkUpdateAsync()
}

// failedUpdState 把更新引擎的失败翻成界面状态:阶段决定标题与归类,原因写成一句话加下一步。
func failedUpdState(p update.Progress) updState {
	st := updState{Phase: updFail, Version: p.Version}
	e := p.Err
	has := func(subs ...string) bool {
		for _, s := range subs {
			if strings.Contains(e, s) {
				return true
			}
		}
		return false
	}
	switch p.ErrStage {
	case "check":
		st.Phase = updNetErr
		switch {
		case has("HTTP 404"):
			st.Title, st.Msg = "检查失败", "还没有找到发布信息(可能尚未发布正式版本)。可在设置 → 关于里打开 GitHub 查看。"
		case has("context canceled", "deadline exceeded", "timeout", "Timeout"):
			st.Msg = "请求超时。检查网络或代理后点「重试」。"
		case has("no such host", "dial tcp", "connection refused", "connection reset", "network is unreachable"):
			st.Msg = "连不上 GitHub。检查网络或代理(设置 → 运行环境)后点「重试」。"
		default:
			st.Title, st.Msg = "检查失败", "无法读取更新信息:"+shortErr(e)+"。稍后点「重试」。"
		}
	case "download":
		st.Msg = "下载中断:" + shortErr(e) + "。点「重试」重新下载。"
	case "verify":
		st.Title, st.Msg = "校验失败", "下载包与发布的校验值不一致,可能已损坏。点「重试」重新下载。"
	case "unpack", "install":
		st.Title, st.Msg = "安装准备失败", shortErr(e)+"。点「重试」,仍失败请手动下载安装包。"
	}
	return st
}

// shortErr 取错误的第一行并截断,供界面显示。
func shortErr(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if r := []rune(s); len(r) > 80 {
		s = string(r[:80]) + "…"
	}
	return s
}
