package demo

import (
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// demoMcpServers 演示用 MCP 状态(覆盖连接态与工具数量)。
func demoMcpServers() []core.McpServerStatus {
	many := make([]string, 24)
	for i := range many {
		many[i] = fmtTool(i)
	}
	return []core.McpServerStatus{
		{Name: "github", Conn: core.McpConnected, Tools: []string{"create_issue", "list_pull_requests", "search_code", "get_file_contents"}},
		{Name: "filesystem", Conn: core.McpConnected, Tools: []string{"read_file", "write_file", "list_directory", "search_files"}},
		{Name: "toolbox", Conn: core.McpConnected, Tools: many},
		{Name: "empty-svc", Conn: core.McpConnected, Tools: nil},
		{Name: "figma", Conn: core.McpNeedsAuth, Error: "需要授权:在浏览器中完成 OAuth", Tools: nil},
		{Name: "postgres", Conn: core.McpFailed, Error: "连接失败:ECONNREFUSED 127.0.0.1:5432"},
		{Name: "sentry", Conn: core.McpDisabled, Error: "已在设置中停用"},
		{Name: "notion", Conn: core.McpPending, Error: "正在连接…"},
	}
}

func fmtTool(i int) string {
	return "tool_" + string(rune('a'+i%26)) + string(rune('0'+i/26%10))
}

func demoContextUsage() core.Usage {
	loaded := []core.McpToolUsage{
		{Server: "github", Tool: "search_code", Tokens: 1200, Loaded: true},
		{Server: "github", Tool: "create_issue", Tokens: 900, Loaded: true},
		{Server: "filesystem", Tool: "read_file", Tokens: 600, Loaded: true},
		{Server: "toolbox", Tool: "tool_a0", Tokens: 400, Loaded: false},
		{Server: "toolbox", Tool: "tool_b1", Tokens: 380, Loaded: false},
		{Server: "figma", Tool: "get_design", Tokens: 200, Loaded: false},
	}
	return core.Usage{
		ContextUsed:    88_000,
		ContextSize:    200_000,
		FiveHourPct:    ptr32(42),
		WeeklyPct:      ptr32(24),
		FiveHourResets: time.Now().Add(2*time.Hour + 10*time.Minute).Unix(),
		WeeklyResets:   time.Now().Add(5 * 24 * time.Hour).Unix(),
		LimitsAt:       time.Now().Add(-3 * time.Minute).Unix(),
		LimitsState:    core.LimitsIdle,
		ContextCategories: []core.ContextCategory{
			{Name: "系统提示", Tokens: 28_000, Kind: core.CtxUsed},
			{Name: "消息", Tokens: 24_000, Kind: core.CtxUsed},
			{Name: "工具定义", Tokens: 18_000, Kind: core.CtxUsed},
			{Name: "MCP 工具", Tokens: 10_000, Kind: core.CtxUsed},
			{Name: "MCP 工具(按需)", Tokens: 3200, Kind: core.CtxDeferred},
			{Name: "空闲", Tokens: 106_800, Kind: core.CtxFree},
		},
		McpTools: loaded,
	}
}

func ptr32(f float32) *float32 { return &f }
