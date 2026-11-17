package claude

import (
	"strings"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// ApplyContextUsage 解析 get_context_usage 控制应答,写入 usage 快照。
func ApplyContextUsage(body obj, ok bool, usage *core.Usage) (core.Event, bool) {
	cats := arr(body, "categories")
	if !ok || cats == nil {
		if usage != nil && !ok {
			usage.ContextDetailErr = "无法读取分类明细"
		}
		return nil, false
	}
	var list []core.ContextCategory
	for _, c := range cats {
		cm := asObj(c)
		name := str(cm, "name")
		n, has := num(cm, "tokens")
		if name == "" || !has {
			continue
		}
		kind := core.CtxUsed
		switch str(cm, "kind") {
		case "free":
			kind = core.CtxFree
		case "buffer":
			kind = core.CtxBuffer
		case "deferred":
			kind = core.CtxDeferred
		}
		list = append(list, core.ContextCategory{Name: name, Tokens: n, Kind: kind})
	}
	if len(list) == 0 {
		return nil, false
	}
	if usage == nil {
		usage = &core.Usage{}
	}
	if m := u64(body, "maxTokens"); m > 0 {
		usage.ContextSize = m
	}
	if n, has := num(body, "totalTokens"); has {
		usage.ContextUsed = n
	}
	usage.ContextCategories = list
	usage.ContextDetailErr = ""
	usage.ContextEstimated = false
	usage.ContextQueriedAt = time.Now().Unix()
	usage.McpTools = nil
	for _, x := range arr(body, "mcpTools") {
		xm := asObj(x)
		server, full := str(xm, "serverName"), str(xm, "name")
		if server == "" || full == "" {
			continue
		}
		usage.McpTools = append(usage.McpTools, core.McpToolUsage{
			Server: server, Tool: strings.TrimPrefix(full, "mcp__"+server+"__"), Tokens: u64(xm, "tokens"), Loaded: boolOf(xm, "isLoaded", true),
		})
	}
	return core.EvUsage{Usage: *usage}, true
}

// autoContext 自动查询上下文分类明细(get_context_usage,本地控制请求,不耗额度、不碰官方限额)。
// 在会话进程握手完成与每个回合结束时调用;进程没起来或上一次查询还没回来就不重复发。
func (b *Backend) autoContext() {
	if b.p == nil || b.hasReq(rkContext) {
		return
	}
	b.request(req{kind: rkContext}, ContextUsageReq)
}
