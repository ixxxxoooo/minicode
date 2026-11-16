package claude

import (
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// stageClock 把 claude 的流式事件切成回合里的阶段(设计稿 resp-time 页)。
// 纯逻辑、不起 goroutine、时间由调用方传入,方便单测。只在 Backend.run 循环里访问。
//
// 实测的事件顺序(claude 2.1.325):
//
//	发出提示 → system init → system status(requesting)   ← 请求已发出
//	         → (上游静默)→ stream_event message_start       ← 第一个字节(带本次调用的 usage)
//	         → content_block_start(thinking / text / tool_use)→ … → message_delta(stop_reason)→ message_stop
//	tool_use 结束后执行工具 → system status(requesting)再来一次 → …
//	最后一个 message_stop → result。
type stageClock struct {
	active   bool
	t0       time.Time
	cur      core.StageKind
	curAt    time.Time
	call     int
	toolName string // 最近一个 tool_use 的名字
	stop     string // 最近一次 message_delta 的 stop_reason
	asks     int    // 未答复的权限 / 提问 / 计划请求个数
	resume   core.StageKind
	spans    []core.TimingPhase
	calls    []core.CallInfo
	curCall  int
	curLabel string
	final    bool // 最后一个 message_stop 的 stop_reason 是 end_turn:回复已出完
}

// begin 开始一个回合:cold = 发送时进程还没起(先是「连接」)。
func (c *stageClock) begin(t0 time.Time, cold bool) {
	*c = stageClock{active: true, t0: t0, curAt: t0}
	if cold {
		c.cur = core.StageConn
	} else {
		c.cur = core.StagePrep
	}
}

// off 回合被打断 / 进程退出:丢弃计时。
func (c *stageClock) off() { *c = stageClock{} }

// enter 进入新阶段;和当前阶段相同则什么都不做。返回是否发生了切换。
func (c *stageClock) enter(k core.StageKind, now time.Time) bool {
	if !c.active || k == c.cur {
		return false
	}
	c.closeSpan(now)
	c.cur, c.curAt = k, now
	c.final = false
	c.curLabel = ""
	if k == core.StageTool {
		c.curLabel = c.toolName
	}
	if k == core.StageWait {
		c.call++
		c.calls = append(c.calls, core.CallInfo{})
	}
	c.curCall = c.call
	return true
}

func (c *stageClock) closeSpan(now time.Time) {
	ms := now.Sub(c.curAt).Milliseconds()
	if c.cur == core.StageNone || ms <= 0 {
		return
	}
	call := 0
	switch c.cur {
	case core.StageWait, core.StageThink, core.StageWrite, core.StageFin:
		call = c.curCall
	}
	c.spans = append(c.spans, core.TimingPhase{Kind: c.cur, Ms: ms, Call: call, Label: c.curLabel})
}

// live 当前阶段的实时快照(给 EvTurnPhase)。
func (c *stageClock) live(now time.Time, cache *core.CacheTokens) core.EvTurnPhase {
	return core.EvTurnPhase{Kind: c.cur, At: now, TurnStart: c.t0, Call: c.call, Cache: cache, Final: c.final && c.cur == core.StageFin}
}

// line 喂入 claude 的一行输出(已解析),返回(是否切换了阶段, 本次调用的缓存用量)。
// 子 Agent 的事件(parent_tool_use_id 非空)不参与:它们不是主回合的等待。
func (c *stageClock) line(v obj, now time.Time) (changed bool, cache *core.CacheTokens) {
	if !c.active || str(v, "parent_tool_use_id") != "" {
		return false, nil
	}
	switch str(v, "type") {
	case "system":
		if str(v, "subtype") == "status" && str(v, "status") == "requesting" {
			c.asks = 0 // 新的请求发出,说明之前的权限 / 提问都已有结果
			return c.enter(core.StageWait, now), nil
		}
	case "control_request":
		if str(sub(v, "request"), "subtype") == "can_use_tool" {
			c.asks++
			if c.asks == 1 {
				c.resume = c.cur
				if c.resume == core.StageAsk || c.resume == core.StageNone {
					c.resume = core.StageTool
				}
				return c.enter(core.StageAsk, now), nil
			}
		}
	case "stream_event":
		ev := sub(v, "event")
		switch str(ev, "type") {
		case "message_start":
			if n := len(c.calls); n > 0 {
				u := sub(sub(ev, "message"), "usage")
				ct := core.CacheTokens{Read: u64(u, "cache_read_input_tokens"), Written: u64(u, "cache_creation_input_tokens"), Uncached: u64(u, "input_tokens")}
				c.calls[n-1].Cache = ct
				cache = &ct
			}
			c.stop = ""
			// 第一个字节到了:等待结束;先记为「生成」,紧随其后的 content_block_start 若是 thinking 再改成「思考」。
			changed = c.enter(core.StageWrite, now)
		case "content_block_start":
			cb := sub(ev, "content_block")
			switch str(cb, "type") {
			case "thinking", "redacted_thinking":
				changed = c.enter(core.StageThink, now)
			case "tool_use":
				c.toolName = str(cb, "name")
				changed = c.enter(core.StageWrite, now)
			default:
				changed = c.enter(core.StageWrite, now)
			}
		case "message_delta":
			c.stop = str(sub(ev, "delta"), "stop_reason")
		case "message_stop":
			if c.stop == "tool_use" {
				changed = c.enter(core.StageTool, now)
			} else {
				changed = c.enter(core.StageFin, now)
				if c.stop == "end_turn" && c.asks == 0 {
					c.final = true
					changed = true // 即使已经在「收尾」里也要再报一次 Final
				}
			}
		}
	}
	return changed, cache
}

// answered 一个权限 / 提问 / 计划请求被作答。全部作答后回到请求到达前的阶段;返回是否切换。
func (c *stageClock) answered(now time.Time) bool {
	if !c.active || c.asks == 0 {
		return false
	}
	c.asks--
	if c.asks > 0 {
		return false
	}
	return c.enter(c.resume, now)
}

// connected 冷启动的进程握手完成、提示已写入:「连接」结束,转入「本地准备」。
func (c *stageClock) connected(now time.Time) bool {
	if c.cur != core.StageConn {
		return false
	}
	return c.enter(core.StagePrep, now)
}

// finish 回合结束:收尾并返回逐阶段耗时与每次调用的缓存。
func (c *stageClock) finish(now time.Time) ([]core.TimingPhase, []core.CallInfo) {
	if !c.active {
		return nil, nil
	}
	c.closeSpan(now)
	spans, calls := mergeSpans(c.spans), c.calls
	c.off()
	return spans, calls
}

// mergeSpans 合并相邻同类(同一次调用、同一工具名)的阶段。
func mergeSpans(in []core.TimingPhase) []core.TimingPhase {
	out := make([]core.TimingPhase, 0, len(in))
	for _, p := range in {
		if n := len(out); n > 0 && out[n-1].Kind == p.Kind && out[n-1].Call == p.Call && out[n-1].Label == p.Label {
			out[n-1].Ms += p.Ms
			continue
		}
		out = append(out, p)
	}
	return out
}

// settled 最后一段回复已出完、正在等 claude 收尾返回 result。
func (c *stageClock) settled() bool { return c.active && c.final && c.cur == core.StageFin }

// snapshot 不结束计时,给出到 now 为止的逐阶段耗时(临时脚注用);当前阶段按已过时间计入。
func (c *stageClock) snapshot(now time.Time) ([]core.TimingPhase, []core.CallInfo) {
	if !c.active {
		return nil, nil
	}
	cp := *c
	cp.spans = append([]core.TimingPhase(nil), c.spans...)
	cp.closeSpan(now)
	return mergeSpans(cp.spans), append([]core.CallInfo(nil), c.calls...)
}
