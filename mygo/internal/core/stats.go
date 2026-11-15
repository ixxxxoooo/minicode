package core

import (
	"fmt"
	"strings"
)

// LimitsTTLSecs 是账户限额数据的缓存时长(秒):限额按小时 / 周重置,变化很慢,
// 点开面板时数据还新就不再请求;想要最新的,点面板里的「刷新」。
const LimitsTTLSecs = 300

// McpConn 是 MCP 服务器的连接状态(取自 claude system init 的 mcp_servers)。
type McpConn int

const (
	McpUnknown   McpConn = iota // 还没有信息(会话尚未启动)或无法识别
	McpConnected                // 已连接
	McpPending                  // 正在连接
	McpFailed                   // 连接失败
	McpNeedsAuth                // 需要授权
	McpDisabled                 // 已被停用
)

// ParseMcpConn 把 claude 报的状态文字归类。
func ParseMcpConn(s string) McpConn {
	l := strings.ToLower(s)
	switch {
	case strings.Contains(l, "disabled"):
		return McpDisabled
	case strings.Contains(l, "auth"):
		return McpNeedsAuth
	case strings.Contains(l, "fail"), strings.Contains(l, "error"):
		return McpFailed
	case strings.Contains(l, "pending"), strings.Contains(l, "connecting"):
		return McpPending
	case strings.Contains(l, "connect"):
		return McpConnected
	}
	return McpUnknown
}

// McpServerStatus 是一个 MCP 服务器在当前会话里的实际情况。
type McpServerStatus struct {
	Name  string
	Conn  McpConn
	Tools []string // 不带 mcp__<名称>__ 前缀
	Error string   // 连接失败时的原因(第一行,已截断)
}

// McpNormName 是 MCP 服务器名在工具名里的写法:字母数字 _ - 之外的字符换成 _
// (如 `claude.ai Claude Docs` → `claude_ai_Claude_Docs`),工具全名是 mcp__<写法>__<工具>。
func McpNormName(name string) string {
	var b strings.Builder
	for _, r := range name {
		if r < 128 && (r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-') {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return b.String()
}

// McpToolUsage 是一个 MCP 工具在上下文里的占用(get_context_usage 的 mcpTools)。
type McpToolUsage struct {
	Server, Tool string
	Tokens       uint64
	Loaded       bool // 完整定义已加载进上下文(false = 按需加载,不占窗口)
}

// ContextKind 是上下文类别的性质。
type ContextKind int

const (
	CtxUsed     ContextKind = iota // 占用窗口
	CtxFree                        // 剩余空间
	CtxBuffer                      // 为压缩预留
	CtxDeferred                    // 按需加载、暂不占用窗口
)

// ContextCategory 是上下文的一个类别(系统提示、工具、消息…)。
type ContextCategory struct {
	Name   string
	Tokens uint64
	Kind   ContextKind
}

// CacheTokens 是提示缓存的 token 分布。
type CacheTokens struct{ Read, Written, Uncached uint64 }

// Total 三项之和。
func (c CacheTokens) Total() uint64 { return c.Read + c.Written + c.Uncached }

// HitPercent 命中率(0..100,取整);没有数据为 0。
func (c CacheTokens) HitPercent() int { return percent(c.Read, c.Total()) }

func percent(a, total uint64) int {
	if total == 0 {
		return 0
	}
	return int(float64(a)/float64(total)*100 + 0.5)
}

// LimitsState 是账户限额的查询状态。
type LimitsState int

const (
	LimitsIdle        LimitsState = iota // 没有进行中的查询
	LimitsChecking                       // 正在向官方接口查询
	LimitsUnavailable                    // 查询失败或当前账户没有限额(如第三方服务商)
)

// Usage 是用量快照。
type Usage struct {
	ContextUsed, ContextSize uint64
	CostUSD                  float64
	FiveHourPct, WeeklyPct   *float32 // 限额使用百分比;nil = 未知
	FiveHourResets           int64    // 重置时刻(Unix 秒);0 = 未知
	WeeklyResets             int64
	ContextCategories        []ContextCategory
	CacheLast, CacheTotal    CacheTokens
	McpTools                 []McpToolUsage
	LimitsAt                 int64 // 限额数据最近一次更新的时刻(Unix 秒);0 = 还没取到
	LimitsState              LimitsState
	ContextDetailErr         string // get_context_usage 失败时的可行动说明(空 = 无错误)
}

// ContextRatio 上下文使用比例 0..1。
func (u Usage) ContextRatio() float32 {
	if u.ContextSize == 0 {
		return 0
	}
	return min(1, float32(u.ContextUsed)/float32(u.ContextSize))
}

// TurnStats 是一个回合的消耗统计(挂在回合的用户消息上,显示在助手回复下面的操作条里)。
type TurnStats struct {
	Model      string // 实际使用的模型 id;空 = 取不到
	Input      uint64 // 未命中缓存的输入 token
	Output     uint64 // 输出 token(含思考)
	Thinking   uint64
	CacheRead  uint64
	CacheWrite uint64
	CostUSD    *float64 // 服务商算不出费用时为 nil
	ApiMs      *uint64  // 模型调用总耗时
	TtftMs     *uint64  // 首字延迟
	Calls      *int     // 模型调用轮数
}

// claudeName: claude-<系列>-<数字…> → 「系列 版本」;不是这种形状返回 false。
func claudeName(id string) (string, bool) {
	rest, ok := strings.CutPrefix(id, "claude-")
	if !ok {
		return "", false
	}
	family := ""
	var version []string
	for _, part := range strings.Split(rest, "-") {
		switch {
		case part != "" && allBytes(part, '0', '9'):
			version = append(version, part)
		case family == "" && part != "" && allBytes(part, 'a', 'z'):
			family = part
		default:
			return "", false
		}
	}
	if family == "" {
		return "", false
	}
	name := strings.ToUpper(family[:1]) + family[1:]
	if len(version) > 0 {
		name += " " + strings.Join(version, ".")
	}
	return name, true
}

func allBytes(s string, lo, hi byte) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < lo || s[i] > hi {
			return false
		}
	}
	return true
}

