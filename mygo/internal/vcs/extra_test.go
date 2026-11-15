package vcs

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseStatusConflict(t *testing.T) {
	cs := parseStatus("UU a.rs\x00 M b.rs\x00AA c.rs\x00")
	if len(cs) != 3 || cs[0].Status != StatusConflict || cs[1].Status != 'M' || cs[2].Status != StatusConflict {
		t.Fatalf("冲突状态解析不对:%+v", cs)
	}
}

func TestParseBlame(t *testing.T) {
	h1 := strings.Repeat("a", 40)
	h0 := strings.Repeat("0", 40)
	out := h1 + " 1 1 2\nauthor Lin\nauthor-time 1700000000\nsummary init\nfilename x\n\tline one\n" +
		h1 + " 2 2\n\tline two\n" + h0 + " 3 3 1\nauthor Not Committed Yet\nsummary Version of x\n\tline three\n"
	ls := parseBlame(out)
	if len(ls) != 3 || ls[0].Author != "Lin" || ls[1].Hash != h1 || ls[1].Author != "Lin" || !ls[2].Uncommitted || ls[2].When != "未提交" {
		t.Fatalf("追溯解析不对:%+v", ls)
	}
}

// 真实 git:块级暂存的落点 StageContent、Index、FileLog、储藏。
func TestCLIStageContentAndStash(t *testing.T) {
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
	run("init", "-q", "-b", "main")
	file := filepath.Join(dir, "a.txt")
	os.WriteFile(file, []byte("1\n2\n3\n"), 0o644)
	run("add", ".")
	run("commit", "-q", "-m", "init")
	os.WriteFile(file, []byte("1\nTWO\n3\nFOUR\n"), 0o644)

	r := &CLI{Dir: dir}
	// 只暂存第一处改动(2 → TWO),第二处(新增 FOUR)留在工作区
	if err := r.StageContent("a.txt", "1\nTWO\n3\n"); err != nil {
		t.Fatal(err)
	}
	if idx, ok := r.Index("a.txt"); !ok || idx != "1\nTWO\n3\n" {
		t.Fatalf("暂存区内容不对:%q %v", idx, ok)
	}
	cs, _ := r.Status()
	var staged, work bool
	for _, c := range cs {
		staged = staged || c.Staged
		work = work || !c.Staged
	}
	if !staged || !work {
		t.Fatalf("同一文件应同时有暂存与未暂存的改动:%+v", cs)
	}
	if log, _ := r.FileLog("a.txt", 5); len(log) != 1 || log[0].Subject != "init" {
		t.Fatalf("文件历史不对:%+v", log)
	}
	if bl, err := r.Blame("a.txt"); err != nil || len(bl) != 4 || bl[0].Author != "t" || !bl[1].Uncommitted {
		t.Fatalf("追溯不对:%+v %v", bl, err)
	}
	if err := r.StashPush("实验"); err != nil {
		t.Fatal(err)
	}
	ss, _ := r.Stashes()
	if len(ss) != 1 || !strings.Contains(ss[0].Msg, "实验") {
		t.Fatalf("储藏列表不对:%+v", ss)
	}
	if err := r.StashApply(0, true); err != nil {
		t.Fatal(err)
	}
	if ss, _ := r.Stashes(); len(ss) != 0 {
		t.Fatalf("弹出后储藏应为空:%+v", ss)
	}
}

// 回退到某个提交:之后的提交被移除,未提交改动保留;哈希不合法被拒。
func TestCLIResetKeep(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("没有 git")
	}
	dir := t.TempDir()
	run := func(args ...string) string {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	run("init", "-q", "-b", "main")
	file := filepath.Join(dir, "a.txt")
	other := filepath.Join(dir, "b.txt")
	os.WriteFile(file, []byte("1\n"), 0o644)
	os.WriteFile(other, []byte("x\n"), 0o644)
	run("add", ".")
	run("commit", "-q", "-m", "c1")
	first := run("rev-parse", "HEAD")
	os.WriteFile(file, []byte("1\n2\n"), 0o644)
	run("commit", "-qam", "c2")
	os.WriteFile(other, []byte("local edit\n"), 0o644) // 未提交的改动应保留

	r := &CLI{Dir: dir}
	if err := r.ResetKeep("zz; rm -rf"); err != ErrInvalidHash {
		t.Fatalf("非法哈希应被拒:%v", err)
	}
	if err := r.ResetKeep(first); err != nil {
		t.Fatal(err)
	}
	if got := run("rev-parse", "HEAD"); got != first {
		t.Fatalf("HEAD 应回到第一个提交:%s", got)
	}
	if b, _ := os.ReadFile(file); string(b) != "1\n" {
		t.Fatalf("已提交的内容应回到当时的样子:%q", b)
	}
	if b, _ := os.ReadFile(other); string(b) != "local edit\n" {
		t.Fatalf("未提交改动应保留:%q", b)
	}
	// 与回退内容冲突:改动 a.txt 后再回退到更早的,应中止
	os.WriteFile(file, []byte("1\n2\n"), 0o644)
	run("commit", "-qam", "c3")
	os.WriteFile(file, []byte("dirty\n"), 0o644)
	if err := r.ResetKeep(first); err == nil {
		t.Fatal("本地改动与回退内容冲突时应中止")
	}
}

func TestMemResetKeep(t *testing.T) {
	m := NewMem()
	for _, msg := range []string{"c1", "c2", "c3"} {
		cur, _ := m.Log(10)
		m.SetState([]Change{{Path: msg + ".txt", Status: 'A', Staged: true}}, cur, nil, nil, 0)
		if err := m.Commit(msg, false); err != nil {
			t.Fatal(err)
		}
	}
	log, _ := m.Log(10)
	if len(log) != 3 {
		t.Fatalf("应有 3 个提交:%d", len(log))
	}
	target := log[2].Hash
	if err := m.ResetKeep(target); err != nil {
		t.Fatal(err)
	}
	after, _ := m.Log(10)
	if len(after) != 1 || after[0].Hash != target {
		t.Fatalf("回退后顶端应是目标提交:%+v", after)
	}
	hasHead := false
	for _, r := range after[0].Refs {
		hasHead = hasHead || r == "HEAD"
	}
	if !hasHead {
		t.Fatalf("HEAD 应搬到目标提交:%v", after[0].Refs)
	}
	if err := m.ResetKeep("nope"); err == nil {
		t.Fatal("找不到的提交应报错")
	}
}
