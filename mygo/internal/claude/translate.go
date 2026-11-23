package claude

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

type blockKind int

const (
	kindText blockKind = iota
	kindThinking
)

type openBlock struct {
	id      string
	parent  string
	created bool
	// pend 是还没建条目时收到的纯空白增量:模型在工具调用之间常吐只有 "\n\n" 的文本块,
	// 不能为它建条目(会成空行、被算进步骤 / 回合统计);等来了真正的文字再连同这些空白一起发出。
	pend string
	// text 是正文块(不是思考);nsf 把正文末尾的 <next_steps> 块截走(下一步建议,见 core/nextsteps.go)
	text bool
	nsf  core.NextStepsFilter
}

type streamedBlock struct {
	kind    blockKind
	claimed bool
}

type toolInfo struct {
	name    string
	input   obj
	started time.Time
	parent  string
}

type planBlock struct {
	id   string
	json strings.Builder
	sent int
}

type blockKey struct {
	msg string
	idx uint64
}

// Translator 把 claude 的每一行 stream-json 输出翻译成 core.Event。有状态:流式块的去重、
// 工具调用与子 Agent 的对应、挂起的权限请求都在这里。只在后端的读线程里使用,不加锁。
type Translator struct {
	replay    bool
	SessionID string
	Cwd       string

	tools     map[string]*toolInfo
	subagents map[string]bool
	curMsg    string
	open      map[blockKey]*openBlock
	streamed  map[string][]*streamedBlock
	pending   map[string]*Pending
	planBlks  map[blockKey]*planBlock
	// 已经以流式卡片出现、还没定稿的计划条目 id
	plansStreamed map[string]bool
	usage         core.Usage
	// Interrupted:用户请求过中断(回合结束原因标成「已取消」)
	Interrupted bool
	// apiError:本回合 claude 发来的 API 错误合成消息的文字(随后的 result 带同样文字,只显示一份)
	apiError string
	seq      uint64
	// CompactManual 本次压缩是用户手动发起的(compacting 状态事件里没有 trigger,靠它区分手动 / 自动)。
	CompactManual bool

	// IDGen 是实时条目 id 的代号:进程重启后序号从头开始,靠它区分,避免与旧进程的条目撞 id。
	IDGen int
	// liveTasks:本进程见过、还没收到终态通知的后台任务。历史里遗留的任务不在其中,无法再停止。
	liveTasks map[string]bool

	// Clock 可注入的时钟,为空用 time.Now。只有测试(回放快照要可复现)会设它。
	Clock func() time.Time
}

// now 当前时间;翻译器里所有「取现在时刻」都走这里。
func (t *Translator) now() time.Time {
	if t.Clock != nil {
		return t.Clock()
	}
	return time.Now()
}

// NewTranslator 创建实时翻译器。
func NewTranslator(cwd string) *Translator {
	return &Translator{
		Cwd: cwd, tools: map[string]*toolInfo{}, subagents: map[string]bool{}, open: map[blockKey]*openBlock{},
		streamed: map[string][]*streamedBlock{}, pending: map[string]*Pending{}, planBlks: map[blockKey]*planBlock{},
		plansStreamed: map[string]bool{}, liveTasks: map[string]bool{},
	}
}

// NewReplayTranslator 创建回放翻译器(历史 jsonl)。
func NewReplayTranslator(cwd string) *Translator {
	t := NewTranslator(cwd)
	t.replay = true
	return t
}

func (t *Translator) nextID(prefix string) string {
	t.seq++
	switch {
	case t.replay:
		return fmt.Sprintf("%s%d", prefix, t.seq)
	case t.IDGen == 0:
		return fmt.Sprintf("l%s%d", prefix, t.seq)
	}
	return fmt.Sprintf("l%d%s%d", t.IDGen, prefix, t.seq)
}

// UserPrompt 用户发出的提示:claude 不回显,由后端自己追加到时间线,并记下条目与消息 uuid 的对应(文件回退用)。
func (t *Translator) UserPrompt(text string, images []string, uuid string) []core.Event {
	id := t.nextID("u")
	return []core.Event{
		core.EvAppend{Item: &core.User{ID: id, Text: text, Images: images}},
		core.EvUserUUID{ID: id, UUID: uuid},
	}
}

// TakePending 取走挂起的请求。
func (t *Translator) TakePending(id string) *Pending {
	p := t.pending[id]
	delete(t.pending, id)
	return p
}

// DrainPending 取出全部挂起请求。
func (t *Translator) DrainPending() map[string]*Pending {
	p := t.pending
	t.pending = map[string]*Pending{}
	return p
}

// TaskIsLive 本进程是否知道这个后台任务还在跑。
func (t *Translator) TaskIsLive(id string) bool { return t.liveTasks[id] }

// Handle 处理一行输出(控制请求与取消请求由 ControlRequest / CancelRequest 处理)。
func (t *Translator) Handle(v obj) []core.Event {
	switch str(v, "type") {
	case "system":
		return t.system(v)
	case "stream_event":
		return t.streamEvent(v)
	case "assistant":
		return t.assistant(v)
	case "user":
		return t.user(v)
	case "result":
		return t.result(v)
	case "rate_limit_event":
		return t.rateLimit(v)
	}
	return nil
}

func (t *Translator) parentOf(v obj) string {
	p := str(v, "parent_tool_use_id")
	if p == "" {
		return ""
	}
	if !t.subagents[p] {
		if info := t.tools[p]; info != nil && (info.name == "Task" || info.name == "Agent") {
			t.subagents[p] = true
		}
	}
	if t.subagents[p] {
		return p
	}
	return ""
}

func notice(level core.NoticeLevel, text string) core.Event {
	return core.EvNotice{Level: level, Text: text}
}

// ---------- system ----------

