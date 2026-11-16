package claude

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// FetchMcpList 无活动会话时用 claude mcp list 取 MCP 连接状态(与 customize 解析一致,避免 import 环)。
func FetchMcpList(proj string) ([]core.McpServerStatus, error) {
	path, ok := FindClaude("")
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
	if err := cmd.Run(); err != nil && out.Len() == 0 {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("mcp list 超时")
		}
		return nil, fmt.Errorf("mcp list 失败: %v", err)
	}
	return parseMcpListLines(out.String()), nil
}

func parseMcpListLines(s string) []core.McpServerStatus {
	var out []core.McpServerStatus
	for _, raw := range strings.Split(s, "\n") {
		raw = strings.TrimSpace(raw)
		if raw == "" || strings.HasPrefix(raw, "Checking") || strings.HasPrefix(raw, "Installed") {
			continue
		}
		// 名字里可能带冒号(plugin:xxx:yyy),分隔符是「冒号 + 空格」
		cut := strings.Index(raw, ": ")
		if cut < 0 {
			continue
		}
		name := strings.TrimSpace(strings.TrimPrefix(raw[:cut], "❯"))
		rest := strings.TrimSpace(raw[cut+2:])
		st := core.McpServerStatus{Name: name}
		low := strings.ToLower(rest)
		switch {
		case strings.Contains(low, "disabled"), strings.Contains(rest, "⊘"):
			st.Conn = core.McpDisabled
		case strings.Contains(low, "failed"), strings.Contains(rest, "✘"):
			st.Conn = core.McpFailed
			st.Error = rest
		case strings.Contains(low, "connected"), strings.Contains(rest, "✔"), strings.Contains(rest, "✓"):
			st.Conn = core.McpConnected
		case strings.Contains(low, "auth"), strings.Contains(low, "pending"), strings.Contains(low, "approval"):
			st.Conn = core.McpNeedsAuth
		default:
			st.Conn = core.McpFailed
			st.Error = rest
		}
		out = append(out, st)
	}
	return out
}
