package core

import (
	"fmt"
	"strings"
)

// TaskKind 是后台任务类型。
type TaskKind int

const (
	TaskOther    TaskKind = iota
	TaskShell             // 后台 Shell 命令(run_in_background 的 Bash)
	TaskAgent             // 后台子 Agent
	TaskMonitor           // Monitor 持续监听
	TaskWorkflow          // 动态工作流
)

// TaskKindFromType 按 claude 的 task_type 字符串映射。Monitor 在协议里也是 local_bash,需结合描述区分。
func TaskKindFromType(t string) TaskKind {
	switch t {
	case "local_bash", "bash", "shell":
		return TaskShell
	case "local_agent", "agent", "remote_agent":
		return TaskAgent
	case "monitor":
		return TaskMonitor
	case "local_workflow", "workflow":
		return TaskWorkflow
	}
	return TaskOther
}

// Label 中文名。
func (k TaskKind) Label() string {
	switch k {
	case TaskShell:
		return "后台命令"
	case TaskAgent:
		return "后台子代理"
	case TaskMonitor:
		return "监听"
	case TaskWorkflow:
		return "工作流"
	}
	return "后台任务"
}

// TaskStatus 是任务状态。
type TaskStatus int

const (
	TaskRunning TaskStatus = iota
	TaskCompleted
	TaskFailed
	TaskStopped // 被用户或系统停止(killed / stopped)
)

// TaskStatusFromCLI 从协议字符串映射。
func TaskStatusFromCLI(s string) TaskStatus {
	switch s {
	case "completed", "success":
		return TaskCompleted
	case "failed", "error":
		return TaskFailed
	case "stopped", "killed", "cancelled":
		return TaskStopped
	}
	return TaskRunning
}

// Finished 是否已结束。
func (s TaskStatus) Finished() bool { return s != TaskRunning }

// Label 中文名。
func (s TaskStatus) Label() string {
	return [...]string{"运行中", "已完成", "已失败", "已停止"}[s]
}

// OutputCap 是任务输出缓冲上限(字节)。
const OutputCap = 16 * 1024

// LogCap 是读日志文件时最多取末尾多少字节(比内存里的 OutputCap 大得多)。
const LogCap = 96 * 1024

// StallMs 是疑似停滞的阈值:运行中且超过这么久没有任何进展。
const StallMs int64 = 60_000

// BackgroundTask 是一个后台任务。
type BackgroundTask struct {
	TaskID         string
	RunID          string // 同一 task_id 重新运行时每次有新的 run_id
	ToolUseID      string // 启动它的工具调用 id,用来关联时间线上的工具卡或子 Agent 卡
	Kind           TaskKind
	Description    string
	Status         TaskStatus
	StartedMs      int64
	EndedMs        int64 // 0 = 未结束
	Tokens         uint64
	ToolUses       int
	LastTool       string
	LastActivityMs int64
	Summary        string
	OutputFile     string
	Output         string // 最近输出的尾部(最多 OutputCap 字节)
	Ambient        bool   // 不属于会话本身的工作(Claude Code 内部任务、实时监听器),不计入活动指示
	OwnedBySub     bool   // 是子 Agent 自己的前台工具,不算后台工作
	Backgrounded   bool   // 是否在后台运行(前台子 Agent 也有 task_started,但不是后台)
}

// NewBackgroundTask 新建一个正在运行的后台任务;now 由调用方给(归约器注入时钟,便于确定性测试)。
func NewBackgroundTask(now int64, id string, kind TaskKind, desc string) *BackgroundTask {
	return &BackgroundTask{TaskID: id, Kind: kind, Description: desc, StartedMs: now, LastActivityMs: now, Backgrounded: true}
}

// CountsAsRunning 是否计入「后台正在运行」:在跑、确实在后台,且不是环境任务或子 Agent 私有。
func (t *BackgroundTask) CountsAsRunning() bool {
	return t.Status == TaskRunning && t.Backgrounded && !t.Ambient && !t.OwnedBySub
}

// Visible 是否应出现在任务列表里。
func (t *BackgroundTask) Visible() bool { return !t.Ambient && !t.OwnedBySub }

// PushOutput 追加输出,超出上限时丢弃最早的部分(按 UTF-8 字符边界截断)。
func (t *BackgroundTask) PushOutput(now int64, text string) {
	t.Output += text
	if len(t.Output) > OutputCap {
		cut := len(t.Output) - OutputCap
		for cut < len(t.Output) && t.Output[cut]&0xC0 == 0x80 { // 落在续字节上,往后挪到字符起点
			cut++
		}
		t.Output = t.Output[cut:]
	}
	t.LastActivityMs = now
}

// ElapsedMs 已运行时长:结束的用结束时间,运行中的用 now。
func (t *BackgroundTask) ElapsedMs(now int64) int64 {
	end := now
	if t.EndedMs != 0 {
		end = t.EndedMs
	}
	return max(0, end-t.StartedMs)
}

// IdleMs 距上次活动已过去的毫秒数。
func (t *BackgroundTask) IdleMs(now int64) int64 { return max(0, now-t.LastActivityMs) }

// TidyLog 整理终端输出:\r\n 归一,行内的回车覆盖只留最后一段(进度条反复覆盖同一行)。
func TidyLog(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	if !strings.Contains(text, "\r") {
		return text
	}
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		parts := strings.Split(l, "\r")
		pick := ""
		for j := len(parts) - 1; j >= 0; j-- {
			if parts[j] != "" {
				pick = parts[j]
				break
			}
		}
		lines[i] = pick
	}
	return strings.Join(lines, "\n")
}

// FormatElapsed 把毫秒格式化成「12 秒」「3 分 05 秒」「1 小时 02 分」。
func FormatElapsed(ms int64) string {
	s := ms / 1000
	switch {
	case s < 60:
		return fmt.Sprintf("%d 秒", s)
	case s < 3600:
		return fmt.Sprintf("%d 分 %02d 秒", s/60, s%60)
	}
	return fmt.Sprintf("%d 小时 %02d 分", s/3600, s%3600/60)
}

// FormatTokens 把 token 数格式化成「820」「8.2k」「1.3M」。
// (修正旧版缺陷:999 950 之类四舍五入后会写成「1000.0k」,现在进位成「1.0M」。)
func FormatTokens(n uint64) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1_000_000 {
		k := float64(n) / 1000
		if fmt.Sprintf("%.1f", k) != "1000.0" {
			return fmt.Sprintf("%.1fk", k)
		}
	}
	return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
}