func (t *Translator) system(v obj) []core.Event {
	switch str(v, "subtype") {
	case "init":
		if id := str(v, "session_id"); id != "" {
			t.SessionID = id
		}
		if t.Cwd == "" {
			t.Cwd = str(v, "cwd")
		}
		if l, ok := mcpServersFromInit(v); ok {
			return []core.Event{core.EvMcpServers{Servers: l}}
		}
	case "compact_boundary":
		// 实测(2.1.325):compact_metadata {trigger: manual|auto, pre_tokens, post_tokens, duration_ms, …}
		md := sub(v, "compact_metadata")
		if md == nil {
			md = sub(v, "compactMetadata")
		}
		pre, _ := num(md, "pre_tokens")
		post, _ := num(md, "post_tokens")
		ms, _ := num(md, "duration_ms")
		auto := str(md, "trigger") == "auto" || (str(md, "trigger") == "" && !t.CompactManual)
		t.CompactManual = false
		return []core.Event{core.EvCompact{State: core.CompactDone, Auto: auto, Pre: uint64(pre), Post: uint64(post), Ms: int64(ms)}}
	case "status":
		if str(v, "status") == "compacting" {
			return []core.Event{core.EvCompact{State: core.CompactRunning, Auto: !t.CompactManual}}
		}
		// 压缩结束:成功由随后的 compact_boundary 给出(带前后 token 数),这里只处理失败
		if r := str(v, "compact_result"); r != "" && r != "success" {
			t.CompactManual = false
			msg := str(v, "compact_error")
			if msg == "" {
				msg = "claude 没有给出原因(compact_result=" + r + ")"
			}
			return []core.Event{core.EvCompact{State: core.CompactFailed, Err: msg}}
		}
		// 权限模式每次变化都会带上当前模式(批准计划、set_permission_mode 之后)
		if m, ok := core.ParseMode(str(v, "permissionMode")); ok {
			return []core.Event{core.EvModeChanged{Mode: m}}
		}
	case "task_started":
		t.liveTasks[str(v, "task_id")] = true
		return []core.Event{core.EvTaskStarted{Task: t.taskFrom(v)}}
	case "task_progress":
		u := sub(v, "usage")
		ev := core.EvTaskProgress{TaskID: str(v, "task_id"), LastTool: str(v, "last_tool_name"), Summary: str(v, "summary")}
		if n, ok := num(u, "total_tokens"); ok {
			ev.Tokens = &n
		}
		if n, ok := num(u, "tool_uses"); ok {
			k := int(n)
			ev.ToolUses = &k
		}
		return []core.Event{ev}
	case "task_updated":
		p := sub(v, "patch")
		ev := core.EvTaskUpdated{TaskID: str(v, "task_id"), Description: str(p, "description")}
		if s := str(p, "status"); s != "" {
			st := core.TaskStatusFromCLI(s)
			ev.Status = &st
		}
		return []core.Event{ev}
	case "task_notification":
		delete(t.liveTasks, str(v, "task_id"))
		return []core.Event{core.EvTaskNotification{
			TaskID: str(v, "task_id"), Status: core.TaskStatusFromCLI(str(v, "status")),
			Summary: str(v, "summary"), OutputFile: str(v, "output_file"),
		}}
	case "background_tasks_changed":
		var list []*core.BackgroundTask
		for _, x := range arr(v, "tasks") {
			list = append(list, t.taskFrom(asObj(x)))
		}
		for _, k := range list {
			t.liveTasks[k.TaskID] = true
		}
		return []core.Event{core.EvTasksChanged{Tasks: list}}
	case "hook_response":
		outcome := str(v, "outcome")
		if outcome == "" || outcome == "success" {
			return nil // 只有失败值得打扰用户(开始与进度太碎)
		}
		text := "钩子 " + str(v, "hook_name") + " 执行失败"
		if c, ok := v["exit_code"].(float64); ok {
			text += fmt.Sprintf("(退出码 %d)", int(c))
		}
		for _, k := range []string{"stderr", "output", "stdout"} {
			if d := strings.TrimSpace(str(v, k)); d != "" {
				text += ":" + truncate(d, 300)
				break
			}
		}
		return []core.Event{notice(core.NoticeWarning, text)}
	case "informational":
		level := core.NoticeInfo
		switch str(v, "level") {
		case "error":
			level = core.NoticeError
		case "warning":
			level = core.NoticeWarning
		}
		if text := strings.TrimSpace(str(v, "content")); text != "" {
			return []core.Event{notice(level, text)}
		}
	case "model_fallback", "model_refusal_fallback", "model_refusal_no_fallback":
		text := strings.TrimSpace(str(v, "content"))
		from, to := str(v, "original_model"), str(v, "fallback_model")
		switch {
		case text != "":
		case to != "":
			text = fmt.Sprintf("模型 %s 暂不可用,已回退到 %s", from, to)
		default:
			text = fmt.Sprintf("模型 %s 拒绝了这次请求", from)
		}
		return []core.Event{notice(core.NoticeWarning, text)}
	case "api_retry":
		attempt, mx := u64(v, "attempt"), u64(v, "max_retries")
		if attempt == 0 {
			attempt = 1
		}
		if mx > 0 {
			return []core.Event{notice(core.NoticeWarning, fmt.Sprintf("请求失败,正在重试(%d/%d)", attempt, mx))}
		}
		return []core.Event{notice(core.NoticeWarning, fmt.Sprintf("请求失败,正在重试(第 %d 次)", attempt))}
	}
	return nil
}

// taskFrom 把 task_started 或快照里的一项解析成后台任务。
func (t *Translator) taskFrom(v obj) *core.BackgroundTask {
	toolUse := str(v, "tool_use_id")
	kind := core.TaskKindFromType(str(v, "task_type"))
	// Monitor 在协议里也是 local_bash,靠发起它的工具名区分
	if info := t.tools[toolUse]; info != nil && info.name == "Monitor" {
		kind = core.TaskMonitor
	}
	desc := ""
	for _, k := range []string{"description", "prompt", "subagent_type"} {
		if s := strings.TrimSpace(str(v, k)); s != "" {
			desc = s
			break
		}
	}
	line := strings.TrimSpace(strings.SplitN(desc, "\n", 2)[0])
	if r := []rune(line); len(r) > 120 {
		line = string(r[:120])
	}
	task := core.NewBackgroundTask(t.now().UnixMilli(), str(v, "task_id"), kind, line)
	task.ToolUseID = toolUse
	task.RunID = str(v, "run_id")
	if p, ok := num(v, "pid"); ok {
		task.ProcessID = int(p)
	}
	task.Ambient = boolOf(v, "ambient", false)
	task.OwnedBySub = boolOf(v, "owned_by_subagent", false)
	task.Backgrounded = boolOf(v, "is_backgrounded", true)
	task.OutputFile = str(v, "output_file")
	return task
}

