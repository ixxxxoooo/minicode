package claude

import (
	"bufio"
	"io"
	"os"
	"strings"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

func parseISOMs(s string) (int64, bool) {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return 0, false
	}
	return t.UnixMilli(), true
}

// secsBetween 两个时间点之间的用时(秒),限制在 1 秒到 maxSecs 之间;end 不晚于 start 返回 false。
func secsBetween(start, end int64, maxSecs uint64) (uint64, bool) {
	if end <= start {
		return 0, false
	}
	s := uint64(end-start+500) / 1000
	return min(max(s, 1), maxSecs), true
}

type usageRec struct {
	id    string
	stats core.TurnStats
}

// recordUsage 记下一条助手记录的用量:同一条消息(message.id)会有多行记录,后出现的覆盖先出现的。
func recordUsage(v obj, acc *[]usageRec) {
	m := sub(v, "message")
	id, u := str(m, "id"), sub(m, "usage")
	if id == "" || u == nil {
		return
	}
	model := str(m, "model")
	if strings.HasPrefix(model, "<") {
		model = ""
	}
	st := core.TurnStats{
		Model: model, Input: u64(u, "input_tokens"), Output: u64(u, "output_tokens"),
		Thinking:  u64(sub(u, "output_tokens_details"), "thinking_tokens"),
		CacheRead: u64(u, "cache_read_input_tokens"), CacheWrite: u64(u, "cache_creation_input_tokens"),
	}
	for i := range *acc {
		if (*acc)[i].id == id {
			(*acc)[i].stats = st
			return
		}
	}
	*acc = append(*acc, usageRec{id, st})
}

// flushStats 把当前回合累积的各消息用量加总,作为 TurnStats 事件追加;没有用量就什么也不加。
func flushStats(uid *string, acc *[]usageRec, out *[]core.Event) {
	var total core.TurnStats
	var best uint64
	calls := 0
	for i, r := range *acc {
		if i == 0 || r.stats.Output > best { // 主模型:输出最多的那条消息用的模型
			best, total.Model = r.stats.Output, r.stats.Model
		}
		total.Input += r.stats.Input
		total.Output += r.stats.Output
		total.Thinking += r.stats.Thinking
		total.CacheRead += r.stats.CacheRead
		total.CacheWrite += r.stats.CacheWrite
		calls++
	}
	*acc = nil
	id := *uid
	*uid = ""
	if calls > 0 {
		total.Calls = &calls
	}
	if id != "" && !total.Empty() {
		*out = append(*out, core.EvTurnStats{ID: id, Stats: total})
	}
}

// ReplayLines 回放若干行 jsonl,得到可直接喂给 State.Apply 的事件序列。
// 除条目事件,还用每条记录的 timestamp 算出思考与整个回合的用时,作为 EvDuration 附在末尾。
func ReplayLines(lines []string, cwd string) []core.Event {
	tr := NewReplayTranslator(cwd)
	var events, durations []core.Event
	title := ""
	var prevTS int64
	hasPrev := false
	var turnID string
	var turnStart int64
	hasTurn := false
	var turnUsage []usageRec
	curUID := ""
	endTurn := func() {
		if hasTurn && hasPrev {
			if s, ok := secsBetween(turnStart, prevTS, 86400); ok {
				durations = append(durations, core.EvDuration{ID: turnID, Secs: s})
			}
		}
		hasTurn = false
	}
	for _, line := range lines {
		v := parseLine(line)
		if v == nil || boolOf(v, "isSidechain", false) {
			continue
		}
		switch str(v, "type") {
		case "ai-title":
			title = str(v, "aiTitle")
		case "user", "assistant":
			ts, hasTS := parseISOMs(str(v, "timestamp"))
			if str(v, "type") == "assistant" {
				recordUsage(v, &turnUsage)
			}
			evs := tr.Handle(v)
			for _, e := range evs {
				a, ok := e.(core.EvAppend)
				if !ok || a.Parent != "" {
					continue
				}
				switch it := a.Item.(type) {
				case *core.User: // 新的用户提示:上一个回合到此结束(用上一条记录的时间收尾)
					endTurn()
					flushStats(&curUID, &turnUsage, &durations)
					curUID = it.ID
					if hasTS {
						turnID, turnStart, hasTurn = it.ID, ts, true
					}
				case *core.Thinking: // 思考:从上一条记录到这一条的时间
					if hasTS && hasPrev {
						if s, ok := secsBetween(prevTS, ts, 3600); ok {
							durations = append(durations, core.EvDuration{ID: it.ID, Secs: s})
						}
					}
				}
			}
			events = append(events, evs...)
			if hasTS {
				prevTS, hasPrev = ts, true
			}
		}
	}
	endTurn()
	flushStats(&curUID, &turnUsage, &durations)
	events = append(events, durations...)
	if u, ok := estimateContextFromLines(lines); ok {
		events = append(events, core.EvUsage{Usage: u})
	}
	// 回放到末尾仍没有终态通知的后台任务,属于已经退出的上一个进程:用一份空的存活快照把它们收尾成「已停止」,
	// 否则任务坞会一直显示「运行中」。之后本进程自己启动的任务由实时事件创建,不受影响。
	events = append(events, core.EvTasksChanged{})
	if title != "" {
		events = append([]core.Event{core.EvTitleChanged{Title: title}}, events...)
	}
	return events
}

// LoadSession 读取并回放一个会话文件。
func LoadSession(path, cwd string) ([]core.Event, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := bufio.NewReaderSize(f, 256*1024)
	var lines []string
	for {
		line, err := r.ReadString('\n')
		if s := strings.TrimSpace(line); s != "" {
			lines = append(lines, s)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	return ReplayLines(lines, cwd), nil
}

const defaultContextWindow = 200_000

// estimateContextFromLines 用历史 jsonl 最后一条助手消息的 usage 估算上下文占用。
func estimateContextFromLines(lines []string) (core.Usage, bool) {
	var lastUsage obj
	var window uint64 = defaultContextWindow
	for _, line := range lines {
		v := parseLine(line)
		if v == nil || boolOf(v, "isSidechain", false) {
			continue
		}
		switch str(v, "type") {
		case "assistant":
			if u := sub(sub(v, "message"), "usage"); u != nil {
				lastUsage = u
			}
		case "result":
			for _, m := range arr(v, "modelUsage") {
				if w := u64(asObj(m), "contextWindow"); w > 0 {
					window = max(window, w)
				}
			}
			if u := sub(v, "usage"); u != nil {
				lastUsage = u
			}
		}
	}
	if lastUsage == nil {
		return core.Usage{}, false
	}
	used := u64(lastUsage, "input_tokens") + u64(lastUsage, "cache_read_input_tokens") +
		u64(lastUsage, "cache_creation_input_tokens") + u64(lastUsage, "output_tokens")
	if used == 0 {
		return core.Usage{}, false
	}
	cats := []core.ContextCategory{
		{Name: "消息", Tokens: used, Kind: core.CtxUsed},
	}
	free := window
	if window > used {
		free = window - used
		cats = append(cats, core.ContextCategory{Name: "空闲", Tokens: free, Kind: core.CtxFree})
	}
	return core.Usage{
		ContextUsed:       used,
		ContextSize:       window,
		ContextCategories: cats,
		ContextEstimated:  true,
	}, true
}
