package core

import "testing"

func TestDockGroupOf(t *testing.T) {
	ag := NewBackgroundTask(0, "a", TaskAgent, "x")
	if DockGroupOf(ag) != DockAgent {
		t.Fatal(DockGroupOf(ag))
	}
	ag.Status = TaskCompleted
	if DockGroupOf(ag) != DockFinished {
		t.Fatal("结束应进已结束")
	}
	sh := NewBackgroundTask(0, "s", TaskMonitor, "watch")
	if DockGroupOf(sh) != DockMonitor {
		t.Fatal(DockGroupOf(sh))
	}
}

func TestGroupTasks(t *testing.T) {
	g := GroupTasks([]*BackgroundTask{
		NewBackgroundTask(0, "a", TaskAgent, "A"),
		NewBackgroundTask(0, "s", TaskShell, "S"),
	})
	if len(g[DockAgent]) != 1 || len(g[DockShell]) != 1 {
		t.Fatalf("%v", g)
	}
}
