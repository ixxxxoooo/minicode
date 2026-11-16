package core

// StopReason 是回合结束原因。
type StopReason int

const (
	StopEndTurn StopReason = iota
	StopCancelled
	StopRefusal
	StopMaxTokens
	StopError // 请求失败(限流、鉴权等),细节见随后的 Notice
)

// Event 是后端产出的事件(*Ev… 指针或值均可)。Parent 为子 Agent 卡片 id,表示该事件属于嵌套时间线。
type Event interface{ event() }

type (
	// EvReady 会话就绪。
	EvReady struct {
		SessionID string
		Models    []ModelInfo
		Model     string
		Mode      PermissionMode
	}
	EvTurnStarted struct{}
	// EvDuration 记录某个条目的用时(秒):思考条目是一次思考的用时,用户消息条目是整个回合的用时。
	EvDuration struct {
		ID   string
		Secs uint64
	}
	// EvUserUUID 用户消息条目对应的 claude 消息 uuid(文件检查点按它记录,回退时用)。
	EvUserUUID struct{ ID, UUID string }
	// EvRewind 文件回退请求的结果(预览或实际执行)。
	EvRewind struct{ Outcome RewindOutcome }
	// EvAppend 追加一个条目。
	EvAppend struct {
		Parent string
		Item   Item
	}
	EvTextDelta struct {
		Parent, ID, Delta string
	}
	EvThinkingDelta struct {
		Parent, ID, Delta string
	}
	EvToolUpdate struct {
		Parent, ID string
		Patch      ToolPatch
	}
	// EvItemDone 条目流式结束(文本 / 思考)。
	EvItemDone struct{ Parent, ID string }
	// EvSubagentStatus 子 Agent 状态变更。
	EvSubagentStatus struct {
		ID     string
		Status SubagentStatus
	}
	// EvSubagentFinished 子 Agent 结束:状态、结果摘要与统计。
	EvSubagentFinished struct {
		ID         string
		Status     SubagentStatus
		Summary    string
		Tokens     uint64
		ToolCount  int
		DurationMs int64 // 0 = 未知
	}
	// EvTodos 待办全量替换。
	EvTodos struct {
		ID    string
		Items []TodoItem
	}
	// EvPromoteToPermission 把已有的同 id 工具卡升级成权限请求;找不到时直接追加。
	EvPromoteToPermission struct {
		Parent  string
		Request *PermissionRequest
	}
	EvPermissionResolved struct{ ID, Option string }
	EvQuestionAnswered   struct {
		ID      string
		Answers [][]string
	}
	EvPlanResolved struct {
		ID       string
		Approved bool
	}
	// EvPlanUpdated 计划正文更新(流式生成中 Streaming=true;最终定稿 false)。
	EvPlanUpdated struct {
		ID, Markdown string
		Streaming    bool
	}
	EvUsage        struct{ Usage Usage }
	EvMcpServers   struct{ Servers []McpServerStatus }
	EvModeChanged  struct{ Mode PermissionMode }
	EvModelChanged struct{ Model string }
	EvTitleChanged struct{ Title string }
	EvNotice       struct {
		Level NoticeLevel
		Text  string
	}
	// EvTaskStarted 后台任务启动;同一 task_id 再次启动视为恢复运行。
	EvTaskStarted struct{ Task *BackgroundTask }
	// EvTaskProgress 后台任务进度,nil / 空字段表示不变。
	EvTaskProgress struct {
		TaskID            string
		Tokens            *uint64
		ToolUses          *int
		LastTool, Summary string
	}
	// EvTaskUpdated 后台任务状态补丁。
	EvTaskUpdated struct {
		TaskID      string
		Status      *TaskStatus
		Description string
		OutputFile  string
	}
	// EvTaskNotification 后台任务终态通知,这是结束的权威信号。
	EvTaskNotification struct {
		TaskID     string
		Status     TaskStatus
		Summary    string
		OutputFile string
	}
	// EvTasksChanged 后台任务全量快照:用它替换缓存,漏掉的事件会被纠正。
	EvTasksChanged struct{ Tasks []*BackgroundTask }
	EvTaskOutput   struct{ TaskID, Text string }
	// EvTurnStats 回合消耗统计;ID 为用户消息 id,空表示最近一条用户消息。
	EvTurnStats struct {
		ID    string
		Stats TurnStats
	}
	EvTurnEnded struct{ Reason StopReason }
)