// ---------- 流式 ----------

func (t *Translator) streamEvent(v obj) []core.Event {
	parent := t.parentOf(v)
	ev := sub(v, "event")
	msgID := str(v, "api_message_id")
	if msgID == "" {
		msgID = t.curMsg
	}
	index := u64(ev, "index")
	key := blockKey{msgID, index}
	switch str(ev, "type") {
	case "message_start":
		t.curMsg = str(sub(ev, "message"), "id")
		// 每次调用模型的开头就带着当前上下文的 token 数:回合进行中(长工具链)也让圆环实时跟着涨
		return t.liveContext(sub(sub(ev, "message"), "usage"), parent == "")
	case "message_delta":
		return t.liveContext(sub(ev, "usage"), parent == "")
	case "content_block_start":
		block := sub(ev, "content_block")
		// ExitPlanMode:计划正文开始生成就让卡片出现在正文里(文字随 input_json_delta 补全)
		if str(block, "type") == "tool_use" && str(block, "name") == "ExitPlanMode" && str(block, "id") != "" {
			id := str(block, "id")
			t.planBlks[key] = &planBlock{id: id}
			t.plansStreamed[id] = true
			return []core.Event{core.EvAppend{Parent: parent, Item: &core.PlanApproval{ID: id, Streaming: true}}}
		}
		if bt := str(block, "type"); bt != "text" && bt != "thinking" {
			return nil
		}
		t.open[key] = &openBlock{id: fmt.Sprintf("%s:%d", msgID, index), parent: parent, text: str(block, "type") == "text"}
	case "content_block_delta":
		delta := sub(ev, "delta")
		if str(delta, "type") == "input_json_delta" {
			pb := t.planBlks[key]
			if pb == nil {
				return nil
			}
			pb.json.WriteString(str(delta, "partial_json"))
			md, ok := partialJSONString(pb.json.String(), "plan")
			if !ok || len(md) <= pb.sent {
				return nil
			}
			pb.sent = len(md)
			return []core.Event{core.EvPlanUpdated{ID: pb.id, Markdown: md, Streaming: true}}
		}
		var text string
		var kind blockKind
		switch str(delta, "type") {
		case "text_delta":
			text, kind = str(delta, "text"), kindText
		case "thinking_delta":
			text, kind = str(delta, "thinking"), kindThinking
		default:
			return nil
		}
		b := t.open[key]
		if text == "" || b == nil {
			return nil
		}
		var extra []core.Event
		if kind == kindText {
			// 正文末尾的 <next_steps> 块不进会话流:截走并解析成建议(只取主对话的,子 Agent 的丢弃)
			show, items := b.nsf.Push(text)
			if len(items) > 0 && b.parent == "" {
				extra = append(extra, core.EvNextSteps{Items: items})
			}
			if text = show; text == "" {
				return extra
			}
		}
		if !b.created {
			if strings.TrimSpace(text) == "" {
				b.pend += text
				return extra
			}
			text, b.pend = b.pend+text, ""
			b.created = true
			t.streamed[msgID] = append(t.streamed[msgID], &streamedBlock{kind: kind})
		}
		if kind == kindText {
			return append(extra, core.EvTextDelta{Parent: b.parent, ID: b.id, Delta: text})
		}
		return []core.Event{core.EvThinkingDelta{Parent: b.parent, ID: b.id, Delta: text}}
	case "content_block_stop":
		if _, ok := t.planBlks[key]; ok {
			delete(t.planBlks, key)
			return nil
		}
		if b := t.open[key]; b != nil {
			delete(t.open, key)
			var evs []core.Event
			if b.text { // 末尾被扣住、其实不是 <next_steps> 开标签的字(如一个孤零零的 "<")补发出去
				if tail := b.nsf.Flush(); tail != "" {
					if !b.created {
						tail, b.pend = b.pend+tail, ""
						b.created = true
						t.streamed[msgID] = append(t.streamed[msgID], &streamedBlock{kind: kindText})
					}
					evs = append(evs, core.EvTextDelta{Parent: b.parent, ID: b.id, Delta: tail})
				}
			}
			if b.created {
				evs = append(evs, core.EvItemDone{Parent: b.parent, ID: b.id})
			}
			return evs
		}
	}
	return nil
}

// claimStreamed 完整消息里的文本 / 思考块是否已被流式增量展示过;命中则认领并返回 true。
func (t *Translator) claimStreamed(msgID string, kind blockKind) bool {
	for _, s := range t.streamed[msgID] {
		if s.kind == kind && !s.claimed {
			s.claimed = true
			return true
		}
	}
	return false
}

// isAPIErrorMessage 是不是 claude 为 API 错误合成的 assistant 消息。
func isAPIErrorMessage(v obj, text string) bool {
	msg := sub(v, "message")
	flagged := boolOf(v, "isApiErrorMessage", false) || boolOf(msg, "isApiErrorMessage", false)
	return flagged || (str(msg, "model") == "<synthetic>" && strings.HasPrefix(strings.TrimLeft(text, " \t\r\n"), "API Error"))
}

// ---------- assistant ----------

