package vcs

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// gitIn 在目录里跑 git,失败直接让测试失败(第 ok 个参数为真时允许失败)。
func gitIn(t *testing.T, dir string, ok bool, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	if out, err := cmd.CombinedOutput(); err != nil && !ok {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestCLIMergeAbort(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("没有 git")
	}
	dir := t.TempDir()
	gitIn(t, dir, false, "init", "-q", "-b", "main")
	f := filepath.Join(dir, "a.txt")
	write := func(s string) {
		if err := os.WriteFile(f, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("base\n")
	gitIn(t, dir, false, "add", ".")
	gitIn(t, dir, false, "commit", "-q", "-m", "base")
	gitIn(t, dir, false, "checkout", "-q", "-b", "other")
	write("other\n")
	gitIn(t, dir, false, "commit", "-q", "-am", "other")
	gitIn(t, dir, false, "checkout", "-q", "main")
	write("main\n")
	gitIn(t, dir, false, "commit", "-q", "-am", "main")
	gitIn(t, dir, true, "merge", "other") // 预期冲突,允许失败

	r, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := r.OpState(); got != "merge" {
		t.Fatalf("冲突后应处于合并中,得到 %q", got)
	}
	if err := r.MergeAbort(); err != nil {
		t.Fatalf("中止合并: %v", err)
	}
	if got := r.OpState(); got != "" {
		t.Fatalf("中止后应无进行中操作,得到 %q", got)
	}
	b, _ := os.ReadFile(f)
	if string(b) != "main\n" {
		t.Fatalf("中止后应回到合并前的内容,得到 %q", b)
	}
	if err := r.MergeAbort(); err == nil {
		t.Fatal("没有进行中的合并时中止应报错")
	}
}

func TestMemMergeAbort(t *testing.T) {
	m := NewMem()
	if err := m.MergeAbort(); err == nil {
		t.Fatal("没有合并时应报错")
	}
	m.SetMergeConflict("合并冲突")
	if err := m.PullMerge(); err == nil {
		t.Fatal("应失败")
	}
	if m.OpState() != "merge" {
		t.Fatalf("应处于合并中: %q", m.OpState())
	}
	if err := m.MergeAbort(); err != nil {
		t.Fatal(err)
	}
	if m.OpState() != "" {
		t.Fatal("中止后应清空")
	}
	for _, c := range func() []Change { cs, _ := m.Status(); return cs }() {
		if c.Status == StatusConflict {
			t.Fatal("冲突条目应被清掉")
		}
	}
}
