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

// ShortMCPErr 把 claude mcp list / mcp_status 给的状态文字压成一句人话:
// 去掉 URL 与状态符号,服务端回的整段 JSON 校验错误(以 [ 或 { 开头,多行)换成一句说明,其余折成单行并截断。
func ShortMCPErr(s string) string {
	s = strings.TrimSpace(s)
	// 常见格式:「<地址> (HTTP) - ✘ Failed to connect — <细节>」,只留细节
	if i := strings.Index(s, "—"); i >= 0 {
		s = strings.TrimSpace(s[i+len("—"):])
	} else if i := strings.Index(s, " - "); i >= 0 {
		s = strings.TrimSpace(s[i+3:])
	}
	for _, p := range []string{"✘", "✔", "⊘", "✓"} {
		s = strings.TrimSpace(strings.TrimPrefix(s, p))
	}
	low := strings.ToLower(s)
	switch {
	case strings.HasPrefix(s, "[") || strings.HasPrefix(s, "{") || strings.Contains(low, "invalid_union") || strings.Contains(low, "invalid_type"):
		return "服务端返回的数据格式不符合 MCP 协议"
	case strings.Contains(low, "disabled for this project"):
		return "本项目已停用"
	case strings.Contains(low, "failed to connect"):
		return "无法连接到服务器"
	case strings.Contains(low, "timed out"), strings.Contains(low, "timeout"):
		return "连接超时"
	case strings.Contains(low, "401"), strings.Contains(low, "unauthorized"):
		return "未授权或密钥无效"
	case strings.Contains(low, "econnrefused"), strings.Contains(low, "connection refused"):
		return "连接被拒绝"
	}
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > 60 {
		s = string(r[:60]) + "…"
	}
	return s
}

func parseMCPList(s string) []mcpLine {
	var lines []mcpLine
	for _, raw := range strings.Split(s, "\n") {
		raw = strings.TrimSpace(raw)
		if raw == "" || strings.HasPrefix(raw, "Checking") || strings.HasPrefix(raw, "Installed") {
			continue
		}
		// 名字里可能带冒号(plugin:example-observability:mcp-gateway),分隔符是「冒号 + 空格」
		cut := strings.Index(raw, ": ")
		if cut < 0 {
			continue
		}
		name := strings.TrimSpace(strings.TrimPrefix(raw[:cut], "❯"))
		rest := strings.TrimSpace(raw[cut+2:])
		ln := mcpLine{Name: name, Desc: rest}
		low := strings.ToLower(rest)
		switch {
		case strings.Contains(low, "disabled"), strings.Contains(rest, "⊘"):
			// 「Disabled for this project」是用户主动停用,此前落进 default 被当成连接失败(红点)
			ln.Status = StOff
			ln.Desc = "已停用 · " + ShortMCPErr(rest)
		case strings.Contains(low, "failed"), strings.Contains(rest, "✘"):
			ln.Status = StErr
			ln.Desc = "连接失败 · " + ShortMCPErr(rest)
		case strings.Contains(low, "connected"), strings.Contains(rest, "✔"), strings.Contains(rest, "✓"):
			ln.Status = StOK
			ln.Desc = "已连接"
		case strings.Contains(low, "auth"), strings.Contains(low, "pending"), strings.Contains(low, "approval"):
			ln.Status = StWarn
			ln.Desc = "需要授权"
		default:
			ln.Status = StErr
			ln.Desc = ShortMCPErr(rest)
		}
		lines = append(lines, ln)
	}
	return lines
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
