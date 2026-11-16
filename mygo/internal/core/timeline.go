package core

import "time"

// Item 是时间线上的一个条目。具体类型都是指针,归约器原地修改它们。
type Item interface{ ItemID() string }

// NoticeLevel 是提示级别。
type NoticeLevel int

const (
	NoticeInfo NoticeLevel = iota
	NoticeWarning
	NoticeError
)

// TodoStatus 是待办状态。
type TodoStatus int

const (
	TodoPending TodoStatus = iota
	TodoInProgress
	TodoCompleted
)

// TodoItem 是一条待办。
type TodoItem struct {
	Content    string
	Status     TodoStatus
	ActiveForm string // 进行中时的动词形式(TodoWrite 的 activeForm);空则界面用 Content
}

// PermissionOptionKind 是权限选项的性质。
type PermissionOptionKind int

const (
	AllowOnce PermissionOptionKind = iota
	AllowAlways
	RejectOnce
	RejectAlways
)

// Rejects 是否是「拒绝」类。
func (k PermissionOptionKind) Rejects() bool { return k == RejectOnce || k == RejectAlways }

// PermissionOption 是权限选项。
type PermissionOption struct {
	ID    string
	Label string
	Kind  PermissionOptionKind
}

// QuestionOption 是提问的候选项。
type QuestionOption struct{ Label, Description string }

// Question 是单个问题(AskUserQuestion 的一题)。
type Question struct {
	Header, Question string
	Options          []QuestionOption
	MultiSelect      bool
}

// SubagentStatus 是子 Agent 状态。
type SubagentStatus int

const (
	SubStarting SubagentStatus = iota
	SubRunning
	SubBackground
	SubCompleted
	SubFailed
	SubStopped // 被用户停止(取消回合或停止任务)
)

// Active 是否仍在运行(含后台)。
func (s SubagentStatus) Active() bool { return s <= SubBackground }

// 条目类型 ───────────────────────────────────────────────────────

// User 是用户消息。
type User struct {
	ID, Text      string
	Images, Files []string
}

// Assistant 是助手正文。
type Assistant struct {
	ID, Text  string
	Streaming bool
}

// Thinking 是思考内容。
type Thinking struct {
	ID, Text  string
	Streaming bool
}

// Todos 是待办清单。
type Todos struct {
	ID    string
	Items []TodoItem
}

// PermissionRequest 是一次权限请求及其结果。
type PermissionRequest struct {
	ID       string
	Tool     *ToolCall
	Options  []PermissionOption
	Resolved string // 已选择的选项 id;空 = 仍在等待
}

// QuestionRequest 是一组提问(向导)及回答。
type QuestionRequest struct {
	ID        string
	Questions []Question
	Answers   [][]string // 每题选中的文本;nil = 尚未回答
	Answered  bool
}

// SubagentCard 是子 Agent 卡片,内部嵌套它自己的时间线。
type SubagentCard struct {
	ID, AgentType, Description string
	Status                     SubagentStatus
	Children                   []Item
	TaskID                     string
	Background                 bool
	StartedMs, EndedMs         int64 // 0 = 未知 / 未结束
	Tokens                     uint64
	ToolCount                  int
	LastTool, Summary          string
	LastActivityMs             int64
}

// NewSubagent 新建子 Agent 卡片。开始时间在卡片进入时间线(Append)时由归约器记录,
// 这样脚本提前构建也不会算错用时。
func NewSubagent(id, agentType, desc string, st SubagentStatus) *SubagentCard {
	return &SubagentCard{ID: id, AgentType: agentType, Description: desc, Status: st, Background: st == SubBackground}
}

// Active 是否仍在运行(含后台)。
func (s *SubagentCard) Active() bool { return s.Status.Active() }

// StampStart 运行中的卡片第一次进入时间线时记录开始时间与最近活动。
func (s *SubagentCard) StampStart(now int64) {
	if s.Active() && s.StartedMs == 0 {
		s.StartedMs, s.LastActivityMs = now, now
	}
}

