package vcs

import "time"

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
