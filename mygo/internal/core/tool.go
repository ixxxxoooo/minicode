package core

import "strings"

// ToolKind 是工具的大类,决定图标与卡片样式。
type ToolKind int

const (
	KindRead ToolKind = iota
	KindEdit
	KindWrite
	KindDelete
	KindMove
	KindSearch
	KindExecute
	KindThink
	KindFetch
	KindBrowser
	KindMCP
	KindSubagent // 子 Agent(Task / Agent 工具)
	KindTodo
	KindSwitchMode
	KindOther
)

// KindFromToolName 按 Claude Code 的工具名推断大类。
func KindFromToolName(name string) ToolKind {
	if strings.HasPrefix(name, "mcp__") {
		if strings.Contains(name, "browser") || strings.Contains(name, "playwright") {
			return KindBrowser
		}
		return KindMCP
	}
	switch name {
	case "Read", "NotebookRead":
		return KindRead
	case "Edit", "MultiEdit", "NotebookEdit":
		return KindEdit
	case "Write":
		return KindWrite
	case "Glob", "Grep", "LS", "ToolSearch", "LSP":
		return KindSearch
	case "Bash", "BashOutput", "KillShell", "KillBash", "Await", "Monitor", "TaskOutput", "TaskStop":
		return KindExecute
	case "WebFetch", "WebSearch":
		return KindFetch
	case "Task", "Agent":
		return KindSubagent
	case "TodoWrite":
		return KindTodo
	case "ExitPlanMode", "EnterPlanMode":
		return KindSwitchMode
	case "Think":
		return KindThink
	}
	return KindOther
}

// ToolStatus 是工具执行状态。
type ToolStatus int

const (
	ToolPending ToolStatus = iota
	ToolRunning
	ToolCompleted
	ToolFailed
	ToolDenied      // 用户拒绝了权限请求
	ToolInterrupted // 回合被中断时仍在运行的工具
)

// Finished 是否已结束(成功、失败、被拒绝或被中断)。
func (s ToolStatus) Finished() bool { return s != ToolPending && s != ToolRunning }

// Unsuccessful 是否属于需要提示用户注意的未成功结果。
func (s ToolStatus) Unsuccessful() bool {
	return s == ToolFailed || s == ToolDenied || s == ToolInterrupted
}

// ToolContent 是工具产出的内容块:*TextBlock / *DiffBlock / *TerminalBlock / *ImageBlock。
type ToolContent interface{ toolContent() }

// TextBlock 是纯文本或带 ANSI 转义的输出。
type TextBlock struct{ Text string }

// DiffBlock 是文件改动。
type DiffBlock struct{ Path, Old, New string }

// TerminalBlock 是终端命令与输出;ExitCode 为 nil 表示命令还在运行(实时输出)。
type TerminalBlock struct {
	Command, Output string
	ExitCode        *int
}

// ImageBlock 是图片(演示用占位说明,真实模式存文件路径)。
type ImageBlock struct{ Label, Path string }

func (*TextBlock) toolContent()     {}
func (*DiffBlock) toolContent()     {}
func (*TerminalBlock) toolContent() {}
func (*ImageBlock) toolContent()    {}

// Int 取地址的小工具,用来写 `ExitCode: core.Int(0)`。
func Int(v int) *int { return &v }

// ToolCall 是一次工具调用。
type ToolCall struct {
	ID         string
	Name       string // Claude Code 原始工具名,如 Bash、mcp__github__get_pr
	Kind       ToolKind
	Title      string // 一行摘要,如「读取 src/main.rs」
	Status     ToolStatus
	Input      map[string]any // 原始入参
	Content    []ToolContent
	Locations  []string // 涉及的文件路径(用于「跟随」与跳转)
	DurationMs int64    // 耗时毫秒,0 = 未知
	TaskID     string   // 后台任务 id:后台 Bash 或 Monitor 启动后,用它关联 State.Tasks
}

// NewToolCall 创建一个运行中的工具调用,大类由工具名推断。
func NewToolCall(id, name, title string) *ToolCall {
	return &ToolCall{ID: id, Name: name, Kind: KindFromToolName(name), Title: title, Status: ToolRunning}
}

// WithInput 链式设置入参。
func (t *ToolCall) WithInput(in map[string]any) *ToolCall { t.Input = in; return t }

// WithContent 链式追加内容块。
func (t *ToolCall) WithContent(c ToolContent) *ToolCall { t.Content = append(t.Content, c); return t }

// WithLocation 链式追加涉及的文件。
func (t *ToolCall) WithLocation(p string) *ToolCall { t.Locations = append(t.Locations, p); return t }

// ToolPatch 是工具调用的增量更新,零值字段表示不变。
type ToolPatch struct {
	Title         string
	Status        *ToolStatus
	AppendContent []ToolContent
	DurationMs    int64
	TaskID        string
	// LiveOutput 是运行中命令的实时输出(整体替换,不是追加):界面把它当作还没结束的终端块显示;
	// 命令结束后由最终结果取代。nil 表示本次没有实时输出。
	LiveOutput *string
}

// Status 取地址的小工具。
func Status(s ToolStatus) *ToolStatus { return &s }

// Str 取地址的小工具。
func Str(s string) *string { return &s }

// Apply 应用增量更新。
func (t *ToolCall) Apply(p ToolPatch) {
	if p.Title != "" {
		t.Title = p.Title
	}
	if p.Status != nil {
		t.Status = *p.Status
	}
	// 最终结果到了:实时输出(没有退出码的终端块)让位给它
	for _, c := range p.AppendContent {
		if _, ok := c.(*TerminalBlock); ok {
			kept := t.Content[:0:0]
			for _, old := range t.Content {
				if tb, ok := old.(*TerminalBlock); ok && tb.ExitCode == nil {
					continue
				}
				kept = append(kept, old)
			}
			t.Content = kept
			break
		}
	}
	if p.LiveOutput != nil {
		var live *TerminalBlock
		for _, c := range t.Content {
			if tb, ok := c.(*TerminalBlock); ok && tb.ExitCode == nil {
				live = tb
				break
			}
		}
		if live != nil {
			live.Output = *p.LiveOutput
		} else {
			cmd, _ := t.Input["command"].(string)
			t.Content = append(t.Content, &TerminalBlock{Command: cmd, Output: *p.LiveOutput})
		}
	}
	t.Content = append(t.Content, p.AppendContent...)
	if p.DurationMs != 0 {
		t.DurationMs = p.DurationMs
	}
	if p.TaskID != "" {
		t.TaskID = p.TaskID
	}
}
