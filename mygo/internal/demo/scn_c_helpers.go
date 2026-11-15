package demo

import (
	"fmt"
	"strings"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// permOptsAll 四种权限选项(与设计稿一致)。
func permOptsAll() []core.PermissionOption {
	return []core.PermissionOption{
		{ID: "allow", Label: "允许一次", Kind: core.AllowOnce},
		{ID: "always", Label: "本会话始终允许", Kind: core.AllowAlways},
		{ID: "deny", Label: "拒绝", Kind: core.RejectOnce},
		{ID: "deny_always", Label: "始终拒绝", Kind: core.RejectAlways},
	}
}

func permRejected(opt string) bool {
	switch opt {
	case "deny", "deny_always", "reject", "reject_always":
		return true
	}
	return strings.Contains(opt, "deny") || strings.Contains(opt, "reject")
}

// promotePerm 在时间线里插入权限请求;parent 非空时落在子 Agent 嵌套时间线。
func promotePerm(s *Script, parent, id string, tc *core.ToolCall) {
	s.Add(ms(80), core.EvPromoteToPermission{Parent: parent, Request: &core.PermissionRequest{
		ID: id, Tool: tc, Options: permOptsAll(),
	}})
}

// resolvePerm 处理权限并可选完成工具。
func resolvePerm(s *Script, parent, id, opt string, done bool, content ...core.ToolContent) {
	s.Add(ms(40), core.EvPermissionResolved{ID: id, Option: opt})
	if permRejected(opt) || !done {
		return
	}
	s.Add(ms(120), core.EvToolUpdate{Parent: parent, ID: id, Patch: core.ToolPatch{
		Status: core.Status(core.ToolCompleted), DurationMs: 900, AppendContent: content,
	}})
}

// finishLab 收尾演示回合。
func finishLab(s *Script, uid string, seed int) {
	s.Add(0, core.EvTurnStats{ID: uid, Stats: *DefaultStats(seed)})
	s.Add(0, core.EvDuration{ID: uid, Secs: 12})
	s.Add(0, core.EvTurnEnded{Reason: core.StopEndTurn})
}

func stampSub(sa *core.SubagentCard, ago time.Duration) {
	now := time.Now().UnixMilli()
	ms := now - ago.Milliseconds()
	sa.StartedMs, sa.LastActivityMs = ms, ms
}

func taskAt(kind core.TaskKind, id, desc string, age, idle time.Duration, st core.TaskStatus) *core.BackgroundTask {
	now := time.Now().UnixMilli()
	t := core.NewBackgroundTask(now-age.Milliseconds(), id, kind, desc)
	t.Status = st
	t.LastActivityMs = now - idle.Milliseconds()
	if st.Finished() {
		t.EndedMs = now - idle.Milliseconds()/2
		t.Summary = desc + " · 已结束"
	}
	return t
}

func longPlanMD() string {
	var b strings.Builder
	b.WriteString("## 路由归一化改造\n\n")
	b.WriteString("1. **抽公共函数**\n   - `normalize(path)` 去尾斜杠 + 小写\n   - 注册与匹配共用\n2. **回退策略**\n   - 找不到精确键时用 `fallback`\n3. **测试矩阵**\n\n")
	b.WriteString("```rust\nfn normalize(p: &str) -> String {\n    p.trim_end_matches('/').to_lowercase()\n}\n```\n\n")
	b.WriteString("- [ ] 单元测试\n- [ ] 集成测试\n- [ ] 更新 CHANGELOG\n")
	return b.String()
}

func todoLong30() []core.TodoItem {
	items := make([]core.TodoItem, 0, 30)
	for i := 1; i <= 30; i++ {
		st := core.TodoPending
		act := ""
		switch {
		case i <= 8:
			st = core.TodoCompleted
		case i == 9:
			st, act = core.TodoInProgress, "编写测试用例"
		}
		items = append(items, core.TodoItem{
			Content: fmt.Sprintf("第 %d 项:检查 handlers 分支 %d", i, i%7),
			Status:  st, ActiveForm: act,
		})
	}
	return items
}

func replayTaskDock(s *Script) {
	now := time.Now().UnixMilli()
	add := func(t *core.BackgroundTask) {
		s.Add(0, core.EvTaskStarted{Task: t})
	}
	add(taskAt(core.TaskAgent, "dk-ag1", "审查错误处理", 50*time.Second, 2*time.Second, core.TaskRunning))
	add(taskAt(core.TaskAgent, "dk-ag2", "调研依赖升级", 3*time.Minute, 90*time.Second, core.TaskRunning)) // 停滞
	add(taskAt(core.TaskShell, "dk-sh1", "cargo watch -x test", 40*time.Second, 0, core.TaskRunning))
	add(taskAt(core.TaskShell, "dk-sh2", "npm run build", 2*time.Minute, 0, core.TaskCompleted))
	add(taskAt(core.TaskMonitor, "dk-mon1", "监听 dev.log", 20*time.Second, 5*time.Second, core.TaskRunning))
	add(taskAt(core.TaskWorkflow, "dk-wf1", "发布流水线", 30*time.Second, 0, core.TaskRunning))
	add(taskAt(core.TaskOther, "dk-ot1", "整理截图资产", 10*time.Minute, 0, core.TaskCompleted))
	fail := taskAt(core.TaskShell, "dk-sh3", "docker compose up", time.Minute, 0, core.TaskFailed)
	fail.Summary = "端口 5432 已被占用"
	add(fail)
	stop := taskAt(core.TaskAgent, "dk-ag3", "生成迁移脚本", 5*time.Minute, 0, core.TaskStopped)
	stop.Summary = "用户停止"
	add(stop)
	_ = now
	s.Add(0, core.EvTaskNotification{TaskID: "dk-sh2", Status: core.TaskCompleted, Summary: "构建成功 · 42s"})
	s.Add(0, core.EvTaskNotification{TaskID: "dk-sh3", Status: core.TaskFailed, Summary: fail.Summary})
	s.Add(0, core.EvTaskNotification{TaskID: "dk-ag3", Status: core.TaskStopped, Summary: stop.Summary})
	s.Add(0, core.EvTaskOutput{TaskID: "dk-sh1", Text: "    Finished test [ok] 12 passed\n"})
}
