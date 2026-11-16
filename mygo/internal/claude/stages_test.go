package claude

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

func jl(t *testing.T, s string) obj {
	t.Helper()
	var v obj
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		t.Fatal(err)
	}
	return v
}

// 实测协议顺序:status(requesting)→ message_start → 工具 → 再次请求 → 文字 → result。
func TestStageClockTwoCalls(t *testing.T) {
	t0 := time.Unix(1000, 0)
	at := func(ms int) time.Time { return t0.Add(time.Duration(ms) * time.Millisecond) }
	var c stageClock
	c.begin(t0, false)
	feed := func(ms int, s string) (bool, *core.CacheTokens) { return c.line(jl(t, s), at(ms)) }

	feed(80, `{"type":"system","subtype":"status","status":"requesting"}`)
	_, cache := feed(5800, `{"type":"stream_event","event":{"type":"message_start","message":{"usage":{"input_tokens":2,"cache_creation_input_tokens":26170,"cache_read_input_tokens":0}}}}`)
	if cache == nil || cache.Written != 26170 {
		t.Fatalf("第一次调用缓存用量没取到: %+v", cache)
	}
	feed(5800, `{"type":"stream_event","event":{"type":"content_block_start","content_block":{"type":"tool_use","name":"Bash"}}}`)
	feed(6290, `{"type":"stream_event","event":{"type":"message_delta","delta":{"stop_reason":"tool_use"}}}`)
	feed(6290, `{"type":"stream_event","event":{"type":"message_stop"}}`)
	feed(7100, `{"type":"system","subtype":"status","status":"requesting"}`)
	feed(10980, `{"type":"stream_event","event":{"type":"message_start","message":{"usage":{"input_tokens":4,"cache_creation_input_tokens":17,"cache_read_input_tokens":26180}}}}`)
	feed(10980, `{"type":"stream_event","event":{"type":"content_block_start","content_block":{"type":"text"}}}`)
	feed(11600, `{"type":"stream_event","event":{"type":"message_delta","delta":{"stop_reason":"end_turn"}}}`)
	feed(11600, `{"type":"stream_event","event":{"type":"message_stop"}}`)
	spans, calls := c.finish(at(64630))

	want := []core.TimingPhase{
		{Kind: core.StagePrep, Ms: 80},
		{Kind: core.StageWait, Ms: 5720, Call: 1},
		{Kind: core.StageWrite, Ms: 490, Call: 1},
		{Kind: core.StageTool, Ms: 810, Label: "Bash"},
		{Kind: core.StageWait, Ms: 3880, Call: 2},
		{Kind: core.StageWrite, Ms: 620, Call: 2},
		{Kind: core.StageFin, Ms: 53030, Call: 2},
	}
	if len(spans) != len(want) {
		t.Fatalf("阶段数 %d != %d: %+v", len(spans), len(want), spans)
	}
	for i := range want {
		if spans[i] != want[i] {
			t.Errorf("第 %d 段 = %+v, 想要 %+v", i, spans[i], want[i])
		}
	}
	if len(calls) != 2 || calls[1].Verdict() != core.CacheHit {
		t.Fatalf("每次调用的缓存: %+v", calls)
	}
	if calls[0].Verdict() != core.CacheWritten {
		t.Errorf("第一次应是「首次写入」: %v", calls[0].Verdict())
	}
}

func TestStageClockColdAndAsk(t *testing.T) {
	t0 := time.Unix(2000, 0)
	at := func(ms int) time.Time { return t0.Add(time.Duration(ms) * time.Millisecond) }
	var c stageClock
	c.begin(t0, true)
	if !c.connected(at(600)) {
		t.Fatal("冷启动握手后应从「连接」转入「本地准备」")
	}
	c.line(jl(t, `{"type":"system","subtype":"status","status":"requesting"}`), at(700))
	c.line(jl(t, `{"type":"stream_event","event":{"type":"message_start","message":{"usage":{}}}}`), at(2000))
	c.line(jl(t, `{"type":"stream_event","event":{"type":"content_block_start","content_block":{"type":"tool_use","name":"Bash"}}}`), at(2000))
	c.line(jl(t, `{"type":"stream_event","event":{"type":"message_delta","delta":{"stop_reason":"tool_use"}}}`), at(2500))
	c.line(jl(t, `{"type":"stream_event","event":{"type":"message_stop"}}`), at(2500))
	// 权限请求到达 → 等你确认;作答后回到「工具」
	c.line(jl(t, `{"type":"control_request","request":{"subtype":"can_use_tool"}}`), at(2600))
	if !c.answered(at(30600)) {
		t.Fatal("作答后应回到工具阶段")
	}
	spans, _ := c.finish(at(33000))
	var ask, tool int64
	for _, p := range spans {
		switch p.Kind {
		case core.StageAsk:
			ask += p.Ms
		case core.StageTool:
			tool += p.Ms
		}
	}
	if ask != 28000 {
		t.Errorf("等你确认应为 28000 ms,得 %d(%+v)", ask, spans)
	}
	if tool != 100+2400 {
		t.Errorf("工具应为 2500 ms,得 %d", tool)
	}
	if spans[0].Kind != core.StageConn || spans[0].Ms != 600 {
		t.Errorf("首段应是 600 ms 的「连接」: %+v", spans[0])
	}
}