// Tools 工具调用次数:优先用协议上报的,否则数嵌套时间线里的工具。
func (s *SubagentCard) Tools() int {
	if s.ToolCount > 0 {
		return s.ToolCount
	}
	n := 0
	for _, c := range s.Children {
		if _, ok := c.(*ToolCall); ok {
			n++
		}
	}
	return n
}

// ElapsedMs 已运行时长;没有开始时间返回 false。
func (s *SubagentCard) ElapsedMs(now int64) (int64, bool) {
	if s.StartedMs == 0 {
		return 0, false
	}
	end := now
	if s.EndedMs != 0 {
		end = s.EndedMs
	}
	return max(0, end-s.StartedMs), true
}

// Stalled 是否疑似停滞:仍在运行,且超过 StallMs 没有任何进展。
func (s *SubagentCard) Stalled(now int64) bool {
	return s.Active() && s.LastActivityMs != 0 && now-s.LastActivityMs >= StallMs
}

// PlanApproval 是计划批准卡(ExitPlanMode)。
type PlanApproval struct {
	ID, Markdown string
	Approved     *bool // nil 等待中,true 已批准,false 已拒绝
	Streaming    bool  // 计划正文还在生成:先出现卡片,文字逐步补全,生成完才出现 Build 按钮
}

// Notice 是提示。
type Notice struct {
	ID    string
	Level NoticeLevel
	Text  string
}

// CompactState 压缩上下文的进行状态。
type CompactState int

const (
	CompactRunning CompactState = iota
	CompactDone
	CompactFailed
)

// Compaction 是会话流里「压缩上下文」的独立卡片(设计稿 flow-compact 页):压缩中 / 已完成(前后对比)/ 失败。
type Compaction struct {
	ID        string
	Auto      bool // claude 自动触发(上下文将满);否则是手动 /compact
	State     CompactState
	Pre, Post uint64 // 压缩前后的 token 数;0 = 不知道
	Ms        int64  // claude 报告的压缩用时
	Err       string
	StartedAt time.Time // 界面拿到「开始」事件的时刻(压缩中的已用时)
	DoneAt    time.Time // 转为已完成 / 失败的时刻(对比条的收缩动画)
}

// Canvas 是画布卡片。
type Canvas struct{ ID, Title string }

// ChangedFile 是改动汇总中的一个文件。
type ChangedFile struct {
	Path           string
	Added, Removed int
}

// Changes 是本轮改动汇总。
type Changes struct {
	ID    string
	Files []ChangedFile
}

// TaskNote 是后台任务结束通知(task_notification):完成 / 失败 / 被停止。
type TaskNote struct {
	ID, TaskID, Text string
	Kind             TaskKind
	Status           TaskStatus
}

func (x *User) ItemID() string              { return x.ID }
func (x *Assistant) ItemID() string         { return x.ID }
func (x *Thinking) ItemID() string          { return x.ID }
func (x *Compaction) ItemID() string        { return x.ID }
func (x *ToolCall) ItemID() string          { return x.ID }
func (x *Todos) ItemID() string             { return x.ID }
func (x *SubagentCard) ItemID() string      { return x.ID }
func (x *PermissionRequest) ItemID() string { return x.ID }
func (x *QuestionRequest) ItemID() string   { return x.ID }
func (x *PlanApproval) ItemID() string      { return x.ID }
func (x *Notice) ItemID() string            { return x.ID }
func (x *Canvas) ItemID() string            { return x.ID }
func (x *Changes) ItemID() string           { return x.ID }
func (x *TaskNote) ItemID() string          { return x.ID }

// AwaitsUser 条目是否在等待用户操作(权限 / 提问 / 计划批准)。
func AwaitsUser(it Item) bool {
	switch x := it.(type) {
	case *PermissionRequest:
		return x.Resolved == ""
	case *QuestionRequest:
		return !x.Answered
	case *PlanApproval:
		return x.Approved == nil
	}
	return false
}
