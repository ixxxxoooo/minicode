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

func gitRepo(t *testing.T) (*CLI, string, func(args ...string)) {
	t.Helper()
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
	run("init", "-q", "-b", "main")
	return &CLI{Dir: dir}, dir, run
}

func TestCLIStatusStageCommit(t *testing.T) {
	r, dir, run := gitRepo(t)
	write := func(name, s string) { os.WriteFile(filepath.Join(dir, name), []byte(s), 0o644) }
	write("a.txt", "1\n2\n3\n")
	run("add", ".")
	run("commit", "-q", "-m", "init")
	write("a.txt", "1\n2 changed\n3\n4\n") // 工作区改动:+2 −1
	write("new.txt", "x\ny\n")             // 未跟踪:+2
	cs, err := r.Status()
	if err != nil {
		t.Fatal(err)
	}
	find := func(p string, staged bool) *Change {
		for i := range cs {
			if cs[i].Path == p && cs[i].Staged == staged {
				return &cs[i]
			}
		}
		return nil
	}
	if c := find("a.txt", false); c == nil || c.Status != 'M' || c.Add != 2 || c.Del != 1 {
		t.Fatalf("a.txt 工作区改动不对:%+v", c)
	}
	if c := find("new.txt", false); c == nil || c.Status != 'U' || c.Add != 2 {
		t.Fatalf("new.txt 未跟踪不对:%+v", c)
	}
	// 暂存 a.txt:同一文件移到暂存区
	if err := r.Stage("a.txt"); err != nil {
		t.Fatal(err)
	}
	cs, _ = r.Status()
	if find("a.txt", true) == nil || find("a.txt", false) != nil {
		t.Fatalf("暂存后应只在暂存区:%+v", cs)
	}
	// 取消暂存
	if err := r.Unstage("a.txt"); err != nil {
		t.Fatal(err)
	}
	cs, _ = r.Status()
	if find("a.txt", false) == nil {
		t.Fatal("取消暂存后应回到工作区")
	}
	// 提交
	r.Stage("a.txt")
	if r.Commit("", false) == nil {
		t.Fatal("空说明应拒绝")
	}
	if err := r.Commit("fix: 改 a", false); err != nil {
		t.Fatal(err)
	}
	log, err := r.Log(10)
	if err != nil || len(log) != 2 || log[0].Subject != "fix: 改 a" || len(log[0].Parents) != 1 {
		t.Fatalf("日志不对:%+v %v", log, err)
	}
	if log[0].Refs[0] != "HEAD" || log[0].Refs[1] != "main" {
		t.Fatalf("引用不对:%v", log[0].Refs)
	}
	files, _ := r.CommitFiles(log[0].Hash)
	if len(files) != 1 || files[0].Path != "a.txt" || files[0].Add != 2 || files[0].Del != 1 {
		t.Fatalf("提交文件不对:%+v", files)
	}
	// HEAD 内容与放弃
	if s, ok := r.Show("HEAD", "a.txt"); !ok || s != "1\n2 changed\n3\n4\n" {
		t.Fatalf("HEAD 内容不对:%q", s)
	}
	write("a.txt", "dirty")
	if err := r.Discard("a.txt"); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "a.txt")); string(b) != "1\n2 changed\n3\n4\n" {
		t.Fatalf("放弃后应还原:%q", b)
	}
	if err := r.Discard("new.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "new.txt")); err == nil {
		t.Fatal("未跟踪文件应被删除")
	}
	if a, b := r.AheadBehind(); a != 0 || b != 0 {
		t.Fatal("没有上游应为 0")
	}
}

func TestCLIEmptyRepo(t *testing.T) {
	r, dir, _ := gitRepo(t)
	os.WriteFile(filepath.Join(dir, "f"), []byte("x"), 0o644)
	if log, err := r.Log(5); err != nil || len(log) != 0 {
		t.Fatalf("空仓库日志应为空:%v %v", log, err)
	}
	if err := r.Stage("f"); err != nil {
		t.Fatal(err)
	}
	cs, _ := r.Status()
	if len(cs) != 1 || !cs[0].Staged || cs[0].Status != 'A' {
		t.Fatalf("%+v", cs)
	}
	if err := r.Unstage("f"); err != nil { // 还没有首个提交时取消暂存
		t.Fatal(err)
	}
}

func TestParseStatusRename(t *testing.T) {
	cs := parseStatus("R  new.txt\x00old.txt\x00 M b.txt\x00MM c.txt\x00?? d.txt\x00")
	if len(cs) != 5 || cs[0].Status != 'R' || cs[0].Path != "new.txt" || cs[3].Staged || !cs[2].Staged {
		t.Fatalf("%+v", cs)
	}
}

func TestMemCommit(t *testing.T) {
	m := NewMem()
	m.SetState([]Change{{Path: "a", Status: 'M', Staged: true}, {Path: "b", Status: 'U'}}, []Commit{{Hash: "h0", Refs: []string{"HEAD", "main"}}}, nil, nil, 0)
	if m.Commit("", false) == nil {
		t.Fatal("空说明应拒绝")
	}
	if err := m.Commit("feat: x", false); err != nil {
		t.Fatal(err)
	}
	cs, _ := m.Status()
	log, _ := m.Log(5)
	if len(cs) != 1 || cs[0].Path != "b" || len(log) != 2 || log[0].Subject != "feat: x" || len(log[1].Refs) != 0 {
		t.Fatalf("提交后状态不对:%+v %+v", cs, log)
	}
	m.Stage("b")
	if c, _ := m.Status(); c[0].Status != 'A' || !c[0].Staged {
		t.Fatalf("未跟踪暂存后应是新增:%+v", c)
	}
}
