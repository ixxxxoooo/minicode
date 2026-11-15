package claude

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// TestRealBackgroundTaskAndSubagent 要求模型起后台 Bash 与 Task 子代理,检验 task_* 事件与 State.Tasks / TaskNote。
func TestRealBackgroundTaskAndSubagent(t *testing.T) {
	realSkip(t)
	cwd := t.TempDir()
	mode := core.ModeDefault
	r := newRealRig(t, Config{Mode: &mode, Cwd: cwd})
	t.Cleanup(func() { cleanClaudeSession(t, r, cwd) })

	prompt := `严格按顺序执行,完成后用一句话总结:
1) 用 Bash 工具且 run_in_background=true 执行: sleep 2 && echo done
2) 用 Agent/Task 子代理(尽量用 explore 类型)只做一件事:用 Bash 执行 echo subagent_ok
不要改文件,不要多余解释。`
	r.b.Send(core.CmdPrompt{Text: prompt})
	r.waitTimeout(180*time.Second, "首回合(含后台任务)", func() bool { return r.turnsEndedLocked() >= 1 })
	skipIfLimitReply(t, r)

	r.mu.Lock()
	started := 0
	for _, e := range r.evs {
		if _, ok := e.(core.EvTaskStarted); ok {
			started++
		}
	}
	r.mu.Unlock()
	if started == 0 {
		t.Skip("模型未发起后台任务/子代理(DeepSeek 可能未调用对应工具),跳过")
	}

	// 等待后台 sleep 完成通知(最多 30s)
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		r.mu.Lock()
		done := false
		for _, e := range r.evs {
			if n, ok := e.(core.EvTaskNotification); ok && n.Status == core.TaskCompleted {
				done = true
			}
		}
		tasks := append([]*core.BackgroundTask(nil), r.st.Tasks...)
		r.mu.Unlock()
		if done {
			break
		}
		_ = tasks
		time.Sleep(500 * time.Millisecond)
	}

	r.mu.Lock()
	var notes int
	for _, it := range r.st.Items {
		if _, ok := it.(*core.TaskNote); ok {
			notes++
		}
	}
	if len(r.st.Tasks) == 0 {
		r.mu.Unlock()
		t.Error("State.Tasks 应至少有一条后台任务")
		return
	}
	if notes == 0 {
		t.Log("时间线尚无 TaskNote(通知可能仅在回合末出现,或子代理未走 task_notification)")
	}
	var liveID string
	for _, tk := range r.st.Tasks {
		if tk.Status == core.TaskRunning {
			liveID = tk.TaskID
			break
		}
	}
	r.mu.Unlock()

	if liveID != "" {
		r.b.Send(core.CmdStopTask{TaskID: liveID})
		r.waitTimeout(30*time.Second, "停止任务", func() bool {
			return r.hasEventLocked(func(e core.Event) bool {
				n, ok := e.(core.EvTaskNotification)
				return ok && n.TaskID == liveID && (n.Status == core.TaskStopped || n.Status == core.TaskCompleted)
			})
		})
	}
}

// TestRealForegroundBashOutput 前台 Bash 应在工具结果里带终端输出;若有 output_file 则 ReadTail 可读。
func TestRealForegroundBashOutput(t *testing.T) {
	realSkip(t)
	cwd := t.TempDir()
	md := core.ModeDefault
	r := newRealRig(t, Config{Mode: &md, Cwd: cwd})
	t.Cleanup(func() { cleanClaudeSession(t, r, cwd) })

	r.b.Send(core.CmdPrompt{Text: "用 Bash 前台执行命令: printf 'line1\\nline2\\n' ,然后回复 ok"})
	r.waitTimeout(120*time.Second, "Bash 回合", func() bool { return r.turnsEndedLocked() >= 1 })
	skipIfLimitReply(t, r)

	r.mu.Lock()
	defer r.mu.Unlock()
	var termOut string
	for _, it := range r.st.Items {
		walkTools(it, func(tl *core.ToolCall) {
			if tl.Name != "Bash" || tl.Status != core.ToolCompleted {
				return
			}
			for _, c := range tl.Content {
				if tb, ok := c.(*core.TerminalBlock); ok {
					termOut = tb.Output
				}
			}
		})
	}
	if termOut == "" {
		t.Skip("模型未调用 Bash 或输出为空")
	}
	if !strings.Contains(termOut, "line1") {
		t.Errorf("前台 Bash 输出应包含 line1:%q", termOut)
	}

	for _, tk := range r.st.Tasks {
		if tk.OutputFile == "" {
			continue
		}
		if _, err := ReadTail(tk.OutputFile, 4096); err != nil && !os.IsNotExist(err) {
			t.Errorf("ReadTail output_file: %v", err)
		}
	}
}

func walkTools(it core.Item, f func(*core.ToolCall)) {
	switch x := it.(type) {
	case *core.ToolCall:
		f(x)
	case *core.PermissionRequest:
		if x.Tool != nil {
			f(x.Tool)
		}
	}
}