func (t *Translator) assistant(v obj) []core.Event {
	parent := t.parentOf(v)
	msg := sub(v, "message")
	msgID := str(msg, "id")
	var out []core.Event
	for _, b := range arr(msg, "content") {
		block := asObj(b)
		switch str(block, "type") {
		case "text":
			text := str(block, "text")
			if strings.TrimSpace(text) == "" || t.claimStreamed(msgID, kindText) {
				continue
			}
			// 完整消息 / 历史回放:同样把末尾的 <next_steps> 块剥掉(流式时已截走的上面已 continue)
			var nsItems []string
			if text, nsItems = core.StripNextSteps(text); len(nsItems) > 0 && parent == "" {
				out = append(out, core.EvNextSteps{Items: nsItems})
			}
			if strings.TrimSpace(text) == "" {
				continue
			}
			// API 错误:合成的 assistant 消息与回合结束的 result 带同样的文字,实时流里只留 result 那条
			if !t.replay && isAPIErrorMessage(v, text) {
				t.apiError = text
				continue
			}
			out = append(out, core.EvAppend{Parent: parent, Item: &core.Assistant{ID: t.nextID("a"), Text: text}})
		case "thinking":
			text := str(block, "thinking")
			if t.claimStreamed(msgID, kindThinking) || strings.TrimSpace(text) == "" {
				continue
			}
			out = append(out, core.EvAppend{Parent: parent, Item: &core.Thinking{ID: t.nextID("th"), Text: text}})
		case "tool_use":
			out = append(out, t.toolUse(block, parent)...)
		}
	}
	return out
}

func (t *Translator) toolUse(block obj, parent string) []core.Event {
	id, name := str(block, "id"), str(block, "name")
	input := sub(block, "input")
	info := &toolInfo{name: name, input: input, parent: parent}
	if !t.replay {
		info.started = t.now()
	}
	t.tools[id] = info
	switch name {
	case "TodoWrite":
		return []core.Event{core.EvTodos{ID: id, Items: parseTodos(input)}}
	case "AskUserQuestion":
		return nil
	case "ExitPlanMode":
		if !t.replay {
			return nil
		}
		// 历史里同一个 tool_use 会先后落几份快照(后面的可能是空输入),空正文的不显示
		md := str(input, "plan")
		if strings.TrimSpace(md) == "" {
			return nil
		}
		return []core.Event{core.EvAppend{Parent: parent, Item: &core.PlanApproval{ID: id, Markdown: md}}}
	case "Task", "Agent":
		t.subagents[id] = true
		typ := str(input, "subagent_type")
		if typ == "" {
			typ = "general-purpose"
		}
		st := core.SubRunning
		if boolOf(input, "run_in_background", false) {
			st = core.SubBackground
		}
		card := core.NewSubagent(id, typ, str(input, "description"), st)
		card.Prompt = str(input, "prompt")
		return []core.Event{core.EvAppend{Parent: parent, Item: card}}
	}
	return []core.Event{core.EvAppend{Parent: parent, Item: BuildTool(id, name, input, t.Cwd)}}
}

// ---------- user(工具结果 / 回放中的用户输入) ----------

func (t *Translator) user(v obj) []core.Event {
	parent := t.parentOf(v)
	msg := sub(v, "message")
	var out []core.Event
	if s, ok := msg["content"].(string); ok {
		if t.replay {
			out = append(out, t.replayUser(s, nil, v)...)
		}
		return out
	}
	var texts, images []string
	for _, b := range arr(msg, "content") {
		block := asObj(b)
		switch str(block, "type") {
		case "image":
			if t.replay { // 回放时用户消息里的图片:落盘后按路径展示
				if p := storeImage(block); p != "" {
					images = append(images, p)
				}
			}
		case "tool_result":
			tur := asObj(v["tool_use_result"])
			if tur == nil {
				tur = asObj(v["toolUseResult"]) // 历史 jsonl 里是驼峰
			}
			out = append(out, t.toolResult(block, tur, parent)...)
		case "text":
			if strings.HasPrefix(str(block, "text"), IDEContextTag) { // 随消息发出的 IDE 上下文,不是用户输入
				continue
			}
			texts = append(texts, str(block, "text"))
		}
	}
	if t.replay && (len(texts) > 0 || len(images) > 0) {
		out = append(out, t.replayUser(strings.Join(texts, "\n"), images, v)...)
	}
	return out
}

// replayUser 回放时把真实用户输入还原成条目;过滤掉元信息与斜杠命令产生的内部文本。
func (t *Translator) replayUser(text string, images []string, v obj) []core.Event {
	// 后台任务完成时 claude 把通知当作「用户消息」注入;它不是用户输入,转成任务通知
	if strings.HasPrefix(strings.TrimLeft(text, " \t\r\n"), "<task-notification>") {
		return t.replayTaskNotification(text)
	}
	internal := boolOf(v, "isMeta", false) || boolOf(v, "isSidechain", false) ||
		strings.HasPrefix(text, "<command-") || strings.HasPrefix(text, "<local-command") ||
		strings.HasPrefix(text, "Caveat:") || strings.HasPrefix(text, "<system-reminder>") ||
		strings.HasPrefix(text, "[Request interrupted by user") ||
		(strings.TrimSpace(text) == "" && len(images) == 0)
	if internal {
		return nil
	}
	id := t.nextID("u")
	out := []core.Event{core.EvAppend{Item: &core.User{ID: id, Text: text, Images: images}}}
	if uuid := str(v, "uuid"); uuid != "" { // jsonl 里这条记录的 uuid 就是文件检查点的键
		out = append(out, core.EvUserUUID{ID: id, UUID: uuid})
	}
	return out
}

func (t *Translator) replayTaskNotification(text string) []core.Event {
	id := tagText(text, "task-id")
	if id == "" {
		return nil
	}
	status := core.TaskStatusFromCLI(tagText(text, "status"))
	out := []core.Event{core.EvTaskNotification{
		TaskID: id, Status: status,
		Summary: tagText(text, "summary"), OutputFile: tagText(text, "output-file"),
	}}
	// 通知里的 <result> 才是子 Agent 的完整结论(summary 只有一句「Agent … finished」):写到子 Agent 卡的「结果」
	if res, tu := tagText(text, "result"), tagText(text, "tool-use-id"); res != "" && tu != "" {
		st := core.SubStopped
		switch status {
		case core.TaskCompleted:
			st = core.SubCompleted
		case core.TaskFailed:
			st = core.SubFailed
		}
		out = append(out, core.EvSubagentFinished{ID: tu, Status: st, Summary: res})
	}
	return out
}

