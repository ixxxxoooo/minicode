package core

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/ixxxxoooo/minicode/internal/diag"
)

var log = diag.L("session")

// InterruptedText 是回合被用户中断时追加的提示文案(界面据此把它画成一行灰字)。
const InterruptedText = "已中断"

// Phase 是会话运行阶段。
type Phase int

const (
	PhaseStarting Phase = iota // 尚未就绪(启动中)
	PhaseIdle                  // 空闲,可以发送
	PhaseRunning               // 回合进行中
)

// State 是一个会话的全部可展示状态。只在界面主线程上读写。
type State struct {
	SessionID string
	Title     string
	Phase     Phase
	Items     []Item
	Models    []ModelInfo
	Model     string
	Mode      PermissionMode
	Effort    Effort
	Usage     Usage
	Mcp       []McpServerStatus // 各 MCP 服务器在本会话里的状态;会话还没启动时为空
	LastStop  *StopReason       // 最近一次回合的结束原因
	// Durations:条目 id → 用时(秒);思考条目是这次思考的用时,用户消息条目是它引出的整个回合的用时。
	Durations map[string]uint64
	// TurnStats:用户消息条目 id → 该回合的消耗统计。
	TurnStats map[string]TurnStats
	// Timings:用户消息条目 id → 本回合从「点发送」到各节点的实测耗时(只有本次运行里发出的回合才有)。
	Timings   map[string]TurnTiming
	TurnAt    time.Time         // 当前回合开始(点发送)的时刻;界面乐观地先置为运行中时也记下
	Live      *LiveStage        // 进行中回合当前所处的阶段(回合结束清空)
	LiveDone  []TimingPhase     // 本回合已结束的阶段(等待行展开时列出,回合结束清空)
	Conn      ConnState         // claude 进程连接状态(演示后端恒为 ConnNone,界面按后端名决定是否显示)
	Tasks     []*BackgroundTask // 后台任务表(按开始顺序);含已结束的
	UserUUIDs map[string]string // 用户消息条目 id → claude 消息 uuid(文件检查点回退用)
	Rewind    *RewindPrompt     // 进行中的文件回退确认(一次只有一条)

	// Rev 每归约一次事件加一:界面据此判断是否需要重建缓存(避免逐项比较)。
	Rev uint64

	// Now 是时钟(unix 毫秒),默认取系统时间;测试里注入固定值,让归约结果确定。
	Now       func() int64
	noticeSeq int
}

// DefaultTitle 是还没有标题的新会话的默认标题。
const DefaultTitle = "新会话"

// TitleFromPrompt 取提问的第一个非空行作标题,最多 40 个字符(超出加省略号);空文本返回空串。
func TitleFromPrompt(text string) string {
	for _, ln := range strings.Split(text, "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		r := []rune(ln)
		if len(r) > 40 {
			return string(r[:40]) + "…"
		}
		return ln
	}
	return ""
}

// NewState 创建空会话。
func NewState() *State {
	return &State{
		Title:     DefaultTitle,
		Models:    DefaultModels(),
		Model:     "sonnet",
		Effort:    EffortMedium,
		Durations: map[string]uint64{},
		TurnStats: map[string]TurnStats{},
		Timings:   map[string]TurnTiming{},
		UserUUIDs: map[string]string{},
		Now:       func() int64 { return time.Now().UnixMilli() },
	}
}

// itemsOf 在条目列表(可能嵌套在子 Agent 内)中定位目标列表。parent 为空或找不到时返回顶层。
func (s *State) itemsOf(parent string) *[]Item {
	if parent == "" {
		return &s.Items
	}
	var find func(items []Item) *[]Item
	find = func(items []Item) *[]Item {
		for _, it := range items {
			if sa, ok := it.(*SubagentCard); ok {
				if sa.ID == parent {
					return &sa.Children
				}
				if r := find(sa.Children); r != nil {
					return r
				}
			}
		}
		return nil
	}
	if r := find(s.Items); r != nil {
		return r
	}
	// 旧版这里静默落到顶层;现在记下来,事件错位时能从日志里看到
	diag.NewCounter("session.parent_missing").Inc()
	log.Warn("找不到父子 Agent 卡片,事件落到顶层", "parent", parent)
	return &s.Items
}

