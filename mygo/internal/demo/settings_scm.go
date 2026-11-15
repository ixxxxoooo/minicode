package demo

// WorktreeRow 是 worktree 列表一行。
type WorktreeRow struct {
	Branch, Path string
}

// DemoWorktrees 演示 worktree 列表。
func DemoWorktrees() []WorktreeRow {
	return []WorktreeRow{{Branch: "feat/auth", Path: "~/.minicode/worktrees/router-demo-auth"}}
}
