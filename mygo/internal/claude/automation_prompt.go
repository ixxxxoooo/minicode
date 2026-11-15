package claude

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ixxxxoooo/minicode/internal/diag"
)

var autoPromptLog = diag.L("claude-auto")

// AutomationStreamResult 是一次自动化流式运行的输出与会话 id(若有)。
type AutomationStreamResult struct {
	Output    string
	SessionID string
}

// RunAutomationPromptStreamCtx 用 stream-json 跑 `claude -p`,从 system init 取 session_id,并流式回调文本块。
func RunAutomationPromptStreamCtx(ctx context.Context, cwd, prompt, permMode string, timeout time.Duration, onChunk func(string)) (AutomationStreamResult, error) {
	return RunAutomationPromptModelStreamCtx(ctx, cwd, prompt, permMode, "", timeout, onChunk)
}

// RunAutomationPromptModelStreamCtx 指定模型的自动化流式运行。
func RunAutomationPromptModelStreamCtx(ctx context.Context, cwd, prompt, permMode, model string, timeout time.Duration, onChunk func(string)) (AutomationStreamResult, error) {
	var empty AutomationStreamResult
	path, ok := FindClaude("")
	if !ok {
		return empty, fmt.Errorf("没有找到 claude 命令")
	}
	if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > timeout {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	// stream-json 在 -p 下必须带 --verbose,否则 claude 只在 stderr 报一行错、退出码还是 0,
	// 自动化就会「成功」却没有任何输出(真实模式实测发现)
	args := []string{"-p", "--output-format", "stream-json", "--verbose"}
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
		return empty, err
	}
	cmd.Stdout = pw

	var mu sync.Mutex
	var out strings.Builder
	var sessionID string

	appendText := func(s string) {
		if s == "" {
			return
		}
		mu.Lock()
		out.WriteString(s)
		mu.Unlock()
		if onChunk != nil {
			onChunk(s)
		}
	}

	parseLine := func(line string) {
		var v obj
		if json.Unmarshal([]byte(line), &v) != nil {
			// 非 JSON 行当作纯文本
			appendText(line + "\n")
			return
		}
		switch str(v, "type") {
		case "system":
			if str(v, "subtype") == "init" {
				if id := str(v, "session_id"); id != "" {
					sessionID = id
				}
			}
		case "stream_event":
			ev, _ := v["event"].(map[string]any)
			if ev == nil {
				return
			}
			if str(ev, "type") == "content_block_delta" {
				if d, _ := ev["delta"].(map[string]any); d != nil {
					appendText(str(d, "text"))
				}
			}
		case "assistant":
			if msg, _ := v["message"].(map[string]any); msg != nil {
				for _, it := range arr(msg, "content") {
					if o := asObj(it); str(o, "type") == "text" {
						appendText(str(o, "text"))
					}
				}
			}
		case "result":
			// result 事件的文本和前面 assistant 事件里的是同一段,只在一个字都没收到时当兜底,
			// 否则输出会重复两遍
			mu.Lock()
			have := out.Len() > 0
			mu.Unlock()
			if s := str(v, "result"); s != "" && !have {
				appendText(s)
			}
		}
	}

	if err := cmd.Start(); err != nil {
		_ = pw.Close()
		_ = pr.Close()
		return empty, err
	}
	_ = pw.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		defer diag.Recover("claude.automation-scan")
		sc := bufio.NewScanner(pr)
		sc.Buffer(make([]byte, 64*1024), 1024*1024)
		for sc.Scan() {
			parseLine(sc.Text())
		}
		_ = pr.Close()
	}()

	killPG := func() {
		if cmd.Process == nil {
			return
		}
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}

	waitDone := make(chan error, 1)
	go func() {
		defer diag.Recover("claude.automation-wait")
		waitDone <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		killPG()
		<-waitDone
		<-done
		return AutomationStreamResult{Output: strings.TrimSpace(out.String()), SessionID: sessionID}, ctx.Err()
	case err := <-waitDone:
		<-done
		text := strings.TrimSpace(out.String())
		if ctx.Err() == context.DeadlineExceeded {
			return AutomationStreamResult{Output: text, SessionID: sessionID}, fmt.Errorf("运行超时(%v)", timeout)
		}
		if err != nil {
			msg := strings.TrimSpace(errb.String())
			if r := []rune(msg); len(r) > 200 {
				msg = string(r[len(r)-200:])
			}
			if msg == "" {
				msg = err.Error()
			}
			autoPromptLog.Warn("claude 自动化流式失败", "err", msg)
			return AutomationStreamResult{Output: text, SessionID: sessionID}, fmt.Errorf("%s", msg)
		}
		if text == "" && sessionID == "" {
			// 退出码为 0 却什么都没产生:当作失败并带上 stderr 摘要,不再显示「成功」
			msg := strings.TrimSpace(errb.String())
			if r := []rune(msg); len(r) > 200 {
				msg = string(r[len(r)-200:])
			}
			if msg == "" {
				msg = "claude 没有返回任何内容"
			}
			autoPromptLog.Warn("claude 自动化没有输出", "stderr", msg)
			return empty, fmt.Errorf("%s", msg)
		}
		return AutomationStreamResult{Output: text, SessionID: sessionID}, nil
	}
}