// Find 递归按 id 查找条目(含子 Agent 内部),从尾部往前找(活跃条目通常在末尾)。
func (s *State) Find(id string) Item {
	var find func(items []Item) Item
	find = func(items []Item) Item {
		for i := len(items) - 1; i >= 0; i-- {
			if items[i].ItemID() == id {
				return items[i]
			}
			if sa, ok := items[i].(*SubagentCard); ok {
				if r := find(sa.Children); r != nil {
					return r
				}
			}
		}
		return nil
	}
	return find(s.Items)
}

// RowOf 返回包含指定 id 的顶层行下标(含子 Agent 内部的条目与权限里的工具),用于局部重新测量行高;没有返回 -1。
func (s *State) RowOf(id string) int {
	var contains func(it Item) bool
	contains = func(it Item) bool {
		if it.ItemID() == id {
			return true
		}
		switch x := it.(type) {
		case *SubagentCard:
			for i := len(x.Children) - 1; i >= 0; i-- {
				if contains(x.Children[i]) {
					return true
				}
			}
		case *PermissionRequest:
			return x.Tool != nil && x.Tool.ID == id
		}
		return false
	}
	for i := len(s.Items) - 1; i >= 0; i-- {
		if contains(s.Items[i]) {
			return i
		}
	}
	return -1
}

// AwaitingUser 是否有等待用户操作的条目。
func (s *State) AwaitingUser() bool {
	var any func(items []Item) bool
	any = func(items []Item) bool {
		for _, it := range items {
			if sa, ok := it.(*SubagentCard); ok {
				if any(sa.Children) {
					return true
				}
			} else if AwaitsUser(it) {
				return true
			}
		}
		return false
	}
	return any(s.Items)
}

// touch 记录子 Agent 卡片有了新活动(用于疑似停滞判断)。
func (s *State) touch(cardID string) {
	if cardID == "" {
		return
	}
	if c, ok := s.Find(cardID).(*SubagentCard); ok {
		c.LastActivityMs = s.Now()
	}
}

func (s *State) nextNoticeID() string {
	s.noticeSeq++
	return fmt.Sprintf("notice-%d", s.noticeSeq)
}

// findRev 在列表里从尾部找 id 相同的条目下标。
func findRev(list []Item, id string) int {
	for i := len(list) - 1; i >= 0; i-- {
		if list[i].ItemID() == id {
			return i
		}
	}
	return -1
}

