package claude

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// RunPrompt 用 `claude -p` 在目录 cwd 里无界面地跑一条提示(自动化的运行器用它)。
// permMode 是 --permission-mode 的值(plan / acceptEdits / default…),空 = 默认。
// 返回标准输出(去掉首尾空白);超时或进程失败返回错误,错误里带 stderr 末尾。
func RunPrompt(cwd, prompt, permMode string, timeout time.Duration) (string, error) {
	return RunPromptModel(cwd, prompt, permMode, "", timeout)
}

// RunPromptCtx 同 RunPrompt,但可取消(自动化「停止」用)。
func RunPromptCtx(ctx context.Context, cwd, prompt, permMode string, timeout time.Duration) (string, error) {
	return RunPromptModelCtx(ctx, cwd, prompt, permMode, "", timeout)
}

// RunPromptModel 同 RunPrompt,另指定模型(如 haiku;空 = 默认)。提交说明这类小活用小模型。
func RunPromptModel(cwd, prompt, permMode, model string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return RunPromptModelCtx(ctx, cwd, prompt, permMode, model, timeout)
}

func RunPromptModelCtx(ctx context.Context, cwd, prompt, permMode, model string, timeout time.Duration) (string, error) {
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
	env := make([]string, 0, len(os.Environ()))
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "CLAUDECODE=") {
			env = append(env, e)
		}
	}
	cmd.Env = env
	cmd.Stdin = strings.NewReader(prompt)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
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
		return strings.TrimSpace(out.String()), fmt.Errorf("%s", msg)
	}
	return strings.TrimSpace(out.String()), nil
}

// PermModeOf 把自动化里的权限说明(如「只读 · 计划模式」)换成 --permission-mode 的值。
func PermModeOf(perm string) string {
	switch {
	case strings.Contains(perm, "计划"), strings.Contains(perm, "只读"):
		return "plan"
	case strings.Contains(perm, "编辑"):
		return "acceptEdits"
	}
	return ""
}