// 子 Agent 的事件与新回合以外的行不参与计时。
func TestStageClockIgnoresSubagentAndInactive(t *testing.T) {
	var c stageClock
	if changed, _ := c.line(jl(t, `{"type":"system","subtype":"status","status":"requesting"}`), time.Now()); changed {
		t.Fatal("没开始回合时不应计时")
	}
	t0 := time.Unix(3000, 0)
	c.begin(t0, false)
	if changed, _ := c.line(jl(t, `{"type":"system","subtype":"status","status":"requesting","parent_tool_use_id":"toolu_1"}`), t0.Add(time.Second)); changed {
		t.Fatal("子 Agent 的事件不应影响主回合")
	}
}

func TestCacheVerdict(t *testing.T) {
	cases := []struct {
		c    core.CacheTokens
		want core.CacheVerdict
	}{
		{core.CacheTokens{}, core.CacheUnknown},
		{core.CacheTokens{Read: 25_700, Written: 17, Uncached: 4}, core.CacheHit},
		{core.CacheTokens{Read: 15_355, Written: 10_825, Uncached: 4}, core.CachePartial},
		{core.CacheTokens{Written: 26_170, Uncached: 2}, core.CacheWritten},
		{core.CacheTokens{Uncached: 30_000}, core.CacheMiss},
	}
	for i, c := range cases {
		if got := (core.CallInfo{Cache: c.c}).Verdict(); got != c.want {
			t.Errorf("用例 %d: %v, 想要 %v", i, got, c.want)
		}
	}
}

// 最后一个 message_stop(end_turn)到达即算「回复出完」:Final 事件 + settled;工具调用的 message_stop 不算。
func TestStageClockFinal(t *testing.T) {
	t0 := time.Unix(4000, 0)
	at := func(ms int) time.Time { return t0.Add(time.Duration(ms) * time.Millisecond) }
	var c stageClock
	c.begin(t0, false)
	feed := func(ms int, s string) { c.line(jl(t, s), at(ms)) }
	feed(50, `{"type":"system","subtype":"status","status":"requesting"}`)
	feed(2000, `{"type":"stream_event","event":{"type":"message_start","message":{"usage":{}}}}`)
	feed(2000, `{"type":"stream_event","event":{"type":"content_block_start","content_block":{"type":"tool_use","name":"Bash"}}}`)
	feed(2500, `{"type":"stream_event","event":{"type":"message_delta","delta":{"stop_reason":"tool_use"}}}`)
	feed(2500, `{"type":"stream_event","event":{"type":"message_stop"}}`)
	if c.settled() {
		t.Fatal("tool_use 结束不是回复出完")
	}
	feed(3000, `{"type":"system","subtype":"status","status":"requesting"}`)
	feed(5000, `{"type":"stream_event","event":{"type":"message_start","message":{"usage":{}}}}`)
	feed(5000, `{"type":"stream_event","event":{"type":"content_block_start","content_block":{"type":"text"}}}`)
	feed(6000, `{"type":"stream_event","event":{"type":"message_delta","delta":{"stop_reason":"end_turn"}}}`)
	feed(6000, `{"type":"stream_event","event":{"type":"message_stop"}}`)
	if !c.settled() {
		t.Fatal("end_turn 的 message_stop 应判为回复出完")
	}
	if ev := c.live(at(6000), nil); !ev.Final || ev.Kind != core.StageFin {
		t.Fatalf("live 事件应带 Final: %+v", ev)
	}
	phases, _ := c.snapshot(at(6000))
	if len(phases) == 0 || phases[len(phases)-1].Kind == core.StageFin {
		t.Errorf("临时用时不该带 0 毫秒的收尾段: %+v", phases)
	}
	// 收尾期间又有新请求:退出 Final
	feed(9000, `{"type":"system","subtype":"status","status":"requesting"}`)
	if c.settled() {
		t.Fatal("又有新请求后不应再是 settled")
	}
}