// Apply 归约一个事件。
func (s *State) Apply(ev Event) {
	s.Rev++
	switch e := ev.(type) {
	case EvReady:
		s.SessionID = e.SessionID
		if len(e.Models) > 0 {
			s.Models = e.Models
		}
		s.Model, s.Mode = e.Model, e.Mode
		// 惰性启动时握手应答晚于第一条提示的「回合开始」到达,不能把进行中的回合打回空闲
		if s.Phase != PhaseRunning {
			s.Phase = PhaseIdle
		}
	case EvConn:
		s.Conn = e.State
	case EvTurnStarted:
		if s.Phase != PhaseRunning || s.TurnAt.IsZero() {
			s.TurnAt = time.Now()
		}
		s.Phase = PhaseRunning
	case EvTurnPhase:
		if e.At.IsZero() { // 演示后端不预知时刻:按归约时的墙钟记
			e.At = time.Now()
		}
		if e.TurnStart.IsZero() {
			e.TurnStart = e.At
			if s.Live != nil {
				e.TurnStart = s.Live.TurnStart
			}
		}
		if s.Live == nil || !s.Live.TurnStart.Equal(e.TurnStart) { // 新回合:清掉上一回合的阶段记录
			s.LiveDone = nil
		} else if ms := e.At.Sub(s.Live.Since).Milliseconds(); ms > 0 && s.Live.Kind != e.Kind {
			s.LiveDone = append(s.LiveDone, TimingPhase{Kind: s.Live.Kind, Ms: ms, Call: s.Live.Call})
		}
		cache := e.Cache
		if cache == nil && s.Live != nil && s.Live.Call == e.Call {
			cache = s.Live.Cache // 同一次调用内保留第一个字节带来的缓存用量
		}
		s.Live = &LiveStage{Kind: e.Kind, Since: e.At, TurnStart: e.TurnStart, Call: e.Call, Cache: cache}
	case EvDuration:
		s.Durations[e.ID] = e.Secs
	case EvTurnStats:
		uid := e.ID
		if uid == "" {
			for i := len(s.Items) - 1; i >= 0; i-- {
				if u, ok := s.Items[i].(*User); ok {
					uid = u.ID
					break
				}
			}
		}
		if uid != "" && !e.Stats.Empty() {
			s.TurnStats[uid] = e.Stats
		}
	case EvTurnTiming:
		uid := e.ID
		if uid == "" {
			for i := len(s.Items) - 1; i >= 0; i-- {
				if u, ok := s.Items[i].(*User); ok {
					uid = u.ID
					break
				}
			}
		}
		if uid != "" && e.Timing.TotalMs > 0 {
			if s.Timings == nil {
				s.Timings = map[string]TurnTiming{}
			}
			s.Timings[uid] = e.Timing
		}
	case EvCompact:
		s.applyCompact(e)
	case EvUserUUID:
		s.UserUUIDs[e.ID] = e.UUID
	case EvRewind:
		s.rewindResult(e.Outcome)
	case EvAppend:
		// 子 Agent 卡片进入时间线时记录开始时间;嵌套条目的到来算作父卡片的一次活动
		if c, ok := e.Item.(*SubagentCard); ok {
			c.StampStart(s.Now())
		}
		if u, ok := e.Item.(*User); ok && e.Parent == "" && s.Title == DefaultTitle {
			if t := TitleFromPrompt(u.Text); t != "" {
				s.Title = t // 后端没给标题时,用第一条提问当标题(与侧栏会话行一致)
			}
		}
		s.touch(e.Parent)
		l := s.itemsOf(e.Parent)
		*l = append(*l, e.Item)
	case EvTextDelta:
		s.touch(e.Parent)
		l := s.itemsOf(e.Parent)
		if i := findRev(*l, e.ID); i >= 0 {
			if a, ok := (*l)[i].(*Assistant); ok {
				a.Text += e.Delta
				return
			}
		}
		*l = append(*l, &Assistant{ID: e.ID, Text: e.Delta, Streaming: true})
	case EvThinkingDelta:
		s.touch(e.Parent) // 旧版漏了这一步:只在思考的子 Agent 会被误判为停滞
		l := s.itemsOf(e.Parent)
		if i := findRev(*l, e.ID); i >= 0 {
			if t, ok := (*l)[i].(*Thinking); ok {
				t.Text += e.Delta
				return
			}
		}
		*l = append(*l, &Thinking{ID: e.ID, Text: e.Delta, Streaming: true})
	case EvToolUpdate:
		s.touch(e.Parent)
		l := *s.itemsOf(e.Parent)
		for i := len(l) - 1; i >= 0; i-- {
			switch x := l[i].(type) {
			case *ToolCall:
				if x.ID == e.ID {
					x.Apply(e.Patch)
					return
				}
			case *PermissionRequest:
				if x.Tool != nil && x.Tool.ID == e.ID {
					x.Tool.Apply(e.Patch)
					return
				}
			}
		}
		diag.NewCounter("session.tool_update_orphan").Inc()
		log.Debug("工具更新找不到目标", "id", e.ID, "parent", e.Parent)
	case EvItemDone:
		l := *s.itemsOf(e.Parent)
		if i := findRev(l, e.ID); i >= 0 {
			switch x := l[i].(type) {
			case *Assistant:
				x.Streaming = false
			case *Thinking:
				x.Streaming = false
			}
		}
	case EvSubagentStatus:
		now := s.Now()
		if c, ok := s.Find(e.ID).(*SubagentCard); ok {
			c.Status = e.Status
			if c.Active() {
				c.StampStart(now)
				c.EndedMs = 0
				c.Background = e.Status == SubBackground
			} else if c.EndedMs == 0 {
				c.EndedMs = now
			}
			c.LastActivityMs = now
		}
	case EvSubagentFinished:
		now := s.Now()
		if c, ok := s.Find(e.ID).(*SubagentCard); ok {
			c.Status = e.Status
			c.EndedMs = now
			if e.DurationMs > 0 {
				c.StartedMs = max(1, now-e.DurationMs)
			}
			if e.Tokens > 0 {
				c.Tokens = e.Tokens
			}
			if e.ToolCount > 0 {
				c.ToolCount = e.ToolCount
			}
			if e.Summary != "" {
				c.Summary = e.Summary
			}
		}
	case EvTodos:
		if t, ok := s.Find(e.ID).(*Todos); ok {
			t.Items = e.Items
		} else {
			s.Items = append(s.Items, &Todos{ID: e.ID, Items: e.Items})
		}
	case EvPromoteToPermission:
		l := s.itemsOf(e.Parent)
		for i := len(*l) - 1; i >= 0; i-- {
			if t, ok := (*l)[i].(*ToolCall); ok && e.Request.Tool != nil && t.ID == e.Request.Tool.ID {
				(*l)[i] = e.Request
				return
			}
		}
		*l = append(*l, e.Request)
	case EvPermissionResolved:
		if p, ok := s.Find(e.ID).(*PermissionRequest); ok {
			// 选了「拒绝」类选项:工具卡标记为被拒绝,与失败区分
			rejected := false
			found := false
			for _, o := range p.Options {
				if o.ID == e.Option {
					rejected, found = o.Kind.Rejects(), true
					break
				}
			}
			if !found {
				rejected = strings.Contains(e.Option, "reject") || strings.Contains(e.Option, "deny")
			}
			if rejected && p.Tool != nil {
				p.Tool.Status = ToolDenied
			}
			p.Resolved = e.Option
		}
	case EvQuestionAnswered:
		if q, ok := s.Find(e.ID).(*QuestionRequest); ok {
			q.Answers, q.Answered = e.Answers, true
		}
	case EvPlanResolved:
		if p, ok := s.Find(e.ID).(*PlanApproval); ok {
			v := e.Approved
			p.Approved, p.Streaming = &v, false
		}
	case EvPlanUpdated:
		if p, ok := s.Find(e.ID).(*PlanApproval); ok {
			p.Markdown, p.Streaming = e.Markdown, e.Streaming
		}
	case EvUsage:
		s.Usage = e.Usage
	case EvMcpServers:
		s.Mcp = e.Servers
	case EvModeChanged:
		s.Mode = e.Mode
	case EvModelChanged:
		s.Model = e.Model
	case EvTitleChanged:
		s.Title = e.Title
	case EvNotice:
		s.Items = append(s.Items, &Notice{ID: s.nextNoticeID(), Level: e.Level, Text: e.Text})
	case EvTaskStarted:
		s.taskStarted(e.Task)
	case EvTaskProgress:
		if t := s.task(e.TaskID); t != nil {
			if e.Tokens != nil {
				t.Tokens = *e.Tokens
			}
			if e.ToolUses != nil {
				t.ToolUses = *e.ToolUses
			}
			if e.LastTool != "" {
				t.LastTool = e.LastTool
			}
			if e.Summary != "" {
				t.Summary = e.Summary
			}
			t.LastActivityMs = s.Now()
			s.syncCard(t)
		}
	case EvTaskUpdated:
		if t := s.task(e.TaskID); t != nil {
			if e.Status != nil {
				t.Status = *e.Status
				if t.Status.Finished() && t.EndedMs == 0 {
					t.EndedMs = s.Now()
				}
			}
			if e.Description != "" {
				t.Description = e.Description
			}
			if e.OutputFile != "" {
				t.OutputFile = e.OutputFile
			}
			t.LastActivityMs = s.Now()
			s.syncCard(t)
		}
	case EvTaskNotification:
		s.taskNotified(e)
	case EvTasksChanged:
		s.tasksChanged(e.Tasks)
	case EvTaskOutput:
		if t := s.task(e.TaskID); t != nil {
			t.PushOutput(s.Now(), e.Text)
		}
	case EvForkSession:
		// 分叉由界面切换会话处理,状态机不归约
	case EvTurnEnded:
		s.Phase = PhaseIdle
		s.Live, s.LiveDone, s.TurnAt = nil, nil, time.Time{}
		r := e.Reason
		s.LastStop = &r
		cancelled := r == StopCancelled
		s.finishStreaming(cancelled)
		if cancelled {
			s.markInterrupted()
		}
	default:
		diag.NewCounter("session.unknown_event").Inc()
		log.Warn("未知事件类型", "type", fmt.Sprintf("%T", ev))
	}
}