func (t *Translator) subagentResult(id string, block, tur obj, isErr bool, info *toolInfo) []core.Event {
	text, _ := resultText(block["content"])
	launched := tur != nil && (boolOf(tur, "isAsync", false) || str(tur, "status") == "async_launched") ||
		strings.HasPrefix(text, "Async agent launched")
	if launched && !isErr {
		out := []core.Event{core.EvSubagentStatus{ID: id, Status: core.SubBackground}}
		agentID := str(tur, "agentId")
		if agentID == "" {
			if _, rest, ok := strings.Cut(text, "agentId:"); ok {
				if f := strings.Fields(rest); len(f) > 0 {
					agentID = f[0]
				}
			}
		}
		outFile := str(tur, "outputFile")
		if agentID != "" {
			if t.replay {
				var input obj
				if info != nil {
					input = info.input
				}
				desc := "后台子代理"
				for _, k := range []string{"description", "prompt"} {
					if d := str(input, k); d != "" {
						desc = d
						break
					}
				}
				task := core.NewBackgroundTask(0, agentID, core.TaskAgent, strings.SplitN(desc, "\n", 2)[0])
				task.ToolUseID, task.OutputFile = id, outFile
				out = append(out, core.EvTaskStarted{Task: task})
			} else {
				out = append(out, core.EvTaskUpdated{TaskID: agentID, OutputFile: outFile})
			}
		}
		return out
	}
	summary := text
	if tur != nil {
		if c, ok := tur["content"]; ok {
			summary, _ = resultText(c)
		}
	}
	summary = strings.TrimSpace(summary)
	if r := []rune(summary); len(r) > 400 {
		summary = string(r[:400])
	}
	st := core.SubCompleted
	if isErr {
		st = core.SubFailed
	}
	return []core.Event{core.EvSubagentFinished{
		ID: id, Status: st, Summary: summary, Tokens: u64(tur, "totalTokens"),
		ToolCount: int(u64(tur, "totalToolUseCount")), DurationMs: int64(u64(tur, "totalDurationMs")),
	}}
}

func (t *Translator) toolResult(block, tur obj, parent string) []core.Event {
	id := str(block, "tool_use_id")
	isErr := boolOf(block, "is_error", false)
	info := t.tools[id]
	delete(t.tools, id)
	if t.subagents[id] {
		return t.subagentResult(id, block, tur, isErr, info)
	}
	if info == nil {
		return nil
	}
	if t.replay && info.name == "ExitPlanMode" {
		// 工具结果是用户的决定:报错 = 没批准,否则批准
		return []core.Event{core.EvPlanResolved{ID: id, Approved: !isErr}}
	}
	switch info.name {
	case "TodoWrite", "AskUserQuestion", "ExitPlanMode":
		return nil
	}
	text, images := resultText(block["content"])
	patch := core.ToolPatch{}
	if isErr {
		patch.Status = core.Status(core.ToolFailed)
	} else {
		patch.Status = core.Status(core.ToolCompleted)
	}
	if !info.started.IsZero() {
		patch.DurationMs = t.now().Sub(info.started).Milliseconds()
	}
	if parent == "" {
		parent = info.parent
	}
	kind := core.KindFromToolName(info.name)
	switch {
	// 后台 Bash:启动结果只是「已在后台运行」,输出在任务面板里看,不当作终端输出
	case kind == core.KindExecute && info.name == "Bash" && !isErr && backgroundTaskID(info.input, tur, text) != "":
		taskID := backgroundTaskID(info.input, tur, text)
		patch.TaskID = taskID
		outFile := backgroundOutputFile(text)
		var out []core.Event
		if t.replay { // 回放的历史没有 system/task_started,这里补一个任务
			task := core.NewBackgroundTask(0, taskID, core.TaskShell, BuildTool(id, info.name, info.input, t.Cwd).Title)
			task.ToolUseID, task.OutputFile = id, outFile
			out = append(out, core.EvTaskStarted{Task: task})
		} else {
			out = append(out, core.EvTaskUpdated{TaskID: taskID, OutputFile: outFile})
		}
		return append(out, core.EvToolUpdate{Parent: parent, ID: id, Patch: patch})
	case kind == core.KindExecute && info.name == "Bash":
		output := text
		if tur != nil {
			if _, has := tur["stdout"]; has {
				o, e := str(tur, "stdout"), str(tur, "stderr")
				output = o
				if e != "" {
					output = o + "\n" + e
				}
			}
		}
		code := 0
		if isErr {
			code = parseExitCode(text)
		}
		patch.AppendContent = append(patch.AppendContent, &core.TerminalBlock{Command: str(info.input, "command"), Output: output, ExitCode: core.Int(code)})
	case kind == core.KindRead && !isErr:
		if f := sub(tur, "file"); f != nil {
			if n, ok := num(f, "numLines"); ok {
				patch.Title = fmt.Sprintf("%s · %d 行", BuildTool(id, info.name, info.input, t.Cwd).Title, n)
			}
		}
	case (kind == core.KindEdit || kind == core.KindWrite) && !isErr:
	default:
		if strings.TrimSpace(text) != "" {
			patch.AppendContent = append(patch.AppendContent, &core.TextBlock{Text: truncate(text, 20000)})
		}
	}
	if isErr && kind != core.KindExecute && strings.TrimSpace(text) != "" && len(patch.AppendContent) == 0 {
		patch.AppendContent = append(patch.AppendContent, &core.TextBlock{Text: truncate(text, 4000)})
	}
	for n, p := range images {
		label := fmt.Sprintf("图片 %d", n+1)
		if kind == core.KindRead && p != "" {
			label = filepath.Base(str(info.input, "file_path"))
		}
		patch.AppendContent = append(patch.AppendContent, &core.ImageBlock{Label: label, Path: p})
	}
	return []core.Event{core.EvToolUpdate{Parent: parent, ID: id, Patch: patch}}
}

// liveContext 回合进行中用某次 API 调用的用量刷新「当前上下文占用」:只认主对话,数没变时不发事件。
func (t *Translator) liveContext(usage obj, main bool) []core.Event {
	used := contextTokens(usage)
	if !main || used == 0 || used == t.usage.ContextUsed {
		return nil
	}
	t.usage.ContextUsed = used
	return []core.Event{core.EvUsage{Usage: t.usage}}
}

