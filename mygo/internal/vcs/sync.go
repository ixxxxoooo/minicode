package vcs

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PullRebase 拉取并变基(git pull --rebase)。
func (r *CLI) PullRebase() error {
	_, err := r.git(90*time.Second, "pull", "--rebase", "-q")
	return err
}

// PullMerge 拉取并合并(git pull --no-rebase)。
func (r *CLI) PullMerge() error {
	_, err := r.git(90*time.Second, "pull", "--no-rebase", "-q")
	return err
}

// RebaseAbort 中止进行中的变基。
func (r *CLI) RebaseAbort() error {
	_, err := r.git(30*time.Second, "rebase", "--abort")
	return err
}

// MergeAbort 中止进行中的合并。
func (r *CLI) MergeAbort() error {
	_, err := r.git(30*time.Second, "merge", "--abort")
	return err
}

// OpState 读 .git 里的进行中标记:rebase-merge / rebase-apply = 变基,MERGE_HEAD = 合并。
func (r *CLI) OpState() string {
	out, err := r.git(3*time.Second, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return ""
	}
	dir := strings.TrimSpace(out)
	for _, n := range []string{"rebase-merge", "rebase-apply"} {
		if _, err := os.Stat(filepath.Join(dir, n)); err == nil {
			return "rebase"
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "MERGE_HEAD")); err == nil {
		return "merge"
	}
	return ""
}
