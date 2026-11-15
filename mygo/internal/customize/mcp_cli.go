package customize

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ixxxxoooo/minicode/internal/claude"
)

type mcpLine struct {
	Name   string
	Status Status
	Desc   string
	Tools  []Tool
}

// RunMCPList 跑 claude mcp list 并解析连接状态。
func RunMCPList(proj string) ([]mcpLine, error) {
	path, ok := claude.FindClaude("")
	if !ok {
		return nil, fmt.Errorf("没有找到 claude 命令")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "mcp", "list")
	if proj != "" {
		cmd.Dir = proj
	}
	cmd.Env = os.Environ()
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	start := time.Now()
	err := cmd.Run()
	alog.Info("mcp list", "ms", time.Since(start).Milliseconds(), "proj", proj, "err", err)
	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("mcp list 超时")
	}
	if err != nil && out.Len() == 0 {
		return nil, fmt.Errorf("mcp list 失败: %v", err)
	}
	return parseMCPList(out.String()), nil
}

func parseMCPList(s string) []mcpLine {
	var lines []mcpLine
	for _, raw := range strings.Split(s, "\n") {
		raw = strings.TrimSpace(raw)
		if raw == "" || strings.HasPrefix(raw, "Checking") || strings.HasPrefix(raw, "Installed") {
			continue
		}
		parts := strings.SplitN(raw, ":", 2)
		if len(parts) != 2 {
			continue
		}
		name := strings.TrimSpace(strings.TrimPrefix(parts[0], "❯"))
		rest := strings.TrimSpace(parts[1])
		ln := mcpLine{Name: name, Desc: rest}
		low := strings.ToLower(rest)
		switch {
		case strings.Contains(low, "connected"), strings.Contains(rest, "✔"), strings.Contains(rest, "✓"):
			ln.Status = StOK
			ln.Desc = "已连接 · " + rest
		case strings.Contains(low, "auth"), strings.Contains(low, "pending"), strings.Contains(low, "approval"):
			ln.Status = StWarn
			ln.Desc = "需要授权 · " + rest
		default:
			ln.Status = StErr
			ln.Desc = rest
		}
		lines = append(lines, ln)
	}
	return lines
}

// RunMCPAdd 调用 claude mcp add(临时 HOME 测试时在环境里设 HOME=…)。
func RunMCPAdd(name, transport, spec, proj string) error {
	path, ok := claude.FindClaude("")
	if !ok {
		return fmt.Errorf("没有找到 claude 命令")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	args := []string{"mcp", "add"}
	if transport == "http" || transport == "sse" {
		args = append(args, "--transport", "http", name, spec)
	} else {
		parts := strings.Fields(spec)
		if len(parts) == 0 {
			return fmt.Errorf("stdio 服务器需要 command")
		}
		args = append(args, name, "--", parts[0])
		args = append(args, parts[1:]...)
	}
	cmd := exec.CommandContext(ctx, path, args...)
	if proj != "" {
		cmd.Dir = proj
	}
	cmd.Env = os.Environ()
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		msg := strings.TrimSpace(out.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s", msg)
	}
	alog.Info("mcp add", "name", name, "proj", proj)
	return nil
}

// RunMCPRemove 调用 claude mcp remove。
func RunMCPRemove(name, proj string) error {
	path, ok := claude.FindClaude("")
	if !ok {
		return fmt.Errorf("没有找到 claude 命令")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "mcp", "remove", name)
	if proj != "" {
		cmd.Dir = proj
	}
	cmd.Env = os.Environ()
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		msg := strings.TrimSpace(out.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s", msg)
	}
	alog.Info("mcp remove", "name", name, "proj", proj)
	return nil
}

func scopeLabel(s string) string {
	switch strings.ToLower(s) {
	case "project":
		return "项目"
	case "local":
		return "本地"
	default:
		return "用户"
	}
}
