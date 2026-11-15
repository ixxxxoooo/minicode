package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// 真实项目:有 Git 仓库的目录,分支 / 改动 / 文件树都读真实内容;切走后回到非仓库目录则分支为空。
func TestSetProject(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("没有 git")
	}
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-b", "trunk")
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-m", "init")
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("two\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("new\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	a, _ := newTestApp(t)
	a.SetProject(dir)
	if a.projLabel() != filepath.Base(dir) {
		t.Fatalf("项目名 %q", a.projLabel())
	}
	if a.branch != "trunk" {
		t.Fatalf("分支 %q", a.branch)
	}
	files := a.ide.allFiles()
	if len(files) != 2 {
		t.Fatalf("文件 %v", files)
	}
	if head, ok := a.headOfRepo("a.txt"); !ok || head != "one\n" {
		t.Fatalf("HEAD 内容 %q %v", head, ok)
	}
	a.scmTick()
	if got := a.welcomeStatus(); got != "trunk · 2 项未提交改动" {
		t.Fatalf("状态行 %q", got)
	}
	if a.ide.git["a.txt"] != "M" {
		t.Fatalf("树上的状态色 %v", a.ide.git)
	}

	// 不是仓库的目录
	plain := t.TempDir()
	a.SetProject(plain)
	if a.repo != nil || a.branch != "" {
		t.Fatalf("非仓库目录不该有仓库:%v %q", a.repo, a.branch)
	}
	if got := a.welcomeStatus(); got != "不是 Git 仓库" {
		t.Fatalf("状态行 %q", got)
	}
}
