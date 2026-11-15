package app

import (
	"strings"
	"testing"
)

const confSrc = "a\n<<<<<<< HEAD\nx1\nx2\n=======\ny1\n>>>>>>> feat\nb\n<<<<<<< HEAD\nc\n||||||| base\nB\n=======\nd\n>>>>>>> feat\nz"

func TestParseAndComposeConflicts(t *testing.T) {
	segs := parseConflicts(confSrc)
	n := 0
	for _, s := range segs {
		if s.conflict {
			n++
		}
	}
	if n != 2 {
		t.Fatalf("应解析出 2 处冲突,得到 %d", n)
	}
	if segs[1].line != 2 || len(segs[1].cur) != 2 || len(segs[1].inc) != 1 || segs[1].incTag != "feat" {
		t.Fatalf("第一处冲突解析不对:%+v", segs[1])
	}
	if c := segs[3]; len(c.cur) != 1 || c.cur[0] != "c" || len(c.inc) != 1 {
		t.Fatalf("diff3 基底应被丢弃:%+v", c)
	}
	got := composeConflicts(segs, map[int]int{0: chCur, 1: chInc})
	if got != "a\nx1\nx2\nb\nd\nz" {
		t.Fatalf("接受当前 / 传入的结果不对:%q", got)
	}
	if got := composeConflicts(segs, map[int]int{0: chBoth, 1: chCur}); got != "a\nx1\nx2\ny1\nb\nc\nz" {
		t.Fatalf("接受双方不对:%q", got)
	}
	// 未处理的保留成标记,再解析仍是两处
	if len(parseConflicts(composeConflicts(segs, nil))) != len(segs) {
		t.Fatal("未处理的冲突应原样保留")
	}
	// 不完整的标记当普通文本
	if segs := parseConflicts("<<<<<<< x\nfoo"); len(segs) != 1 || segs[0].conflict {
		t.Fatalf("没有闭合的标记应是普通文本:%+v", segs)
	}
}

func TestConflictTabResolve(t *testing.T) {
	a, tt := demoIDE(t)
	tt.Frame()
	a.demoConflict()
	tt.Frame()
	var tb *ideTab
	for _, x := range a.ide.tabs {
		if x.virt == vConflict {
			tb = x
		}
	}
	if tb == nil {
		t.Fatal("应打开合并冲突标签")
	}
	s := a.conflictOf(tb.ref)
	if total, done := s.count(); total != 2 || done != 0 {
		t.Fatalf("应有 2 处未处理冲突:%d/%d", done, total)
	}
	s.choice[0], s.choice[1] = chInc, chBoth
	tt.Frame()
	a.finishConflict(tb, s)
	tt.Frame()
	out, _ := a.ide.fs.Read(tb.ref)
	if strings.Contains(out, "<<<<<<<") || !strings.Contains(out, "/healthz") || !strings.Contains(out, "State::default()") || !strings.Contains(out, "State::new(auth())") {
		t.Fatalf("写回的内容不对:\n%s", out)
	}
	if a.ide.tab(tb.id) != nil {
		t.Fatal("解决后标签应关闭")
	}
}

func TestStashAndUndoCommit(t *testing.T) {
	a, tt := demoIDE(t)
	a.OpenSCM()
	tt.Frame()
	tt.Frame()
	if len(a.sc.stashes) != 5 {
		t.Fatalf("演示应有 5 条储藏:%d", len(a.sc.stashes))
	}
	nCh := len(a.sc.changes)
	a.scmStashPush()
	tt.Frame()
	if len(a.sc.stashes) != 6 || len(a.sc.changes) != 0 {
		t.Fatalf("储藏后应有 6 条储藏、没有改动:%d / %d", len(a.sc.stashes), len(a.sc.changes))
	}
	a.scmStashApply(0, true)
	tt.Frame()
	if len(a.sc.stashes) != 5 || len(a.sc.changes) != nCh {
		t.Fatalf("弹出后改动应恢复:%d / %d(期望 %d)", len(a.sc.stashes), len(a.sc.changes), nCh)
	}
	head := a.sc.log[0].Hash
	a.scmUndoCommit()
	tt.Frame()
	if a.sc.log[0].Hash == head {
		t.Fatal("撤销上次提交后 HEAD 应前移")
	}
}

func TestBlameFallbackAndTimeline(t *testing.T) {
	a, tt := demoIDE(t)
	tt.Frame()
	a.ide.openFile("src/router.rs", true)
	tt.Frame()
	g := a.gitInfoFor("src/router.rs")
	tt.Frame()
	if g == nil || len(g.log) == 0 {
		t.Fatal("src/router.rs 应有文件历史")
	}
	tb := a.ide.cur()
	if bl, ok := a.blameLine(tb, g, 0); !ok || bl.Author == "" || bl.Summary == "" {
		t.Fatalf("没有追溯数据时应兜底到最近提交:%+v", bl)
	}
}
