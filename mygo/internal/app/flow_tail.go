package app

import (
	"time"

	"github.com/ixxxxoooo/minicode/internal/claude"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/diag"
)

// liveTermOut 合并工具卡内的终端输出与关联后台任务的 output_file 尾读缓存。
func (a *App) liveTermOut(toolID string, tb *core.TerminalBlock) string {
	out := tb.Output
	if a.termLive == nil {
		return out
	}
	if extra, ok := a.termLive[toolID]; ok && extra != "" {
		if out == "" {
			return extra
		}
		return out + extra
	}
	return out
}

// syncTaskOutput 任务 output 增量写入缓存(由 afterEvent 调用)。
func (a *App) syncTaskOutput(taskID, text string) {
	if a.termLive == nil {
		a.termLive = map[string]string{}
	}
	for _, it := range a.st.Items {
		tc, ok := it.(*core.ToolCall)
		if !ok || tc.TaskID != taskID {
			continue
		}
		a.termLive[tc.ID] = text
		return
	}
}

// pollTaskOutputFiles 从磁盘尾读关联 output_file(真实模式前台 / 后台命令)。
func (a *App) pollTaskOutputFiles() {
	for _, t := range a.st.Tasks {
		if t.Status != core.TaskRunning || t.OutputFile == "" {
			continue
		}
		path, id := t.OutputFile, t.TaskID
		a.bg("task.tail", func() func() {
			end := diag.SpanSlow("app", "尾读任务输出", 250*time.Millisecond, "task", id)
			raw, err := claude.ReadTail(path, core.LogCap)
			end()
			if err != nil {
				return nil
			}
			text := claude.TidyTaskLog(raw)
			return func() { a.syncTaskOutput(id, text) }
		})
	}
}
