package vcs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ixxxxoooo/minicode/internal/diag"
)

// IgnoreChecker 是可选能力:判断哪些路径被 .gitignore 规则忽略。
// 只有真实 git(CLI)实现;Mem 等没有,调用方要按类型断言并准备兜底。
type IgnoreChecker interface {
	// Ignored 返回 paths(相对仓库目录)里被忽略的那些;已被跟踪的文件不算忽略(与 git 一致)。
	Ignored(paths []string) (map[string]bool, error)
}

// ignoreBatch 是一次 git check-ignore 传的路径数(分批让单次子进程耗时与内存可控)。
const ignoreBatch = 400

// Ignored 用 `git check-ignore` 判定,正确处理嵌套 .gitignore、全局忽略与取反规则。
func (r *CLI) Ignored(paths []string) (map[string]bool, error) {
	out := map[string]bool{}
	if len(paths) == 0 {
		return out, nil
	}
	end := diag.SpanSlow("vcs", "git check-ignore", 1500*time.Millisecond, "paths", len(paths))
	defer end()
	for i := 0; i < len(paths); i += ignoreBatch {
		j := min(i+ignoreBatch, len(paths))
		hit, err := r.checkIgnoreBatch(paths[i:j])
		if err != nil {
			return nil, err
		}
		for _, p := range hit {
			out[p] = true
		}
	}
	return out, nil
}

// checkIgnoreBatch 跑一批。退出码 1 = 一个都没被忽略(正常),其余非零才是错误。
func (r *CLI) checkIgnoreBatch(batch []string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	// -z 只能和 --stdin 一起用:路径以 NUL 分隔从标准输入给,输出同样 NUL 分隔(不被引号转义)
	cmd := exec.CommandContext(ctx, "git", "--no-optional-locks", "check-ignore", "--stdin", "-z")
	cmd.Dir = r.Dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "LC_ALL=C")
	cmd.Stdin = strings.NewReader(strings.Join(batch, "\x00") + "\x00")
	var so, se bytes.Buffer
	cmd.Stdout, cmd.Stderr = &so, &se
	err := cmd.Run()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && ee.ExitCode() == 1 {
			return nil, nil
		}
		msg := strings.TrimSpace(se.String())
		if msg == "" {
			msg = err.Error()
		}
		log.Debug("git check-ignore 失败", "err", msg)
		return nil, fmt.Errorf("%s", msg)
	}
	var hit []string
	for _, p := range strings.Split(so.String(), "\x00") {
		if p != "" {
			hit = append(hit, p)
		}
	}
	return hit, nil
}