// ModelLabel 模型显示名:去掉末尾的日期后缀,Claude 的模型 id 再写成「系列 版本」
// (claude-haiku-4-5-20251001 → Haiku 4.5);认不出的保留去掉日期后的 id。没有记录返回空串。
func (t TurnStats) ModelLabel() string {
	m := strings.TrimSpace(t.Model)
	if m == "" || strings.HasPrefix(m, "<") {
		return ""
	}
	base := m
	if i := strings.LastIndexByte(m, '-'); i >= 0 {
		if tail := m[i+1:]; len(tail) == 8 && allBytes(tail, '0', '9') {
			base = m[:i]
		}
	}
	if n, ok := claudeName(base); ok {
		return n
	}
	return base
}

// InputTotal 输入总量:未缓存 + 缓存读取 + 缓存写入。
func (t TurnStats) InputTotal() uint64 { return t.Input + t.CacheRead + t.CacheWrite }

// HitPercent 缓存命中率;没有输入为 0。
func (t TurnStats) HitPercent() int { return percent(t.CacheRead, t.InputTotal()) }

// Empty 是否没有可显示的内容。
func (t TurnStats) Empty() bool { return t.InputTotal() == 0 && t.Output == 0 }

// CostText 费用文案:不到 1 分钱保留三位小数($0.004),否则两位($0.12)。
func CostText(c float64) string {
	if c < 0.01 {
		return fmt.Sprintf("$%.3f", c)
	}
	return fmt.Sprintf("$%.2f", c)
}

// costShown 是否展示费用:claude CLI 给的 total_cost_usd 是按 Claude 价格表估算的,
// 第三方模型(DeepSeek 等)的这个数字不是真实账单,宁可不显示也不误导。
// 没有记录模型名、或是 Claude 系列模型时才展示。
func (t TurnStats) costShown() bool {
	if t.CostUSD == nil {
		return false
	}
	m := strings.ToLower(strings.TrimSpace(t.Model))
	if m == "" || strings.HasPrefix(m, "<") || strings.HasPrefix(m, "claude") {
		return true
	}
	for _, fam := range []string{"sonnet", "opus", "haiku"} {
		if strings.Contains(m, fam) {
			return true
		}
	}
	return false
}

// ShortText 操作条上的简短文案:`↑ 21.1k ↓ 230 · 缓存 98% · $0.004`(没有的项跳过)。
func (t TurnStats) ShortText() string {
	parts := []string{fmt.Sprintf("↑ %s ↓ %s", FormatTokens(t.InputTotal()), FormatTokens(t.Output))}
	if t.InputTotal() > 0 && (t.CacheRead > 0 || t.CacheWrite > 0) {
		parts = append(parts, fmt.Sprintf("缓存 %d%%", t.HitPercent()))
	}
	if t.costShown() {
		parts = append(parts, CostText(*t.CostUSD))
	}
	return strings.Join(parts, " · ")
}

// DetailText 悬停明细:逐项列出 token 分类、费用、延迟与调用轮数(多行)。
func (t TurnStats) DetailText() string {
	k := FormatTokens
	lines := []string{fmt.Sprintf("输入 %s(未缓存 %s · 缓存读取 %s · 缓存写入 %s)", k(t.InputTotal()), k(t.Input), k(t.CacheRead), k(t.CacheWrite))}
	if t.Thinking > 0 {
		lines = append(lines, fmt.Sprintf("输出 %s(其中思考 %s)", k(t.Output), k(t.Thinking)))
	} else {
		lines = append(lines, fmt.Sprintf("输出 %s", k(t.Output)))
	}
	if t.InputTotal() > 0 {
		lines = append(lines, fmt.Sprintf("缓存命中率 %d%%", t.HitPercent()))
	}
	if t.costShown() {
		lines = append(lines, "费用 "+CostText(*t.CostUSD))
	}
	secs := func(ms uint64) string { return fmt.Sprintf("%.1fs", float64(ms)/1000) }
	var timing []string
	if t.TtftMs != nil {
		timing = append(timing, "首字 "+secs(*t.TtftMs))
	}
	if t.ApiMs != nil {
		timing = append(timing, "模型耗时 "+secs(*t.ApiMs))
	}
	if t.Calls != nil {
		timing = append(timing, fmt.Sprintf("%d 次调用", *t.Calls))
	}
	if len(timing) > 0 {
		lines = append(lines, strings.Join(timing, " · "))
	}
	return strings.Join(lines, "\n")
}