// ---------- result / 限额 ----------

func (t *Translator) result(v obj) []core.Event {
	var out []core.Event
	if c, ok := flt(v, "total_cost_usd"); ok {
		t.usage.CostUSD = c
	}
	var size uint64
	for _, m := range sub(v, "modelUsage") {
		size = max(size, u64(asObj(m), "contextWindow"))
	}
	if size > 0 {
		t.usage.ContextSize = size
	}
	usage := sub(v, "usage")
	// 取最后一次迭代的用量作为当前上下文占用;没有迭代信息时用整体用量
	last := usage
	hasIter := false
	if it := arr(usage, "iterations"); len(it) > 0 {
		last, hasIter = asObj(it[len(it)-1]), true
	}
	used := u64(last, "input_tokens") + u64(last, "cache_read_input_tokens") + u64(last, "cache_creation_input_tokens") + u64(last, "output_tokens")
	// 没有迭代明细时 usage 是整个回合所有调用的合计(缓存读取被重复累加,多轮工具调用会虚高);
	// 多于一次调用就保留实时值(每次调用自带的用量,已是准确的当前占用)
	turns := u64(v, "num_turns")
	if _, ok := v["num_turns"]; !ok {
		turns = 1
	}
	if used > 0 && (hasIter || turns <= 1) {
		t.usage.ContextUsed = used
	}
	cache := func(u obj) core.CacheTokens {
		return core.CacheTokens{Read: u64(u, "cache_read_input_tokens"), Written: u64(u, "cache_creation_input_tokens"), Uncached: u64(u, "input_tokens")}
	}
	lastCache, turnCache := cache(last), cache(usage)
	if lastCache.Total() > 0 {
		t.usage.CacheLast = lastCache
	}
	t.usage.CacheTotal.Read += turnCache.Read
	t.usage.CacheTotal.Written += turnCache.Written
	t.usage.CacheTotal.Uncached += turnCache.Uncached
	out = append(out, core.EvUsage{Usage: t.usage})
	if st := turnStats(v); !st.Empty() {
		out = append(out, core.EvTurnStats{Stats: st})
	}

	isErr := boolOf(v, "is_error", false)
	// 合成的 API 错误消息被跳过了,而这次 result 又没标错误:把错误文字补成提示,不能丢
	if t.apiError != "" && !isErr {
		out = append(out, notice(core.NoticeError, t.apiError))
	}
	t.apiError = ""
	var reason core.StopReason
	switch {
	case t.Interrupted:
		reason = core.StopCancelled
	case isErr:
		text := str(v, "result")
		if text == "" {
			var errs []string
			for _, e := range arr(v, "errors") {
				if s, ok := e.(string); ok {
					errs = append(errs, s)
				}
			}
			if text = strings.Join(errs, "\n"); text == "" {
				text = fmt.Sprintf("回合失败(%s)", str(v, "subtype"))
			}
		}
		out = append(out, notice(core.NoticeError, text))
		reason = core.StopError
	default:
		switch str(v, "stop_reason") {
		case "refusal":
			reason = core.StopRefusal
		case "max_tokens":
			reason = core.StopMaxTokens
		default:
			reason = core.StopEndTurn
		}
	}
	t.Interrupted = false
	// 回合结束后,流式状态不应再有残留
	t.open = map[blockKey]*openBlock{}
	t.streamed = map[string][]*streamedBlock{}
	// 计划卡生成到一半回合就结束了:没有等到权限请求,按未批准收尾,免得留下点不动的 Build 按钮
	t.planBlks = map[blockKey]*planBlock{}
	for id := range t.plansStreamed {
		out = append(out, core.EvPlanResolved{ID: id, Approved: false})
	}
	t.plansStreamed = map[string]bool{}
	return append(out, core.EvTurnEnded{Reason: reason})
}

// turnStats 从 result 事件取出本回合的消耗统计。
// usage 是本回合所有模型调用的合计;费用用 model_usage_since_last_result(单回合;
// total_cost_usd 是会话累计值,不能直接用);没有费用(或为 0,如第三方服务商)时为 nil。
func turnStats(v obj) core.TurnStats {
	u := sub(v, "usage")
	cost := 0.0
	model := ""
	var best uint64
	first := true
	for k, x := range sub(v, "model_usage_since_last_result") {
		xm := asObj(x)
		if c, ok := flt(xm, "costUSD"); ok {
			cost += c
		}
		if o := u64(xm, "outputTokens"); first || o > best || (o == best && k < model) {
			best, first = o, false
			model = str(xm, "canonicalModel")
			if model == "" {
				model = k
			}
		}
	}
	st := core.TurnStats{
		Model: model, Input: u64(u, "input_tokens"), Output: u64(u, "output_tokens"),
		Thinking:  u64(sub(u, "output_tokens_details"), "thinking_tokens"),
		CacheRead: u64(u, "cache_read_input_tokens"), CacheWrite: u64(u, "cache_creation_input_tokens"),
	}
	if cost > 0 {
		st.CostUSD = &cost
	}
	if n, ok := num(v, "duration_api_ms"); ok {
		st.ApiMs = &n
	}
	if n, ok := num(v, "ttft_ms"); ok {
		st.TtftMs = &n
	}
	if n, ok := num(v, "num_turns"); ok {
		c := int(n)
		st.Calls = &c
	}
	return st
}

func (t *Translator) rateLimit(v obj) []core.Event {
	info := sub(v, "rate_limit_info")
	windows := sub(info, "unifiedWindows")
	pct := func(k string) *float32 {
		if f, ok := flt(sub(windows, k), "utilization"); ok {
			p := float32(f * 100)
			return &p
		}
		return nil
	}
	if p := pct("five_hour"); p != nil {
		t.usage.FiveHourPct, t.usage.LimitsAt = p, t.now().Unix()
	}
	if p := pct("seven_day"); p != nil {
		t.usage.WeeklyPct, t.usage.LimitsAt = p, t.now().Unix()
	}
	out := []core.Event{core.EvUsage{Usage: t.usage}}
	status := str(info, "status")
	if status != "" && status != "allowed" && status != "allowed_warning" {
		which := ""
		switch str(info, "rateLimitType") {
		case "five_hour":
			which = "5 小时"
		case "seven_day", "seven_day_opus", "seven_day_sonnet":
			which = "每周"
		}
		out = append(out, notice(core.NoticeWarning, "已触及"+which+"使用限额,请稍后再试"))
	}
	return out
}

