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
	return runPromptArgs(ctx, cwd, prompt, permMode, model, timeout, nil)
}

// RunPromptLeanCtx 同 RunPromptModelCtx,但是「纯文字问答」的精简调用:不给任何内置工具、不加载技能 / 斜杠命令、不连 MCP 服务器,
// 每次只带很小的上下文(下一步建议这类每轮都要跑的小活用它,费用与耗时都比完整调用低一个量级)。
// system 替换默认系统提示词(默认那份约 2k token,每次都要付)。实测:haiku 一次约 3 秒、约 1 美分。
func RunPromptLeanCtx(ctx context.Context, cwd, system, prompt, model string, timeout time.Duration) (string, error) {
	return runPromptArgs(ctx, cwd, prompt, "plan", model, timeout, []string{"--tools", "", "--disable-slash-commands", "--strict-mcp-config", "--system-prompt", system})
}

func runPromptArgs(ctx context.Context, cwd, prompt, permMode, model string, timeout time.Duration, extra []string) (string, error) {
	path, ok := FindClaude("")
	if !ok {
		return "", fmt.Errorf("没有找到 claude 命令")
	}
	if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > timeout {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	// 一次性调用(提交说明 / 行内补全 / 画布与自定义生成)不留会话:否则每次都会在
	// ~/.claude/projects 里多出一个「你是 Git 提交说明助手…」之类的会话,污染用户的历史列表
	args := []string{"-p", "--output-format", "text", "--no-session-persistence"}
	if permMode != "" {
		args = append(args, "--permission-mode", permMode)
	}
	if model != "" {
		args = append(args, "--model", model)
	}
	args = append(args, extra...)
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
