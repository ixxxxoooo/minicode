package vcs

import "testing"

func TestMemPullRebaseConflict(t *testing.T) {
	m := NewMem()
	m.SetRebaseConflict("变基冲突")
	if err := m.PullRebase(); err == nil {
		t.Fatal("应失败")
	}
	if !m.rebase {
		t.Fatal("应进入 rebase 状态")
	}
	if err := m.RebaseAbort(); err != nil {
		t.Fatal(err)
	}
	if m.rebase {
		t.Fatal("中止后应退出 rebase")
	}
}

func TestMemStashFiles(t *testing.T) {
	m := NewMem()
	m.SetStash([]Stash{{Ref: "stash@{0}", Msg: "wip", Files: 1}})
	m.SetStashHeld(0, []Change{{Path: "a.go", Status: 'M'}}, nil)
	fs, err := m.StashFiles(0)
	if err != nil || len(fs) != 1 || fs[0].Path != "a.go" {
		t.Fatalf("stash files: %v err=%v", fs, err)
	}
}