// markInterrupted 回合被用户中断时在末尾追加一行「已中断」提示(对标 Claude Code 的 Interrupted 标记),
// 否则正文流式中途被打断时时间线看起来像正常结束。连续中断只留一行。
func (s *State) markInterrupted() {
	if n, ok := lastOf(s.Items).(*Notice); ok && n.Text == InterruptedText {
		return
	}
	s.Items = append(s.Items, &Notice{ID: s.nextNoticeID(), Level: NoticeInfo, Text: InterruptedText})
}

func lastOf(items []Item) Item {
	if len(items) == 0 {
		return nil
	}
	return items[len(items)-1]
}

// finishStreaming 回合结束时收尾:停止所有流式条目;取消时把未完成的工具标为被中断。
func (s *State) finishStreaming(cancelled bool) {
	var walk func(items []Item)
	walk = func(items []Item) {
		for _, it := range items {
			switch x := it.(type) {
			case *Assistant:
				x.Streaming = false
			case *Thinking:
				x.Streaming = false
			case *Compaction:
				if x.State == CompactRunning { // 回合结束了还没有结果:取消 / 没有完成,不能一直转
					x.State, x.DoneAt = CompactFailed, time.Now()
					if cancelled {
						x.Err = "已取消"
					} else {
						x.Err = "压缩没有完成(没有收到结果)"
					}
				}
			case *ToolCall:
				if cancelled && !x.Status.Finished() {
					x.Status = ToolInterrupted
				}
			case *PermissionRequest:
				// 旧版漏了这里:取消时权限卡里还在等待的工具永远停在运行态
				if cancelled && x.Tool != nil && !x.Tool.Status.Finished() {
					x.Tool.Status = ToolInterrupted
				}
			case *SubagentCard:
				// 只有前台子 Agent 会随回合一起被中断;后台的继续跑,等 task_notification 给出终态
				if cancelled && !x.Background && (x.Status == SubStarting || x.Status == SubRunning) {
					x.Status = SubStopped
					if x.EndedMs == 0 {
						x.EndedMs = s.Now()
					}
				}
				walk(x.Children)
			}
		}
	}
	walk(s.Items)
}

