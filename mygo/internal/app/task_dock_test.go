package app

import (
	"testing"

	"github.com/ixxxxoooo/minicode/internal/core"
)

func TestGroupTasksVisible(t *testing.T) {
	st := core.NewState()
	st.Apply(core.EvTaskStarted{Task: core.NewBackgroundTask(0, "a", core.TaskMonitor, "watch")})
	g := core.GroupTasks(st.Tasks)
	if len(g[core.DockMonitor]) != 1 {
		t.Fatalf("%v", g)
	}
}
