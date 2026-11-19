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
	"sync"
	"syscall"
	"time"

	"github.com/ixxxxoooo/minicode/internal/diag"
)

// 语言包(npm 系的语言服务器)托管:装到 <数据目录>/lsp/<组>/node_modules/.bin/…,版本固定(Spec.NPM)。
// 一个 npm 包提供多个服务器的(vscode-langservers-extracted:HTML / CSS / JSON)共用一个目录(Spec.NPMGroup)。
// 不自己重新托管包(Intelephense 是专有许可,不能再分发),一律从 npm 源装;安装用 --ignore-scripts,不跑包里的安装脚本。

// pkgsMarker 记录装进这个目录的包与版本,用来判断「可更新」(清单里的固定版本变了)。
const pkgsMarker = ".minicode-pkgs"

func lspRoot() string { return filepath.Join(dataRoot(), "lsp") }

// Managed 报告这个服务器能不能由应用托管安装(即清单里固定了 npm 包)。
func (s *Spec) Managed() bool { return len(s.NPM) > 0 }

// ManagedPrefix 是托管安装目录。
func (s *Spec) ManagedPrefix() string {
	g := s.NPMGroup
	if g == "" {
		g = s.ID
	}
	return filepath.Join(lspRoot(), g)
}

// IsManagedPath 判断一个可执行文件是不是托管目录里的。
func IsManagedPath(p string) bool { return strings.HasPrefix(p, lspRoot()+string(filepath.Separator)) }

// ManagedBin 返回托管目录里已装好的可执行文件。
func (s *Spec) ManagedBin() (string, bool) {
	if !s.Managed() {
		return "", false
	}
	for _, b := range s.Bins {
		p := filepath.Join(s.ManagedPrefix(), "node_modules", ".bin", b)
		if isExec(p) {
			return p, true
		}
	}
	return "", false
}

// ManagedStale 报告托管目录里装的包与清单里固定的不一致(有新版本可更新)。没装过返回 false。
func (s *Spec) ManagedStale() bool {
	if _, ok := s.ManagedBin(); !ok {
		return false
	}
	b, err := os.ReadFile(filepath.Join(s.ManagedPrefix(), pkgsMarker))
	return err != nil || strings.TrimSpace(string(b)) != strings.Join(s.NPM, "\n")
}

// UninstallManaged 删掉托管目录(Node 运行时保留,别的服务器还要用)。共用目录的服务器会一起没掉。
func (s *Spec) UninstallManaged() error {
	if !s.Managed() {
		return nil
	}
	rtLog.Info("卸载托管语言包", "id", s.ID, "dir", s.ManagedPrefix())
	return os.RemoveAll(s.ManagedPrefix())
}

// npmEnv 是跑 npm 的环境:PATH 前置所用 Node 的目录(npm 是 #!/usr/bin/env node 脚本),
// 指定下载源与缓存目录,关掉更新提示。代理等其余环境变量沿用本进程的。
func npmEnv(node NodeInfo, m Mirror) []string {
	var out []string
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "PATH=") || strings.HasPrefix(kv, "npm_config_") || strings.HasPrefix(kv, "NPM_CONFIG_") {
			continue
		}
		out = append(out, kv)
	}
	return append(out,
		"PATH="+node.Dir+":/usr/bin:/bin:/usr/sbin:/sbin",
		"npm_config_registry="+m.Registry(),
		"npm_config_cache="+filepath.Join(dataRoot(), "cache", "npm"),
		"npm_config_update_notifier=false",
		"npm_config_fund=false",
		"npm_config_audit=false",
	)
}

// installMu 保证同一时刻只有一个托管安装在写目录(界面侧也只排一个任务,这里是兜底)。
var installMu sync.Mutex

// InstallManaged 下载安装这个服务器:先选 Node(本机优先,没有合适的就下载托管的),再用它的 npm 装固定版本的包。
// 先装到临时目录,验证可执行文件都在才原子改名,失败不动旧版本。logf 收 npm 的输出(进「输出 → 语言服务」)。
func (s *Spec) InstallManaged(ctx context.Context, m Mirror, prog Progress, logf func(string)) error {
	if !s.Managed() {
		return fmt.Errorf("%s 不支持下载安装", s.Name)
	}
	installMu.Lock()
	defer installMu.Unlock()
	defer diag.SpanSlow("lsp", "下载安装语言包", 120*time.Second, "server", s.Name, "mirror", string(m))()

	node, ok := ResolveNode(s.MinNode, true)
	if !ok {
		if err := InstallManagedNode(ctx, m, prog); err != nil {
			return err
		}
		if node, ok = ResolveNode(s.MinNode, true); !ok {
			return fmt.Errorf("托管 Node 下载后仍不可用(要求 Node ≥ %s)", s.MinNode)
		}
	}
	rtLog.Info("安装语言包", "server", s.Name, "pkgs", strings.Join(s.NPM, " "), "node", node.Version, "managed", node.Managed, "mirror", string(m))
	if err := ctx.Err(); err != nil {
		return err
	}
	prog("安装 "+strings.Join(s.NPM, " "), 0, 0)

	if err := os.MkdirAll(lspRoot(), 0o755); err != nil {
		return err
	}
	stage, err := os.MkdirTemp(lspRoot(), ".install-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)

	args := append([]string{"install", "--prefix", stage, "--no-package-lock", "--ignore-scripts", "--no-fund", "--no-audit", "--no-update-notifier", "--loglevel=error"}, s.NPM...)
	cmd := exec.CommandContext(ctx, filepath.Join(node.Dir, "npm"), args...)
	cmd.Dir = stage
	cmd.Env = npmEnv(node, m)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) } // 取消时连 npm 的子进程一起杀
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
		rtLog.Warn("npm 安装失败", "server", s.Name, "err", werr, "tail", why, "耗时", time.Since(t0))
		return fmt.Errorf("npm 安装失败:%v(%s)", werr, why)
	}

	for _, b := range s.Bins[:1] { // 至少第一个可执行文件要在
		if !isExec(filepath.Join(stage, "node_modules", ".bin", b)) {
			return fmt.Errorf("安装完成但没有找到 %s(包结构变了?)", b)
		}
	}
	if err := os.WriteFile(filepath.Join(stage, pkgsMarker), []byte(strings.Join(s.NPM, "\n")+"\n"), 0o644); err != nil {
		return err
	}
	dst := s.ManagedPrefix()
	if err := os.RemoveAll(dst); err != nil {
		return err
	}
	if err := os.Rename(stage, dst); err != nil {
		return err
	}
	rtLog.Info("语言包已安装", "server", s.Name, "dir", dst, "耗时", time.Since(t0))
	return nil
}
