package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/backend"
	"github.com/ixxxxoooo/minicode/internal/demo"
	"github.com/ixxxxoooo/minicode/internal/vcs"
)

func quickApp(t *testing.T) (*App, *ui.Tester) {
	t.Helper()
	a, tt := newTestApp(t)
	a.SetRepo(vcs.NewMem())
	a.SetSessionOpener(func(id string) backend.Backend { return demo.Session(id, 0.01) })
	tt.Frame()
	return a, tt
}

func TestBranchPicker(t *testing.T) {
	a, tt := quickApp(t)
	if a.BranchName() != "main" {
		t.Fatalf("分支名 %q", a.BranchName())
	}
	a.OpenQuick("branch", "")
	tt.Frame()
	rows := a.qpRows()
	if len(rows) != 4 || rows[0].title != "main" || !rows[0].current || rows[3].group != "远程" {
		t.Fatalf("列表不对:%+v", rows)
	}
	// 筛选:feat 命中两个本地分支,还出现「创建并切换」
	a.qp.query = "feat"
	rows = a.qpRows()
	if len(rows) != 3 || rows[2].icon != "plus" {
		t.Fatalf("筛选结果不对:%+v", rows)
	}
	// 回车选第一项(feat/auth)
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if a.qp != nil || a.BranchName() != "feat/auth" {
		t.Fatalf("回车应切到 feat/auth 并关闭:qp=%v branch=%s", a.qp != nil, a.BranchName())
	}
	// 输入全新名字 → 只有「创建并切换」,回车创建
	a.OpenQuick("branch", "fix/new-thing")
	tt.Frame()
	if rows := a.qpRows(); len(rows) != 1 || rows[0].icon != "plus" {
		t.Fatalf("应只有创建项:%+v", rows)
	}
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if a.BranchName() != "fix/new-thing" {
		t.Fatalf("应创建并切换:%s", a.BranchName())
	}
	// 非法名字不出现创建项
	a.OpenQuick("branch", "bad name")
	if rows := a.qpRows(); len(rows) != 0 {
		t.Fatalf("非法名字不应可创建:%+v", rows)
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if a.qp != nil {
		t.Fatal("Esc 应关闭")
	}
}

func TestHistoryPicker(t *testing.T) {
	a, tt := quickApp(t)
	a.OpenQuick("history", "")
	tt.Frame()
	rows := a.qpRows()
	if len(rows) < 8 || rows[0].group != "今天" || !rows[0].current {
		t.Fatalf("历史列表不对:%+v", rows[:2])
	}
	a.qp.query = "markdown"
	if rows := a.qpRows(); len(rows) != 1 || rows[0].group != "昨天" {
		t.Fatalf("筛选应只剩 Markdown 那条:%+v", rows)
	}
	tt.Frame()
	if err := tt.Click("Markdown 与代码高亮:发布说明"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.qp != nil || a.curSess != "md" {
		t.Fatalf("点击应切到会话 md:cur=%s", a.curSess)
	}
	if a.st.Title != "Markdown 与代码高亮:发布说明" {
		t.Fatalf("新会话标题未载入:%q", a.st.Title)
	}
}