// ───────── 后台任务 ─────────

func (s *State) task(id string) *BackgroundTask {
	for _, t := range s.Tasks {
		if t.TaskID == id {
			return t
		}
	}
	return nil
}

// Task 后台任务按 id 查找。
func (s *State) Task(id string) *BackgroundTask { return s.task(id) }

// RunningTasks 正在后台运行的任务(不含环境任务与子 Agent 私有任务)。
func (s *State) RunningTasks() []*BackgroundTask {
	var out []*BackgroundTask
	for _, t := range s.Tasks {
		if t.CountsAsRunning() {
			out = append(out, t)
		}
	}
	return out
}

// RunningTaskCount 用于任务坞、状态栏与会话行的计数。
func (s *State) RunningTaskCount() int { return len(s.RunningTasks()) }

// syncCard 把任务的进度同步到启动它的子 Agent 卡片与后台工具卡。
func (s *State) syncCard(t *BackgroundTask) {
	if t.ToolUseID == "" {
		return
	}
	switch c := s.Find(t.ToolUseID).(type) {
	case *SubagentCard:
		c.TaskID = t.TaskID
		c.Background = t.Backgrounded
		c.Tokens = t.Tokens
		if t.ToolUses > 0 {
			c.ToolCount = t.ToolUses
		}
		if t.LastTool != "" {
			c.LastTool = t.LastTool
		}
		if t.Summary != "" {
			c.Summary = t.Summary
		}
		c.LastActivityMs = t.LastActivityMs
		switch {
		case t.Status == TaskRunning && t.Backgrounded:
			c.Status = SubBackground
		case t.Status == TaskRunning:
			c.Status = SubRunning
		case t.Status == TaskCompleted:
			c.Status = SubCompleted
		case t.Status == TaskFailed:
			c.Status = SubFailed
		default:
			c.Status = SubStopped
		}
		if t.Status.Finished() {
			c.EndedMs = t.EndedMs
			if c.EndedMs == 0 {
				c.EndedMs = s.Now()
			}
		} else {
			c.EndedMs = 0
		}
	case *ToolCall:
		c.TaskID = t.TaskID
	}
}

func (s *State) taskStarted(t *BackgroundTask) {
	if t == nil {
		return
	}
	if old := s.task(t.TaskID); old != nil {
		// 同一个 task_id 再次启动(恢复运行):整体替换,保留输出
		t.Output = old.Output
		*old = *t
		s.syncCard(old)
		return
	}
	s.Tasks = append(s.Tasks, t)
	s.syncCard(t)
}

