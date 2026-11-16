package app

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/customize"
)

const mcpDebounce = 3 * time.Second

// McpRow 是输入框 MCP 弹层与自定义页列表共用的视图行。
type McpRow struct {
	Name      string
	Scope     string // 来源层:用户 / 项目 / 本地 / 插件
	On        bool
	Conn      core.McpConn
	Detail    string
	Tools     int
	Tokens    uint64
	OnDemand  bool
	ToolsList []core.McpToolUsage
	Runtime   bool // 连接状态来自 mcp_status(运行时会话)
}

func mcpConnLabel(c core.McpConn, runtime bool) string {
	if !runtime {
		switch c {
		case core.McpUnknown:
			return "未连接 · 会话未启动,尚未查询"
		default:
			return "未连接 · 尚未查询运行时状态"
		}
	}
	switch c {
	case core.McpConnected:
		return "已连接"
	case core.McpPending:
		return "连接中"
	case core.McpNeedsAuth:
		return "需要授权"
	case core.McpFailed:
		return "连接失败"
	case core.McpDisabled:
		return "已停用"
	case core.McpUnknown:
		return "状态未知"
	}
	return ""
}

func (a *App) mcpRuntimeReady() bool {
	return a.sessProcLive
}

// mcpRowsUnified 合并配置层与 st.Mcp 运行时状态,两界面读同一结果。
func (a *App) mcpRowsUnified() []McpRow {
	st := a.cuStore()
	byName := map[string]McpRow{}
	if st != nil {
		for _, it := range st.List(customize.MCP, customize.All, "") {
			mv := McpRow{
				Name:     it.Name,
				Scope:    it.Scope.Label(),
				On:       it.On,
				OnDemand: it.OnDemand,
				Tools:    len(it.Tools),
				Conn:     mcpConnFromItem(it.Status),
				Detail:   mcpDetailFromItem(it),
			}
			if mv.Detail == "" {
				mv.Detail = mcpConnLabel(mv.Conn, false)
			}
			byName[it.Name] = mv
		}
	}
	runtimeAt := a.mcpRuntimeAt
	for _, s := range a.st.Mcp {
		mv, ok := byName[s.Name]
		if !ok {
			mv = McpRow{Name: s.Name, On: s.Conn != core.McpDisabled, Tools: len(s.Tools)}
		}
		mv.Conn = s.Conn
		mv.Runtime = true
		if s.Error != "" {
			mv.Detail = s.Error
		} else {
			lbl := mcpConnLabel(s.Conn, true)
			if len(s.Tools) > 0 {
				mv.Detail = fmt.Sprintf("%s · %d 个工具", lbl, len(s.Tools))
			} else {
				mv.Detail = lbl
			}
		}
		if len(s.Tools) > 0 {
			mv.Tools = len(s.Tools)
		}
		if runtimeAt > 0 && stringsDetailNeedsTime(mv.Detail) {
			mv.Detail += " · 更新于 " + relTime(a.now(), time.Unix(runtimeAt, 0))
		}
		byName[s.Name] = mv
	}
	for _, tl := range a.st.Usage.McpTools {
		mv := byName[tl.Server]
		mv.Tokens += tl.Tokens
		mv.ToolsList = append(mv.ToolsList, tl)
		byName[tl.Server] = mv
	}
	var names []string
	for n := range byName {
		names = append(names, n)
	}
	sort.Strings(names)
	out := make([]McpRow, 0, len(names))
	for _, n := range names {
		out = append(out, byName[n])
	}
	return out
}

func stringsDetailNeedsTime(detail string) bool {
	if detail == "" {
		return false
	}
	for _, sub := range []string{"更新于", "尚未查询", "未启动"} {
		if strings.Contains(detail, sub) {
			return false
		}
	}
	return true
}

func (a *App) mcpLookup(name string) (McpRow, bool) {
	for _, r := range a.mcpRowsUnified() {
		if r.Name == name {
			return r, true
		}
	}
	return McpRow{}, false
}
