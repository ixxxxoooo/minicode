package claude

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ixxxxoooo/minicode/internal/diag"
)

var oneshotLog = diag.L("claude-oneshot")

// RunPromptStreamCtx 同 RunPromptCtx,但边读 stdout 边回调 onChunk(可能从后台 goroutine 调用)。
// 取消 ctx 时会向整个子进程组发 SIGKILL。
func RunPromptStreamCtx(ctx context.Context, cwd, prompt, permMode string, timeout time.Duration, onChunk func(string)) (string, error) {
	return RunPromptModelStreamCtx(ctx, cwd, prompt, permMode, "", timeout, onChunk)
}

// RunPromptModelStreamCtx 指定模型并流式读输出。
func RunPromptModelStreamCtx(ctx context.Context, cwd, prompt, permMode, model string, timeout time.Duration, onChunk func(string)) (string, error) {
	path, ok := FindClaude("")
	if !ok {
		return "", fmt.Errorf("没有找到 claude 命令")
	}
	if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > timeout {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	args := []string{"-p", "--output-format", "text"}
	if permMode != "" {
		args = append(args, "--permission-mode", permMode)
	}
	if model != "" {
		args = append(args, "--model", model)
	}
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Dir = cwd
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	env := make([]string, 0, len(os.Environ()))
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "CLAUDECODE=") {
			env = append(env, e)
		}
	}
	cmd.Env = env
	cmd.Stdin = strings.NewReader(prompt)

	var errb bytes.Buffer
	cmd.Stderr = &errb
	pr, pw, err := os.Pipe()
	if err != nil {
		return "", err
	}
	cmd.Stdout = pw

	var outMu sync.Mutex
	var out strings.Builder
	appendOut := func(p []byte) {
		if len(p) == 0 {
			return
		}
		s := string(p)
		outMu.Lock()
		out.WriteString(s)
		outMu.Unlock()
		if onChunk != nil {
			onChunk(s)
		}
	}

	if err := cmd.Start(); err != nil {
		_ = pw.Close()
		_ = pr.Close()
		return "", err
	}
	_ = pw.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		sc := bufio.NewScanner(pr)
		sc.Buffer(make([]byte, 64*1024), 1024*1024)
		for sc.Scan() {
			appendOut(sc.Bytes())
			appendOut([]byte("\n"))
		}
		_ = pr.Close()
	}()

	killPG := func() {
		if cmd.Process == nil {
			return
		}
		pgid := cmd.Process.Pid
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
	}

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		killPG()
		<-waitDone
		<-done
		return strings.TrimSpace(out.String()), ctx.Err()
	case err := <-waitDone:
		<-done
		if ctx.Err() == context.DeadlineExceeded {
			return strings.TrimSpace(out.String()), fmt.Errorf("运行超时(%v)", timeout)
		}
		if err != nil {
			msg := strings.TrimSpace(errb.String())
			if r := []rune(msg); len(r) > 200 {
				msg = string(r[len(r)-200:])
			}
			if msg == "" {
				msg = err.Error()
			}
			oneshotLog.Warn("claude -p 失败", "err", msg)
			return strings.TrimSpace(out.String()), fmt.Errorf("%s", msg)
		}
		return strings.TrimSpace(out.String()), nil
	}
}