func (s *State) taskNotified(e EvTaskNotification) {
	now := s.Now()
	kind := TaskOther
	visible := true
	wasFinished := false
	if t := s.task(e.TaskID); t != nil {
		wasFinished = t.Status.Finished()
		t.Status = e.Status
		if t.EndedMs == 0 {
			t.EndedMs = now
		}
		if e.Summary != "" {
			t.Summary = e.Summary
		}
		if e.OutputFile != "" {
			t.OutputFile = e.OutputFile
		}
		kind = t.Kind
		visible = t.Visible() && t.Backgrounded
		s.syncCard(t)
	} else if e.Summary == "" {
		// 恢复会话时 claude 会为上个进程留下的任务补一条空的 stopped 通知,没有信息量
		return
	}
	// 子 Agent 的状态已经在它自己的卡片上,不再重复插一行通知(否则会把并行派发的卡片拆开)
	if kind == TaskAgent {
		if t := s.task(e.TaskID); t != nil && t.ToolUseID != "" {
			if _, ok := s.Find(t.ToolUseID).(*SubagentCard); ok {
				return
			}
		}
	}
	// 只有任务此前已是终态时才去重;恢复运行后的再次完成应当再出一条通知
	dup := false
	if wasFinished {
		for i := len(s.Items) - 1; i >= 0 && i >= len(s.Items)-6; i-- {
			if n, ok := s.Items[i].(*TaskNote); ok && n.TaskID == e.TaskID && n.Status == e.Status {
				dup = true
				break
			}
		}
	}
	if visible && !dup {
		text := e.Summary
		if text == "" {
			text = kind.Label() + e.Status.Label()
		}
		s.Items = append(s.Items, &TaskNote{ID: fmt.Sprintf("tasknote-%s-%d", e.TaskID, len(s.Items)), TaskID: e.TaskID, Kind: kind, Status: e.Status, Text: text})
	}
}

func (s *State) tasksChanged(list []*BackgroundTask) {
	now := s.Now()
	live := make([]string, 0, len(list))
	for _, in := range list {
		live = append(live, in.TaskID)
		old := s.task(in.TaskID)
		if old == nil {
			s.Tasks = append(s.Tasks, in)
			s.syncCard(in)
			continue
		}
		// 快照只带描述与标记,进度、输出沿用已有的
		if in.Description != "" {
			old.Description = in.Description
		}
		old.Ambient, old.Backgrounded, old.OwnedBySub = in.Ambient, in.Backgrounded, in.OwnedBySub
		if in.RunID != "" {
			old.RunID = in.RunID
		}
		if in.ToolUseID != "" {
			old.ToolUseID = in.ToolUseID
		}
		if old.Status.Finished() && in.Status == TaskRunning {
			// 已结束的任务又出现在存活集合里:恢复运行
			old.Status, old.EndedMs = TaskRunning, 0
		}
		s.syncCard(old)
	}
	// 快照里没有、但还在后台跑的任务:已经结束(通知漏了也能被纠正)
	for _, t := range s.Tasks {
		if t.Backgrounded && t.Status == TaskRunning && !slices.Contains(live, t.TaskID) {
			t.Status, t.EndedMs = TaskStopped, now
			s.syncCard(t)
		}
	}
}

// applyCompact 归约压缩事件:复用时间线末尾进行中的卡片,没有就新建。
func (s *State) applyCompact(e EvCompact) {
	now := time.Now()
	var cur *Compaction
	for i := len(s.Items) - 1; i >= 0; i-- {
		if c, ok := s.Items[i].(*Compaction); ok && c.State == CompactRunning {
			cur = c
			break
		}
		if _, ok := s.Items[i].(*User); ok { // 只看当前回合
			break
		}
	}
	if cur == nil {
		if e.State == CompactRunning || e.State == CompactDone || e.State == CompactFailed {
			s.noticeSeq++
			cur = &Compaction{ID: fmt.Sprintf("cp%d", s.noticeSeq), Auto: e.Auto, StartedAt: now}
			s.Items = append(s.Items, cur)
		}
	}
	if cur == nil {
		return
	}
	if e.Auto {
		cur.Auto = true
	}
	switch e.State {
	case CompactRunning:
		if cur.StartedAt.IsZero() {
			cur.StartedAt = now
		}
	case CompactDone, CompactFailed:
		cur.State, cur.Pre, cur.Post, cur.Ms, cur.Err, cur.DoneAt = e.State, e.Pre, e.Post, e.Ms, e.Err, now
	}
}
