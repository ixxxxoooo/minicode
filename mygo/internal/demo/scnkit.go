package demo

import (
	"fmt"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// Script 是写演示剧本用的小构建器:自动编号、自动补 TurnStarted / 流式分块。
// 历史回放(Scenario.Live=false)时 After 被忽略;进行中场景(Live=true)按 After 播放。
type Script struct {
	steps []Step
	n     int
	sid   string
}

// NewScript 以 EvReady + EvTitleChanged 起头。
func NewScript(id, title string) *Script {
	s := &Script{sid: id}
	s.Add(0, core.EvReady{SessionID: "demo-" + id, Models: core.DefaultModels(), Model: "sonnet", Mode: core.ModeDefault})
	if title != "" {
		s.Add(0, core.EvTitleChanged{Title: title})
	}
	return s
}

// Add 追加一步原始事件。
func (s *Script) Add(after time.Duration, ev core.Event) *Script {
	s.steps = append(s.steps, Step{After: after, Ev: ev})
	return s
}

// Steps 返回剧本。
func (s *Script) Steps() []Step { return s.steps }

// ID 生成一个本剧本内唯一的条目 id,形如 x12。
func (s *Script) ID(prefix string) string {
	s.n++
	return fmt.Sprintf("%s%d", prefix, s.n)
}

func ms(n int) time.Duration { return time.Duration(n) * time.Millisecond }

// User 追加用户消息并开始一个回合,返回用户消息 id(给 End 用)。
func (s *Script) User(text string, files ...string) string {
	id := s.ID("u")
	s.Add(0, core.EvAppend{Item: &core.User{ID: id, Text: text, Files: files}})
	s.Add(ms(250), core.EvTurnStarted{})
	return id
}

// Think 追加一段思考(流式两块 + 用时)。
func (s *Script) Think(text string, secs uint64) string {
	id := s.ID("t")
	s.Add(ms(150), core.EvAppend{Item: &core.Thinking{ID: id, Streaming: true}})
	for _, c := range chunks(text, 40) {
		s.Add(ms(40), core.EvThinkingDelta{ID: id, Delta: c})
	}
	s.Add(ms(100), core.EvItemDone{ID: id})
	s.Add(0, core.EvDuration{ID: id, Secs: secs})
	return id
}

// Say 追加助手正文(按 24 字符一块流式输出)。
func (s *Script) Say(text string) string {
	id := s.ID("a")
	s.Add(ms(200), core.EvAppend{Item: &core.Assistant{ID: id, Streaming: true}})
	for _, c := range chunks(text, 24) {
		s.Add(ms(30), core.EvTextDelta{ID: id, Delta: c})
	}
	s.Add(ms(60), core.EvItemDone{ID: id})
	return id
}

// Tool 追加一次工具调用:先以运行中出现,再按 st 收尾并带上内容块。
// st 为 ToolRunning / ToolPending 时只出现、不收尾(演示进行中)。
func (s *Script) Tool(tc *core.ToolCall, st core.ToolStatus, durMs int64, content ...core.ToolContent) *core.ToolCall {
	if tc.ID == "" {
		tc.ID = s.ID("k")
	}
	s.Add(ms(150), core.EvAppend{Item: tc})
	if st == core.ToolRunning || st == core.ToolPending {
		return tc
	}
	s.Add(ms(400), core.EvToolUpdate{ID: tc.ID, Patch: core.ToolPatch{Status: core.Status(st), DurationMs: durMs, AppendContent: content}})
	return tc
}

// Notice 追加一条提示。
func (s *Script) Notice(level core.NoticeLevel, text string) {
	s.Add(ms(100), core.EvNotice{Level: level, Text: text})
}

// End 收尾一个回合:用时 + 可选统计 + 结束原因。
func (s *Script) End(uid string, secs uint64, reason core.StopReason, stats *core.TurnStats) {
	if stats != nil {
		s.Add(0, core.EvTurnStats{ID: uid, Stats: *stats})
	}
	s.Add(0, core.EvDuration{ID: uid, Secs: secs})
	s.Add(0, core.EvTurnEnded{Reason: reason})
}

// DefaultStats 造一份带费用的回合统计(演示用,数值随 seed 略有变化)。
func DefaultStats(seed int) *core.TurnStats {
	cost := 0.01 + float64(seed%9)*0.004
	api, ttft, calls := uint64(8000+seed*700), uint64(600+seed%5*90), 2+seed%6
	return &core.TurnStats{Model: "claude-sonnet-5-5", Input: uint64(12000 + seed*900), Output: uint64(300 + seed*37),
		CacheRead: uint64(90000 + seed*1500), CacheWrite: 2000, CostUSD: &cost, ApiMs: &api, TtftMs: &ttft, Calls: &calls}
}