func (EvReady) event()               {}
func (EvTurnStarted) event()         {}
func (EvDuration) event()            {}
func (EvUserUUID) event()            {}
func (EvRewind) event()              {}
func (EvAppend) event()              {}
func (EvTextDelta) event()           {}
func (EvThinkingDelta) event()       {}
func (EvToolUpdate) event()          {}
func (EvItemDone) event()            {}
func (EvSubagentStatus) event()      {}
func (EvSubagentFinished) event()    {}
func (EvTodos) event()               {}
func (EvPromoteToPermission) event() {}
func (EvPermissionResolved) event()  {}
func (EvQuestionAnswered) event()    {}
func (EvPlanResolved) event()        {}
func (EvPlanUpdated) event()         {}
func (EvUsage) event()               {}
func (EvMcpServers) event()          {}
func (EvModeChanged) event()         {}
func (EvModelChanged) event()        {}
func (EvTitleChanged) event()        {}
func (EvNotice) event()              {}
func (EvTaskStarted) event()         {}
func (EvTaskProgress) event()        {}
func (EvTaskUpdated) event()         {}
func (EvTaskNotification) event()    {}
func (EvTasksChanged) event()        {}
func (EvTaskOutput) event()          {}
func (EvTurnStats) event()           {}
func (EvTurnEnded) event()           {}

// Command 是界面发给后端的命令。
type Command interface{ command() }

type (
	// CmdPrompt 发送提示;Steer 为真表示运行中追加的「引导」消息。
	CmdPrompt struct {
		Text          string
		Images, Files []string
		Steer         bool
	}
	CmdCancel            struct{}
	CmdSetModel          struct{ Model string }
	CmdSetMode           struct{ Mode PermissionMode }
	CmdSetEffort         struct{ Effort Effort }
	CmdResolvePermission struct{ ID, Option string }
	CmdAnswerQuestion    struct {
		ID      string
		Answers [][]string
	}
	// CmdResolvePlan 计划批准;批准时 Mode 是之后要切到的权限模式(nil 沿用 claude 的建议)。
	CmdResolvePlan struct {
		ID      string
		Approve bool
		Mode    *PermissionMode
	}
	CmdStopTask struct{ TaskID string }
	// CmdBackgroundCurrent 把前台运行中的 Bash / 子 Agent 转入后台(Ctrl+B);ToolUseID 为空表示全部。
	CmdBackgroundCurrent struct{ ToolUseID string }
	// CmdRefreshUsage 按需查询上下文分类与/或账户限额;界面指定 Context / Limits,不默认全查。
	CmdRefreshUsage struct {
		Force     bool // 限额:越过 LimitsTTLSecs 缓存
		Context   bool // get_context_usage
		Limits    bool // get_usage
		StartProc bool // 惰性后端:为查上下文而启动 claude 进程
	}
	// CmdRefreshMcp 刷新 MCP:界面先重读配置,Runtime 为真且进程在跑时发 mcp_status。
	CmdRefreshMcp struct {
		Runtime   bool // 发 mcp_status(需进程已启动,除非 StartProc)
		StartProc bool // 显式启动进程再查运行时状态
		Force     bool // 由界面防抖后传入,后端只做占位
	}
	// CmdMcpToggle 热切换一个 MCP 服务器。
	CmdMcpToggle struct {
		Name    string
		Enabled bool
	}
	// CmdSetEnv 替换注入 claude 的额外环境变量(切换模型服务商时用)。
	CmdSetEnv struct{ Env []EnvVar }
	// CmdRewindFiles 把 Claude 改过的文件回退到某条用户消息发出之前;DryRun 只预览。
	CmdRewindFiles struct {
		UUID   string
		DryRun bool
	}
	// CmdRunScenario 触发演示场景(仅演示后端识别)。
	CmdRunScenario struct{ ID string }
	// CmdShutdown 关闭后端(结束子进程)。
	CmdShutdown struct{}
)

// EnvVar 是一个环境变量。
type EnvVar struct{ Key, Value string }

func (CmdPrompt) command()            {}
func (CmdCancel) command()            {}
func (CmdSetModel) command()          {}
func (CmdSetMode) command()           {}
func (CmdSetEffort) command()         {}
func (CmdResolvePermission) command() {}
func (CmdAnswerQuestion) command()    {}
func (CmdResolvePlan) command()       {}
func (CmdStopTask) command()          {}
func (CmdBackgroundCurrent) command() {}
func (CmdRefreshUsage) command()      {}
func (CmdRefreshMcp) command()        {}
func (CmdMcpToggle) command()         {}
func (CmdSetEnv) command()            {}
func (CmdRewindFiles) command()       {}
func (CmdRunScenario) command()       {}
func (CmdShutdown) command()          {}