// ---------- 控制请求 ----------

// ControlRequest 处理 claude 发来的控制请求,返回要追加的事件与需要立刻回复的应答(nil = 不用立刻回复:
// 权限类请求等用户决定后由调用方应答)。
func (t *Translator) ControlRequest(v obj) ([]core.Event, obj) {
	reqID := str(v, "request_id")
	req := sub(v, "request")
	switch sub := str(req, "subtype"); sub {
	case "can_use_tool":
		return t.canUseTool(reqID, req), nil
	case "elicitation":
		// MCP 服务向用户索取输入:暂不支持在对话里填写,自动拒绝,但要让用户知道发生了什么
		server := "MCP 服务"
		for _, k := range []string{"mcp_server_name", "server_name", "serverName"} {
			if s := str(req, k); s != "" {
				server = s
				break
			}
		}
		text := server + " 请求输入,已自动拒绝(暂不支持在对话里填写)"
		if m := str(req, "message"); m != "" {
			text = server + " 请求输入「" + m + "」,已自动拒绝(暂不支持在对话里填写)"
		}
		return []core.Event{notice(core.NoticeInfo, text)}, successResponse(reqID, obj{"action": "decline"})
	case "hook_callback":
		return nil, successResponse(reqID, obj{})
	default:
		return nil, errorResponse(reqID, "不支持的控制请求:"+sub)
	}
}

func (t *Translator) canUseTool(reqID string, req obj) []core.Event {
	toolName := str(req, "tool_name")
	input := sub(req, "input")
	if input == nil {
		input = obj{}
	}
	toolUseID := str(req, "tool_use_id")
	itemID := toolUseID
	if itemID == "" {
		itemID = reqID
	}
	parent := ""
	if info := t.tools[toolUseID]; info != nil {
		parent = info.parent
	}
	kind := PendPermission
	switch toolName {
	case "AskUserQuestion":
		kind = PendQuestion
	case "ExitPlanMode":
		kind = PendPlan
	}
	p := &Pending{RequestID: reqID, ToolUseID: toolUseID, ToolName: toolName, Input: input, Suggestions: arr(req, "permission_suggestions"), Kind: kind}
	var ev core.Event
	switch {
	case kind == PendQuestion:
		ev = core.EvAppend{Parent: parent, Item: &core.QuestionRequest{ID: itemID, Questions: parseQuestions(input)}}
	case kind == PendPlan && t.plansStreamed[itemID]:
		// 计划已经以流式卡片出现过:定稿正文(此时才有 Build 按钮),不重复追加
		delete(t.plansStreamed, itemID)
		ev = core.EvPlanUpdated{ID: itemID, Markdown: str(input, "plan")}
	case kind == PendPlan:
		ev = core.EvAppend{Parent: parent, Item: &core.PlanApproval{ID: itemID, Markdown: str(input, "plan")}}
	default:
		tool := BuildTool(itemID, toolName, input, t.Cwd)
		ev = core.EvPromoteToPermission{Parent: parent, Request: &core.PermissionRequest{ID: itemID, Tool: tool, Options: permissionOptions(p)}}
	}
	t.pending[itemID] = p
	return []core.Event{ev}
}

// CancelRequest claude 撤销了某个控制请求(例如回合被中断):把对应卡片标记为已拒绝。
func (t *Translator) CancelRequest(reqID string) []core.Event {
	for key, p := range t.pending {
		if p.RequestID != reqID {
			continue
		}
		delete(t.pending, key)
		switch p.Kind {
		case PendQuestion:
			return []core.Event{core.EvQuestionAnswered{ID: key}}
		case PendPlan:
			return []core.Event{core.EvPlanResolved{ID: key}}
		}
		return []core.Event{core.EvPermissionResolved{ID: key, Option: "reject"}}
	}
	return nil
}

// ---------- 控制应答 → 事件 ----------

// McpStatusEvent 处理 mcp_status 的应答:每个服务器的状态、失败原因与工具列表。格式不符或失败返回 false。
func (t *Translator) McpStatusEvent(body obj, ok bool) (core.Event, bool) {
	list := arr(body, "mcpServers")
	if !ok || list == nil {
		return nil, false
	}
	var out []core.McpServerStatus
	for _, s := range list {
		sm := asObj(s)
		name := str(sm, "name")
		if name == "" {
			continue
		}
		st := core.McpServerStatus{Name: name, Conn: core.ParseMcpConn(str(sm, "status")), Error: shortError(str(sm, "error"))}
		for _, tl := range arr(sm, "tools") {
			st.Tools = append(st.Tools, strings.TrimPrefix(str(asObj(tl), "name"), "mcp__"+core.McpNormName(name)+"__"))
		}
		out = append(out, st)
	}
	return core.EvMcpServers{Servers: out}, true
}

// ContextBreakdown 处理 get_context_usage 的应答:按类别的明细与总量。
func (t *Translator) ContextBreakdown(body obj, ok bool) (core.Event, bool) {
	ev, hit := ApplyContextUsage(body, ok, &t.usage)
	if hit {
		t.usage = ev.(core.EvUsage).Usage
	}
	return ev, hit
}

// UsageSnapshot 返回当前用量快照(控制请求失败时用于附带错误文案)。
func (t *Translator) UsageSnapshot() core.Usage { return t.usage }

// UsageChecking 开始查询账户限额:返回带「查询中」状态的快照。
func (t *Translator) UsageChecking() core.Event {
	t.usage.LimitsState = core.LimitsChecking
	return core.EvUsage{Usage: t.usage}
}

