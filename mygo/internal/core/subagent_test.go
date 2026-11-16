package core

import "testing"

func TestSubagentPendingPermissions(t *testing.T) {
	c := NewSubagent("a", "x", "y", SubRunning)
	c.Children = []Item{
		NewToolCall("t1", "Bash", "ls"),
		&PermissionRequest{ID: "t1", Tool: NewToolCall("t1", "Bash", "ls")},
		&PermissionRequest{ID: "t2", Tool: NewToolCall("t2", "Bash", "rm"), Resolved: "allow"},
	}
	if n := c.PendingPermissions(); n != 1 {
		t.Fatalf("待确认=%d", n)
	}
}
