package demo

import (
	"fmt"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

func init() {
	Register(Scenario{ID: "c-task-dock", Title: "任务 · 任务坞全类别", Label: "刚刚", Build: buildTaskDock})
	Register(Scenario{ID: "c-task-live", Title: "任务 · 进度 Live 推进", Label: "刚刚", Live: true, Build: buildTaskLive})
}

func buildTaskDock() []Step {
	s := NewScript("c-task-dock", "任务 · 任务坞全类别")
	uid := s.User("任务坞各类与通知")
	s.Say("打开输入框上方的任务坞,应能看到子代理 / 后台命令 / 监听 / 工作流 / 其他 / 已结束 分组。")
	replayTaskDock(s)
	s.Add(0, core.EvMcpServers{Servers: demoMcpServers()})
	s.Add(0, core.EvUsage{Usage: demoContextUsage()})
	finishLab(s, uid, 30)
	return s.Steps()
}

func buildTaskLive() []Step {
	s := NewScript("c-task-live", "任务 · 进度 Live 推进")
	uid := s.User("观察进度")
	id := "live-sh"
	now := time.Now().UnixMilli()
	t := core.NewBackgroundTask(now, id, core.TaskShell, "npm run build")
	t.Summary = "编译中…"
	s.Add(ms(200), core.EvTaskStarted{Task: t})
	for step := 1; step <= 6; step++ {
		tu := uint64(step * 800)
		tools := step / 2
		s.Add(time.Duration(step)*time.Second, core.EvTaskProgress{TaskID: id, Tokens: &tu, ToolUses: &tools, Summary: fmt.Sprintf("阶段 %d/6", step), LastTool: "tsc"})
		s.Add(time.Duration(step)*time.Second+200*time.Millisecond, core.EvTaskOutput{TaskID: id, Text: fmt.Sprintf("step %d ok\n", step)})
	}
	s.Add(7*time.Second, core.EvTaskUpdated{TaskID: id, Status: func() *core.TaskStatus { st := core.TaskCompleted; return &st }(), Description: "构建完成"})
	s.Add(7*time.Second, core.EvTaskNotification{TaskID: id, Status: core.TaskCompleted, Summary: "构建成功 · 52s"})
	s.Say("后台命令应随时间更新进度。")
	finishLab(s, uid, 31)
	return s.Steps()
}