// AccountUsage 处理 get_usage 的应答:rate_limits.five_hour / seven_day 的 utilization 已是百分比。
func (t *Translator) AccountUsage(body obj, ok bool) core.Event {
	limits := sub(body, "rate_limits")
	pct := func(k string) *float32 {
		if f, has := flt(sub(limits, k), "utilization"); has {
			p := float32(f)
			return &p
		}
		return nil
	}
	resets := func(k string) int64 {
		if s := str(sub(limits, k), "resets_at"); s != "" {
			if tm, err := time.Parse(time.RFC3339, s); err == nil {
				return tm.Unix()
			}
		}
		return 0
	}
	five, weekly := pct("five_hour"), pct("seven_day")
	if ok && (five != nil || weekly != nil) {
		if r := resets("five_hour"); r != 0 {
			t.usage.FiveHourResets = r
		}
		if r := resets("seven_day"); r != 0 {
			t.usage.WeeklyResets = r
		}
		if five != nil {
			t.usage.FiveHourPct = five
		}
		if weekly != nil {
			t.usage.WeeklyPct = weekly
		}
		t.usage.LimitsState, t.usage.LimitsAt = core.LimitsIdle, t.now().Unix()
	} else {
		t.usage.LimitsState = core.LimitsUnavailable
	}
	return core.EvUsage{Usage: t.usage}
}

// ---------- 辅助函数 ----------

func mcpServersFromInit(init obj) ([]core.McpServerStatus, bool) {
	servers := arr(init, "mcp_servers")
	if servers == nil {
		return nil, false
	}
	var tools []string
	for _, x := range arr(init, "tools") {
		if s, ok := x.(string); ok {
			tools = append(tools, s)
		}
	}
	var out []core.McpServerStatus
	for _, s := range servers {
		sm := asObj(s)
		name := str(sm, "name")
		if name == "" {
			continue
		}
		prefix := "mcp__" + core.McpNormName(name) + "__"
		st := core.McpServerStatus{Name: name, Conn: core.ParseMcpConn(str(sm, "status"))}
		for _, tl := range tools {
			if rest, ok := strings.CutPrefix(tl, prefix); ok {
				st.Tools = append(st.Tools, rest)
			}
		}
		out = append(out, st)
	}
	return out, true
}

func shortError(e string) string {
	for _, l := range strings.Split(e, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			if r := []rune(l); len(r) > 120 {
				return string(r[:120]) + "…"
			}
			return l
		}
	}
	return ""
}

// contextTokens 一次 API 调用占用的上下文:输入 + 缓存读 + 缓存写 + 输出(输入侧全为 0 时返回 0)。
func contextTokens(u obj) uint64 {
	in := u64(u, "input_tokens") + u64(u, "cache_read_input_tokens") + u64(u, "cache_creation_input_tokens")
	if in == 0 {
		return 0
	}
	return in + u64(u, "output_tokens")
}

// backgroundTaskID 后台 Bash 的任务 id:优先 toolUseResult.backgroundTaskId,其次解析「running in background with ID: xxx」。
func backgroundTaskID(input, tur obj, text string) string {
	if id := str(tur, "backgroundTaskId"); id != "" {
		return id
	}
	if boolOf(input, "run_in_background", false) || strings.Contains(text, "running in background with ID:") {
		_, rest, ok := strings.Cut(text, "with ID:")
		if !ok {
			return ""
		}
		f := strings.FieldsFunc(strings.TrimSpace(rest), func(r rune) bool { return r == ' ' || r == '\n' || r == '\t' || r == '.' })
		if len(f) > 0 {
			return strings.TrimRight(f[0], ".")
		}
	}
	return ""
}

// backgroundOutputFile 解析「Output is being written to: <path>.」里的输出文件路径。
func backgroundOutputFile(text string) string {
	_, rest, ok := strings.Cut(text, "written to:")
	if !ok {
		return ""
	}
	rest = strings.TrimSpace(rest)
	i := strings.Index(rest, ".output")
	if i < 0 {
		return ""
	}
	return strings.TrimSpace(rest[:i+len(".output")])
}

// parseExitCode 解析「Exit code N」;解析不出按 1。
func parseExitCode(text string) int {
	rest, ok := strings.CutPrefix(text, "Exit code ")
	if !ok {
		return 1
	}
	end := 0
	for end < len(rest) && (rest[end] == '-' || rest[end] >= '0' && rest[end] <= '9') {
		end++
	}
	n := 0
	neg := false
	for i, c := range rest[:end] {
		if i == 0 && c == '-' {
			neg = true
			continue
		}
		n = n*10 + int(c-'0')
	}
	if end == 0 {
		return 1
	}
	if neg {
		return -n
	}
	return n
}

// resultText 工具结果内容 → (文本, 图片本地路径列表;落盘失败的为空串)。
func resultText(content any) (string, []string) {
	switch c := content.(type) {
	case string:
		return c, nil
	case []any:
		var texts, images []string
		for _, b := range c {
			bm := asObj(b)
			switch str(bm, "type") {
			case "text":
				texts = append(texts, str(bm, "text"))
			case "image":
				images = append(images, storeImage(bm))
			}
		}
		return strings.Join(texts, "\n"), images
	}
	return "", nil
}

// storeImage 把 base64 图片块落盘到临时目录,返回路径(失败为空串)。同内容只写一次。
func storeImage(block obj) string {
	src := sub(block, "source")
	if str(src, "type") != "base64" {
		return ""
	}
	raw, err := base64.StdEncoding.DecodeString(str(src, "data"))
	if err != nil || len(raw) == 0 {
		return ""
	}
	ext := "png"
	switch str(src, "media_type") {
	case "image/jpeg":
		ext = "jpg"
	case "image/gif":
		ext = "gif"
	case "image/webp":
		ext = "webp"
	}
	sum := sha1.Sum(raw)
	dir := filepath.Join(os.TempDir(), "minicode-img")
	if os.MkdirAll(dir, 0o755) != nil {
		return ""
	}
	p := filepath.Join(dir, hex.EncodeToString(sum[:8])+"."+ext)
	if _, err := os.Stat(p); err != nil {
		if os.WriteFile(p, raw, 0o644) != nil {
			return ""
		}
	}
	return p
}
