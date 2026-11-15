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
