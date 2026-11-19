package lsp

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/ixxxxoooo/minicode/internal/diag"
)

// 不依赖 Node 的语言服务器(gopls、rust-analyzer、clangd、zls、taplo、texlab…)的「下载安装」:
// 由应用在后台替用户执行清单里的安装命令(go install / brew install / rustup / gem / dotnet / ghcup),
// 不开终端;输出逐行写进日志、取消时连子进程一起杀;命令跑完再用 Find 确认真的装上了。
// 应用不替用户装工具链本身(Go / Homebrew / rustup…):缺了就直接说明缺什么、去哪装。
//
// @author ixxxxoooo

// toolHints 是缺少某个工具链时给用户的下一步(键 = 安装命令的第一个词)。
var toolHints = map[string]string{
	"go":           "没有找到 Go 工具链,请先安装 Go(https://go.dev/dl)",
	"brew":         "没有找到 Homebrew,请先安装(https://brew.sh)",
	"rustup":       "没有找到 rustup,请先安装 Rust(https://rustup.rs)",
	"gem":          "没有找到 RubyGems,请先安装 Ruby",
	"dotnet":       "没有找到 .NET SDK,请先安装(https://dotnet.microsoft.com/download)",
	"ghcup":        "没有找到 ghcup,请先安装(https://www.haskell.org/ghcup)",
	"xcode-select": "没有找到 xcode-select",
}

// CmdInstallable 报告这个服务器能不能由应用后台执行安装命令(不依赖 Node、清单里有安装命令)。
func (s *Spec) CmdInstallable() bool { return !s.Managed() && strings.TrimSpace(s.Install) != "" }

// Installable 报告这个服务器能不能由应用「下载安装」(托管 npm 包,或后台执行安装命令)。
func (s *Spec) Installable() bool { return s.Managed() || s.CmdInstallable() }

// installTool 是安装命令的第一个词(要用的工具链)。
func (s *Spec) installTool() string {
	f := strings.Fields(s.Install)
	if len(f) == 0 {
		return ""
	}
	return f[0]
}

// cmdInstallEnv 是跑安装命令的环境:PATH 补上常见工具目录(从访达启动的应用只有 launchd 的 PATH,看不到 brew / go / rustup),
// 关掉交互提示,国内镜像时指向镜像。其余变量(代理等)沿用本进程的。
func cmdInstallEnv(m Mirror) []string {
	dirs := searchDirs()
	if home, _ := os.UserHomeDir(); home != "" {
		dirs = append(dirs, filepath.Join(home, ".ghcup", "bin"), "/usr/local/share/dotnet")
	}
	seen := map[string]bool{}
	var path []string
	for _, d := range dirs {
		if d != "" && !seen[d] {
			seen[d] = true
			path = append(path, d)
		}
	}
	env := []string{}
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "PATH=") {
			env = append(env, kv)
		}
	}
	env = append(env, "PATH="+strings.Join(path, ":"),
		"NONINTERACTIVE=1", "HOMEBREW_NO_AUTO_UPDATE=1", "HOMEBREW_NO_ENV_HINTS=1", "HOMEBREW_NO_INSTALL_CLEANUP=1")
	if m == MirrorCN {
		env = append(env, "GOPROXY=https://goproxy.cn,direct",
			"HOMEBREW_BOTTLE_DOMAIN=https://mirrors.ustc.edu.cn/homebrew-bottles",
			"RUSTUP_DIST_SERVER=https://mirrors.ustc.edu.cn/rust-static", "RUSTUP_UPDATE_ROOT=https://mirrors.ustc.edu.cn/rust-static/rustup")
	}
	return env
}

// InstallByCommand 后台执行清单里的安装命令。
// 参数:ctx 取消时杀掉整个进程组;m 下载源;prog 阶段回调(没有可量的进度,只报阶段文字);logf 命令输出逐行回调。
// 返回:nil = 命令成功且装好的可执行文件已能找到;否则是写给用户看的原因。
func (s *Spec) InstallByCommand(ctx context.Context, m Mirror, prog Progress, logf func(string)) error {
	if !s.CmdInstallable() {
		return fmt.Errorf("%s 不支持下载安装", s.Name)
	}
	installMu.Lock()
	defer installMu.Unlock()
	defer diag.SpanSlow("lsp", "执行语言服务器安装命令", 180*time.Second, "server", s.Name)()

	env := cmdInstallEnv(m)
	tool := s.installTool()
	if err := lookInEnv(tool, env); err != nil {
		if h := toolHints[tool]; h != "" {
			return fmt.Errorf("%s", h)
		}
		return fmt.Errorf("没有找到 %s,请先安装它", tool)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	prog("执行 "+s.Install, 0, 0)
	rtLog.Info("执行安装命令", "server", s.Name, "cmd", s.Install, "mirror", string(m))

	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", s.Install)
	cmd.Env = env
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) } // 取消时连子进程一起杀
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = cmd.Stdout
	t0 := time.Now()
	if err := cmd.Start(); err != nil {
		return err
	}
	var tail []string // 输出最后几行,失败时拼进错误
	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 64<<10), 1<<20)
	for sc.Scan() {
		ln := strings.TrimSpace(sc.Text())
		if ln == "" {
			continue
		}
		if logf != nil {
			logf(ln)
		}
		if tail = append(tail, ln); len(tail) > 4 {
			tail = tail[1:]
		}
	}
	_, _ = io.Copy(io.Discard, stdout)
	werr := cmd.Wait()
	if cerr := ctx.Err(); cerr != nil {
		return cerr
	}
	if werr != nil {
		why := strings.Join(tail, " | ")
		if r := []rune(why); len(r) > 400 {
			why = string(r[:400]) + "…"
		}
		rtLog.Warn("安装命令失败", "server", s.Name, "cmd", s.Install, "err", werr, "tail", why, "耗时", time.Since(t0))
		return fmt.Errorf("安装命令失败:%v(%s)", werr, why)
	}
	rtLog.Info("安装命令已执行完", "server", s.Name, "耗时", time.Since(t0))
	if tool == "xcode-select" { // 只是弹出系统的安装窗口,要等用户在窗口里装完
		return fmt.Errorf("已弹出系统的「命令行开发者工具」安装窗口,装完后点「重新检测」")
	}
	if _, ok := s.Find(); !ok {
		return fmt.Errorf("安装命令已执行完,但在常见目录和 PATH 里没找到 %s,可点「重新检测」或查看日志", s.Bins[0])
	}
	return nil
}

// lookInEnv 在给定环境的 PATH 里找可执行文件 name。
func lookInEnv(name string, env []string) error {
	for _, kv := range env {
		if v, ok := strings.CutPrefix(kv, "PATH="); ok {
			for _, d := range filepath.SplitList(v) {
				if isExec(filepath.Join(d, name)) {
					return nil
				}
			}
		}
	}
	return fmt.Errorf("not found: %s", name)
}
