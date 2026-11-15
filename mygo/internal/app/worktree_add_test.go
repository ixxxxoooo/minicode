package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestWorktreeNaming(t *testing.T) {
	if got := worktreeDirName("/a/b/proj", "feat/login x"); got != "proj-feat-login-x" {
		t.Errorf("目录名:%q", got)
	}
	t.Setenv("HOME", "/home/u")
	if got := worktreeBase(""); got != "/home/u/.minicode/worktrees" {
		t.Errorf("默认位置:%q", got)
	}
	if got := worktreeBase("~/wt"); got != "/home/u/wt" {
		t.Errorf("~ 展开:%q", got)
	}
	if got := worktreeBase("/tmp/x"); got != "/tmp/x" {
		t.Errorf("绝对路径:%q", got)
	}
}

// 真实 git:新分支 / 已有分支 / 非法名 / 目录已存在 / 空名都有明确结果。
func TestWorktreeAdd(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("没有 git")
	}
	repo := filepath.Join(t.TempDir(), "proj")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	gitIn(t, repo, "init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(repo, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, repo, "add", ".")
	gitIn(t, repo, "commit", "-q", "-m", "init")
	base := filepath.Join(t.TempDir(), "wts")

	dir, err := worktreeAdd(repo, base, "feat/login")
	if err != nil {
		t.Fatalf("新分支应成功:%v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "a.txt")); err != nil {
		t.Errorf("worktree 里应有文件:%v", err)
	}
	if !strings.HasSuffix(dir, "proj-feat-login") {
		t.Errorf("目录名:%s", dir)
	}
	// 同名再建:目录已存在
	if _, err := worktreeAdd(repo, base, "feat/login"); err == nil || !strings.Contains(err.Error(), "目录已存在") {
		t.Errorf("重复应提示目录已存在:%v", err)
	}
	// 已有分支(未被检出)直接检出
	gitIn(t, repo, "branch", "old")
	if _, err := worktreeAdd(repo, base, "old"); err != nil {
		t.Errorf("已有分支应直接检出:%v", err)
	}
	// 非法名 / 空名
	if _, err := worktreeAdd(repo, base, "bad name?"); err == nil || !strings.Contains(err.Error(), "不合法") {
		t.Errorf("非法分支名:%v", err)
	}
	if _, err := worktreeAdd(repo, base, "  "); err == nil || !strings.Contains(err.Error(), "分支名") {
		t.Errorf("空分支名:%v", err)
	}
}

func TestFriendlyWorktreeRmErr(t *testing.T) {
	cases := map[string]string{
		"fatal: '/x' contains modified or untracked files, use --force to delete it": "未提交",
		"fatal: cannot remove a locked working tree":                                 "锁定",
	}
	for in, want := range cases {
		if got := friendlyWorktreeRmErr(in); !strings.Contains(got, want) {
			t.Errorf("%q → %q", in, got)
		}
	}
}
