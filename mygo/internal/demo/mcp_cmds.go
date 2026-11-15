package demo

import (
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// demoMcpServers 演示用 MCP 状态(与自定义页 demo 数据对齐)。
func demoMcpServers() []core.McpServerStatus {
	return []core.McpServerStatus{
		{Name: "github", Conn: core.McpConnected, Tools: []string{"create_issue", "list_pull_requests", "search_code"}},
		{Name: "filesystem", Conn: core.McpConnected, Tools: []string{"read_file", "write_file", "list_directory"}},
		{Name: "figma", Conn: core.McpNeedsAuth, Error: "需要授权", Tools: nil},
		{Name: "postgres", Conn: core.McpFailed, Error: "连接失败:ECONNREFUSED"},
	}
}

func demoContextUsage() core.Usage {
	return core.Usage{
		ContextUsed:    76_000,
		ContextSize:    200_000,
		FiveHourPct:    ptr32(38),
		WeeklyPct:      ptr32(21),
		FiveHourResets: time.Now().Add(2*time.Hour + 10*time.Minute).Unix(),
		WeeklyResets:   time.Now().Add(5 * 24 * time.Hour).Unix(),
		LimitsAt:       time.Now().Add(-3 * time.Minute).Unix(),
		LimitsState:    core.LimitsIdle,
		ContextCategories: []core.ContextCategory{
			{Name: "系统提示", Tokens: 28_000, Kind: core.CtxUsed},
			{Name: "消息", Tokens: 22_000, Kind: core.CtxUsed},
			{Name: "工具定义", Tokens: 18_000, Kind: core.CtxUsed},
			{Name: "MCP 工具", Tokens: 8_000, Kind: core.CtxUsed},
			{Name: "空闲", Tokens: 124_000, Kind: core.CtxFree},
		},
		McpTools: []core.McpToolUsage{
			{Server: "github", Tool: "search_code", Tokens: 1200, Loaded: true},
			{Server: "github", Tool: "create_issue", Tokens: 900, Loaded: false},
		},
	}
}

func ptr32(f float32) *float32 { return &f }
