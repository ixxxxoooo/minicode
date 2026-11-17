package app

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/ixxxxoooo/minicode/internal/diag"
)

// @author ixxxxoooo

// 一键安装字体:macOS 上字体都有 Homebrew cask,点「安装」就在后台跑 brew install --cask,
// 进度和失败原因直接显示在那一行(不用开终端);装完立刻重新检测、预览随之更新。

// fontCasks 是字体族 → Homebrew cask 的对应(只列确认过存在的)。键是 normFont 之后的族名。
var fontCasks = map[string]string{
	normFont("SF Mono"):                 "font-sf-mono",
	normFont("JetBrains Mono"):          "font-jetbrains-mono",
	normFont("Fira Code"):               "font-fira-code",
	normFont("Cascadia Code"):           "font-cascadia-code",
	normFont("Source Code Pro"):         "font-source-code-pro",
	normFont("IBM Plex Mono"):           "font-ibm-plex-mono",
	normFont("Hack"):                    "font-hack",
	normFont("MesloLGS NF"):             "font-meslo-for-powerlevel10k",
	normFont("MesloLGS Nerd Font"):      "font-meslo-lg-nerd-font",
	normFont("JetBrainsMono Nerd Font"): "font-jetbrains-mono-nerd-font",
	normFont("FiraCode Nerd Font"):      "font-fira-code-nerd-font",
	normFont("Hack Nerd Font"):          "font-hack-nerd-font",
	normFont("CaskaydiaCove Nerd Font"): "font-caskaydia-cove-nerd-font",
	normFont("Symbols Nerd Font"):       "font-symbols-only-nerd-font",
	normFont("Sarasa Mono SC"):          "font-sarasa-gothic",
}

// fontCaskFor 返回字体族对应的 cask;没有对应(系统自带字体或不认识的字体)返回 "", false。
func fontCaskFor(family string) (string, bool) {
	c, ok := fontCasks[normFont(family)]
	return c, ok
}

// fontInstall 是一个字体的安装状态。
type fontInstall struct {
	running bool
	failed  string // 失败原因(stderr 摘要);空 = 没失败
	at      time.Time
}

var brewMu sync.Mutex // 同一时间只跑一个 brew(它自己有锁,并发会互相等)

// brewBin 找 brew 的可执行文件;GUI 应用的 PATH 里没有 /opt/homebrew/bin,所以按固定位置找。
func brewBin() string {
	for _, p := range []string{"/opt/homebrew/bin/brew", "/usr/local/bin/brew"} {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p
		}
	}
	return ""
}

// runBrewCask 执行 brew install --cask,返回合并后的输出与错误。skipUpdate 为真时不自动更新索引(快很多)。
func runBrewCask(brew, cask string, skipUpdate bool) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, brew, "install", "--cask", cask)
	env := append(os.Environ(), "HOMEBREW_NO_INSTALL_CLEANUP=1", "HOMEBREW_NO_ENV_HINTS=1", "HOMEBREW_NO_ANALYTICS=1")
	if skipUpdate {
		env = append(env, "HOMEBREW_NO_AUTO_UPDATE=1")
	}
	cmd.Env = env
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	return out.String(), err
}

// tailLines 取输出末尾几行作失败摘要,界面只放得下一两行。
func tailLines(s string, n int) string {
	ls := strings.Split(strings.TrimSpace(s), "\n")
	if len(ls) > n {
		ls = ls[len(ls)-n:]
	}
	return strings.TrimSpace(strings.Join(ls, " "))
}

// installFont 后台安装字体族 family 对应的 cask。
// 已在安装中、没有对应 cask、本机没有 Homebrew 时给出原因而不执行。
func (a *App) installFont(family string) {
	cask, ok := fontCaskFor(family)
	if !ok {
		return
	}
	if a.set.fontInst == nil {
		a.set.fontInst = map[string]*fontInstall{}
	}
	st := a.set.fontInst[family]
	if st == nil {
		st = &fontInstall{}
		a.set.fontInst[family] = st
	}
	if st.running {
		return
	}
	brew := brewBin()
	if brew == "" {
		st.failed = "没有找到 Homebrew,请先安装它(brew.sh),或手动下载字体"
		log.Warn("安装字体:没有 Homebrew", "family", family)
		return
	}
	st.running, st.failed, st.at = true, "", time.Now()
	log.Info("安装字体:开始", "family", family, "cask", cask)
	a.bg("install-font", func() func() {
		end := diag.SpanSlow("font", "brew 安装字体", 60*time.Second, "cask", cask)
		start := time.Now()
		brewMu.Lock()
		out, err := runBrewCask(brew, cask, true)
		if err != nil && strings.Contains(out, "No Cask") { // 本地索引没有这个 cask:允许它更新一次再试
			out, err = runBrewCask(brew, cask, false)
		}
		brewMu.Unlock()
		end()
		return func() {
			st.running = false
			if err != nil {
				st.failed = tailLines(out, 2)
				if st.failed == "" {
					st.failed = err.Error()
				}
				log.Warn("安装字体:失败", "family", family, "cask", cask, "耗时", time.Since(start), "err", err, "输出", tailLines(out, 4))
				return
			}
			log.Info("安装字体:完成", "family", family, "cask", cask, "耗时", time.Since(start))
			resetFontScans()
			a.startFontScan()
			a.applyPrefs() // 终端用的字体链按新装的字体重新套用
			a.flash("已安装字体 " + family)
		}
	})
}
