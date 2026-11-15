package vcs

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ixxxxoooo/minicode/internal/diag"
)

var log = diag.L("vcs")

// CLI 用 git 命令行实现 Repo。Dir 是仓库(或其子目录)路径。
type CLI struct{ Dir string }

// Open 返回 dir 所在仓库的 CLI;dir 不在仓库里时返回错误。
func Open(dir string) (*CLI, error) {
	r := &CLI{Dir: dir}
	if _, err := r.git(3*time.Second, "rev-parse", "--git-dir"); err != nil {
		return nil, fmt.Errorf("不是 Git 仓库:%w", err)
	}
	return r, nil
}

// git 运行一条 git 命令。读命令一律带 --no-optional-locks(否则 git status 会改写 index,
// 触发文件监视形成反馈环),且禁止交互式认证提示。
func (r *CLI) git(timeout time.Duration, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"--no-optional-locks"}, args...)...)
	cmd.Dir = r.Dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "LC_ALL=C")
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	end := diag.SpanSlow("vcs", "git "+args[0], 300*time.Millisecond)
	err := cmd.Run()
	end()
	if err != nil {
		msg := strings.TrimSpace(errb.String())
		if msg == "" {
			msg = err.Error()
		}
		log.Debug("git 失败", "args", args, "err", msg)
		return "", fmt.Errorf("%s", msg)
	}
	return strings.TrimRight(out.String(), "\n"), nil
}

func (r *CLI) Current() string {
	if s, err := r.git(3*time.Second, "symbolic-ref", "--short", "-q", "HEAD"); err == nil && s != "" {
		return s
	}
	if s, err := r.git(3*time.Second, "rev-parse", "--short", "HEAD"); err == nil {
		return s
	}
	return ""
}

func (r *CLI) Branches() ([]Branch, error) {
	out, err := r.git(5*time.Second, "for-each-ref", "--sort=-committerdate",
		"--format=%(refname)\t%(HEAD)\t%(committerdate:relative)", "refs/heads", "refs/remotes")
	if err != nil {
		return nil, err
	}
	return parseBranches(out), nil
}

// parseBranches 解析 for-each-ref 的输出:本地在前、远程在后;远程里有同名本地的略去,忽略 HEAD 指针。
func parseBranches(out string) []Branch {
	var local, remote []Branch
	have := map[string]bool{}
	for _, ln := range strings.Split(out, "\n") {
		f := strings.Split(ln, "\t")
		if len(f) < 3 {
			continue
		}
		when := zhRelative(f[2])
		switch {
		case strings.HasPrefix(f[0], "refs/heads/"):
			n := strings.TrimPrefix(f[0], "refs/heads/")
			have[n] = true
			local = append(local, Branch{Name: n, Current: f[1] == "*", When: when})
		case strings.HasPrefix(f[0], "refs/remotes/"):
			n := strings.TrimPrefix(f[0], "refs/remotes/")
			if strings.HasSuffix(n, "/HEAD") {
				continue
			}
			remote = append(remote, Branch{Name: n, Remote: true, When: when})
		}
	}
	var rest []Branch
	for _, b := range remote {
		if i := strings.Index(b.Name, "/"); i >= 0 && have[b.Name[i+1:]] {
			continue
		}
		rest = append(rest, b)
	}
	return append(local, rest...)
}

// zhRelative 把 git 的英文相对时间译成中文:「2 hours ago」→「2 小时前」。
func zhRelative(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var n int
	var unit string
	if _, err := fmt.Sscanf(s, "%d %s", &n, &unit); err != nil {
		return s
	}
	unit = strings.TrimSuffix(unit, "s") // seconds → second
	zh := map[string]string{"second": "秒", "minute": "分钟", "hour": "小时", "day": "天", "week": "周", "month": "个月", "year": "年"}[unit]
	if zh == "" {
		return s
	}
	if zh == "秒" && n < 30 {
		return "刚刚"
	}
	return fmt.Sprintf("%d %s前", n, zh)
}

func (r *CLI) Switch(name string) error {
	if i := strings.Index(name, "/"); i > 0 && r.isRemote(name) {
		_, err := r.git(15*time.Second, "switch", "--track", name)
		return err
	}
	_, err := r.git(15*time.Second, "switch", name)
	return err
}

func (r *CLI) isRemote(name string) bool {
	_, err := r.git(3*time.Second, "show-ref", "--verify", "--quiet", "refs/remotes/"+name)
	return err == nil
}

func (r *CLI) Create(name string) error {
	if !ValidName(name) {
		return ErrInvalidName
	}
	_, err := r.git(15*time.Second, "switch", "-c", name)
	return err
}
