package app

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/vcs"
)

func scmApp(t *testing.T) (*App, *ui.Tester) {
	t.Helper()
	a, tt := quickApp(t)
	a.OpenSCM()
	tt.Frame()
	tt.Frame()
	return a, tt
}

func scmHasSplit(cs []vcs.Change, staged bool, path string) bool {
	for _, c := range cs {
		if c.Path == path && c.Staged == staged {
			return true
		}
	}
	return false
}

func TestSCMLoadsAndGroups(t *testing.T) {
	a, _ := scmApp(t)
	st, wk := a.sc.split()
	if !a.sc.loaded || len(st) < 60 || len(wk) < 150 {
		t.Fatalf("分组不对:loaded=%v staged=%d work=%d", a.sc.loaded, len(st), len(wk))
	}
	if !scmHasSplit(st, true, "src/router.rs") || !scmHasSplit(wk, false, "tests/router_test.rs") || !scmHasSplit(wk, false, "tests/case_test.rs") {
		t.Fatalf("router-demo 核心改动应在正确分组:staged=%+v work=%+v", st, wk)
	}
	if len(a.sc.log) < 6 || len(a.sc.rows) != len(a.sc.log) || a.sc.ahead != 3 {
		t.Fatalf("图谱不对:log=%d rows=%d ahead=%d", len(a.sc.log), len(a.sc.rows), a.sc.ahead)
	}
	if a.sc.log[0].Hash != "a3f9c21" || a.sc.log[5].Hash != "f5521ab" {
		t.Fatalf("快照提交仍在顶端:%s … %s", a.sc.log[0].Hash, a.sc.log[5].Hash)
	}
	// 文件树的状态色随改动同步
	if a.ide.git["tests/case_test.rs"] != "A" || a.ide.git["src/router.rs"] != "M" {
		t.Fatalf("文件树状态色没同步:%v", a.ide.git)
	}
}

func TestSCMStageUnstageCommit(t *testing.T) {
	a, tt := scmApp(t)
	stN, wkN := a.sc.split()
	a.scmStage("tests/router_test.rs")
	tt.Frame()
	st, wk := a.sc.split()
	if len(st) != len(stN)+1 || len(wk) != len(wkN)-1 || !scmHasSplit(st, true, "tests/router_test.rs") {
		t.Fatalf("暂存后应 +1 staged −1 work:%d/%d (原 %d/%d)", len(st), len(wk), len(stN), len(wkN))
	}
	a.scmUnstage("tests/router_test.rs")
	tt.Frame()
	if st, wk := a.sc.split(); len(st) != len(stN) || len(wk) != len(wkN) {
		t.Fatalf("取消暂存后应恢复:%d/%d 原 %d/%d", len(st), len(wk), len(stN), len(wkN))
	}
	// 没写说明不能提交
	a.scmCommit(false)
	if !strings.Contains(a.toast, "说明") {
		t.Fatalf("应提示填写说明:%q", a.toast)
	}
	a.sc.msg = "fix: 测试提交"
	a.scmCommit(false)
	tt.Frame()
	tt.Frame()
	st, _ = a.sc.split()
	if len(st) != 0 || a.sc.msg != "" || a.sc.log[0].Subject != "fix: 测试提交" || a.sc.ahead != 4 {
		t.Fatalf("提交后状态不对:staged=%d msg=%q head=%q ahead=%d", len(st), a.sc.msg, a.sc.log[0].Subject, a.sc.ahead)
	}
	// 提交并推送 → ahead 归零
	a.scmStage("tests/router_test.rs")
	tt.Frame()
	a.sc.msg = "test: 再来一个"
	a.scmCommit(true)
	tt.Frame()
	tt.Frame()
	if a.sc.ahead != 0 {
		t.Fatalf("推送后 ahead 应为 0:%d", a.sc.ahead)
	}
}

func TestSCMDiscardAndCommitFiles(t *testing.T) {
	a, tt := scmApp(t)
	_, wkN := a.sc.split()
	a.scmDiscard("tests/case_test.rs")
	tt.Frame()
	if _, wk := a.sc.split(); len(wk) != len(wkN)-1 || scmHasSplit(wk, false, "tests/case_test.rs") {
		t.Fatalf("放弃后应少 1 个 work 文件:%d→%d", len(wkN), len(wk))
	}
	a.SetSCM("", "a3f9c21")
	tt.Frame()
	if fs := a.sc.files["a3f9c21"]; len(fs) != 2 {
		t.Fatalf("提交文件不对:%+v", fs)
	}
	var c vcs.Commit
	for _, x := range a.sc.log {
		if x.Hash == "a3f9c21" {
			c = x
		}
	}
	a.scmOpenCommitFile(c, "src/router.rs", true)
	if tb := a.ide.tab("diff:rev-a3f9c21:src/router.rs"); tb == nil || tb.title() != "router.rs (a3f9c21 的改动)" {
		t.Fatalf("提交对比标签不对:%+v", tb)
	}
}

func TestDraftCommitMsg(t *testing.T) {
	cases := []struct {
		cs   []vcs.Change
		want string
	}{
		{nil, ""},
		{[]vcs.Change{{Path: "src/router.rs", Status: 'M'}}, "fix(router): 更新 router.rs"},
		{[]vcs.Change{{Path: "tests/a_test.rs", Status: 'M'}}, "test: 更新 a_test.rs"},
		{[]vcs.Change{{Path: "README.md", Status: 'M'}}, "docs: 更新 README.md"},
		{[]vcs.Change{{Path: "src/new.rs", Status: 'U'}}, "feat(new): 新增 new.rs"},
		{[]vcs.Change{{Path: "a.go"}, {Path: "b.go"}, {Path: "c.go"}, {Path: "d.go"}}, "fix(a): 更新 a.go、b.go、c.go、等 4 个文件"},
	}
	for _, c := range cases {
		if got := draftCommitMsg(c.cs); got != c.want {
			t.Errorf("%+v → %q,想要 %q", c.cs, got, c.want)
		}
	}
}
