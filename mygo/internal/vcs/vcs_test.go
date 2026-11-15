package vcs

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestValidName(t *testing.T) {
	for _, ok := range []string{"main", "feat/auth", "fix-1", "v1.2"} {
		if !ValidName(ok) {
			t.Errorf("%q 应合法", ok)
		}
	}
	for _, bad := range []string{"", "-x", "a b", "a..b", "x.lock", "a/", "/a", "a~1", "a:b", "@"} {
		if ValidName(bad) {
			t.Errorf("%q 应不合法", bad)
		}
	}
}

func TestMem(t *testing.T) {
	m := NewMem()
	if err := m.Switch("origin/dev"); err != nil || m.Current() != "dev" {
		t.Fatalf("切远程分支应建同名本地:%v cur=%s", err, m.Current())
	}
	if m.Create("feat/x") != nil || m.Current() != "feat/x" {
		t.Fatal("创建并切换失败")
	}
	if m.Create("main") == nil {
		t.Fatal("已存在的分支不应重复创建")
	}
	if m.Create("bad name") != ErrInvalidName {
		t.Fatal("非法名字应拒绝")
	}
}

func TestZhRelative(t *testing.T) {
	cases := map[string]string{"2 hours ago": "2 小时前", "1 day ago": "1 天前", "3 weeks ago": "3 周前", "5 seconds ago": "刚刚", "weird": "weird"}
	for in, want := range cases {
		if got := zhRelative(in); got != want {
			t.Errorf("%q → %q,想要 %q", in, got, want)
		}
	}
}

func TestParseBranches(t *testing.T) {
	out := "refs/heads/main\t*\t2 hours ago\nrefs/heads/dev\t \t1 day ago\nrefs/remotes/origin/HEAD\t \t1 day ago\nrefs/remotes/origin/dev\t \t1 day ago\nrefs/remotes/origin/ci\t \t3 days ago"
	bs := parseBranches(out)
	if len(bs) != 3 || bs[0].Name != "main" || !bs[0].Current || bs[2].Name != "origin/ci" || !bs[2].Remote {
		t.Fatalf("解析结果不对:%+v", bs)
	}
}

// 真实 git:建临时仓库,验证列出 / 创建 / 切换。
func TestCLI(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("没有 git")
	}
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if _, err := Open(dir); err == nil {
		t.Fatal("空目录不应当作仓库")
	}
	run("init", "-q", "-b", "main")
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644)
	run("add", ".")
	run("commit", "-q", "-m", "init")
	r, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if r.Current() != "main" {
		t.Fatalf("当前分支 %q", r.Current())
	}
	if err := r.Create("feat/x"); err != nil || r.Current() != "feat/x" {
		t.Fatalf("创建失败:%v cur=%s", err, r.Current())
	}
	if err := r.Switch("main"); err != nil || r.Current() != "main" {
		t.Fatalf("切换失败:%v", err)
	}
	bs, _ := r.Branches()
	if len(bs) != 2 {
		t.Fatalf("应有 2 个分支:%+v", bs)
	}
	if r.Switch("nope") == nil {
		t.Fatal("切不存在的分支应报错")
	}
}
