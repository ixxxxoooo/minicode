package core

// TaskDockGroup 是任务坞胶囊的分组(与设计稿 tdock 一致)。
type TaskDockGroup int

const (
	DockAgent TaskDockGroup = iota
	DockShell
	DockMonitor
	DockWorkflow
	DockOther
	DockFinished
)

// Key 是弹层键后缀(tpop:<key>)。
func (g TaskDockGroup) Key() string {
	return [...]string{"agent", "cmd", "monitor", "workflow", "other", "done"}[g]
}

// Label 是胶囊文案前缀。
func (g TaskDockGroup) Label() string {
	return [...]string{"子代理", "后台命令", "监听", "工作流", "其他", "已结束"}[g]
}

// Icon 是胶囊图标 id。
func (g TaskDockGroup) Icon() string {
	return [...]string{"bot", "terminal", "eye", "layers", "activity", "check"}[g]
}

// DockGroupOf 把一个后台任务归到任务坞的某一类。
func DockGroupOf(t *BackgroundTask) TaskDockGroup {
	if t == nil {
		return DockOther
	}
	if t.Status.Finished() {
		return DockFinished
	}
	switch t.Kind {
	case TaskAgent:
		return DockAgent
	case TaskShell:
		return DockShell
	case TaskMonitor:
		return DockMonitor
	case TaskWorkflow:
		return DockWorkflow
	}
	return DockOther
}

// GroupTasks 按任务坞分组;只含应出现在列表里的任务(Visible)。
func GroupTasks(tasks []*BackgroundTask) map[TaskDockGroup][]*BackgroundTask {
	out := map[TaskDockGroup][]*BackgroundTask{}
	for _, tk := range tasks {
		if tk == nil || !tk.Visible() {
			continue
		}
		g := DockGroupOf(tk)
		out[g] = append(out[g], tk)
	}
	for g := range out {
		slice := out[g]
		sortTasks(slice)
		out[g] = slice
	}
	return out
}

func sortTasks(s []*BackgroundTask) {
	// 运行中在前,再按开始时间倒序
	for i := 0; i < len(s); i++ {
		for j := i + 1; j < len(s); j++ {
			si, sj := s[i], s[j]
			ri := si.Status == TaskRunning
			rj := sj.Status == TaskRunning
			if ri == rj && si.StartedMs >= sj.StartedMs {
				continue
			}
			if ri == rj && si.StartedMs < sj.StartedMs {
				s[i], s[j] = sj, si
				continue
			}
			if rj && !ri {
				s[i], s[j] = sj, si
			}
		}
	}
}
