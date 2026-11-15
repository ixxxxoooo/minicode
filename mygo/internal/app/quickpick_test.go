package app

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/backend"
	"github.com/ixxxxoooo/minicode/internal/demo"
)

func quickApp(t *testing.T) (*App, *ui.Tester) {
	t.Helper()
	a, tt := newTestApp(t)
	a.SetRepo(demo.Repo())
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
	branches, err := a.repo.Branches()
	if err != nil {
		t.Fatal(err)
	}
	local, remote := 0, 0
	for _, b := range branches {
		if b.Remote {
			remote++
		} else {
			local++
		}
	}
	if len(rows) != local+remote || rows[0].title != "main" || !rows[0].current {
		t.Fatalf("列表不对:len=%d want=%d first=%+v", len(rows), local+remote, rows[0])
	}
	if rows[local].group != "远程" {
		t.Fatalf("远程分支应从第 %d 项起:%+v", local, rows[local])
	}
	// 筛选:feat 命中多个本地分支 + 远程 feat,还可能出现「创建并切换」
	a.qp.query = "feat"
	rows = a.qpRows()
	if len(rows) < 3 || rows[len(rows)-1].icon != "plus" {
		t.Fatalf("筛选结果不对:%+v", rows)
	}
	// 回车选第一项(feat/auth)
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	if a.pd.kind == "branch-dirty" && a.pd.branchFn != nil {
		a.pd.branchFn()
		a.pd = projDlg{} // 模拟用户点了「带着改动切换」,对话框随之关闭
	}
	tt.Frame()
	if a.qp != nil || a.BranchName() != "feat/auth" {
		t.Fatalf("回车应切到 feat/auth 并关闭:qp=%v branch=%s", a.qp != nil, a.BranchName())
	}
	// 输入全新名字 → 只有「创建并切换」,回车创建
	a.OpenQuick("branch", "fix/new-thing")
	tt.Frame()
	rows = a.qpRows()
	if len(rows) != 1 || rows[0].icon != "plus" {
		t.Fatalf("应只有创建项:%+v", rows)
	}
	rows[0].run()
	if a.pd.kind == "branch-dirty" && a.pd.branchFn != nil {
		a.pd.branchFn()
		a.pd = projDlg{} // 模拟用户点了「带着改动切换」,对话框随之关闭
	}
	tt.Frame()
	if a.BranchName() != "fix/new-thing" {
		t.Fatalf("应创建并切换:%s", a.BranchName())
	}
	// 非法名字不出现创建项(直接调 run 不会关选择器,先关掉再重开)
	a.closeQuick()
	tt.Frame()
	a.OpenQuick("branch", "bad name")
	if rows := a.qpRows(); len(rows) != 0 {
		t.Fatalf("非法名字不应可创建:%+v", rows)
	}
	tt.Frame()
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if a.qp != nil {
		t.Fatalf("Esc 应关闭 pd=%q popup=%q", a.pd.kind, a.popup)
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
	a.qp.query = "终稿"
	if rows := a.qpRows(); len(rows) != 1 || rows[0].group != "昨天" || rows[0].title != "【演示】发布说明终稿" {
		t.Fatalf("筛选应只剩 md 自定义标题:%+v", rows)
	}
	a.closeQuick()
	a.OpenSession("md")
	tt.Frame()
	if a.curSess != "md" {
		t.Fatalf("应切到会话 md:cur=%s", a.curSess)
	}
	if !strings.Contains(a.st.Title, "Markdown") {
		t.Fatalf("新会话标题未载入:%q", a.st.Title)
	}
}
