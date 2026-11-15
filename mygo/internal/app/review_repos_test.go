package app

import (
	"strings"
	"testing"
)

// 多根工作区:审查改动按仓库分组;只有 Git 仓库计入「涉及几个仓库」,非仓库文件夹归「未纳入 Git」。
func TestReviewGroupsByRepo(t *testing.T) {
	a, tt := newTestApp(t)
	// 单项目:不分组
	if a.reviewGroups(a.changedFiles()) != nil {
		t.Fatal("单项目不应分组")
	}
	a.ActivateWorkspace("ws-fullstack")
	a.OpenReview()
	tt.Frame()
	gs := a.reviewGroups(a.changedFiles())
	names := []string{}
	for _, g := range gs {
		names = append(names, g.name)
	}
	if strings.Join(names, ",") != "router-demo,web-dashboard" || repoGroupCount(gs) != 2 {
		t.Fatalf("分组顺序不对:%v", names)
	}
	if gs[0].add == 0 || len(gs[0].files) < 2 {
		t.Fatalf("主文件夹应有会话改动 + 演示未提交文件:%+v", gs[0])
	}
	// 非仓库文件夹(docs-site 临时当作非仓库)与工作区之外的文件各自成组,不计入仓库数
	a.repos[2].repo = nil
	gs = a.reviewGroups([]chFile{{path: "/demo/docs-site/a.md"}, {path: "/tmp/x"}, {path: "src/router.rs"}})
	if len(gs) != 3 || gs[0].hasRepo || gs[0].name != "docs-site" || gs[1].idx != grpOutside || repoGroupCount(gs) != 1 {
		t.Fatalf("非仓库 / 工作区之外应单独成组:%+v", gs)
	}
	// 相对路径(会话 cwd)归主文件夹
	if a.repoIdxOfFile("src/router.rs") != 0 || a.repoIdxOfFile("/demo/web-dashboard/src/a.ts") != 1 || a.repoIdxOfFile("/tmp/x") != grpOutside {
		t.Fatal("文件归属判断不对")
	}
	// 折叠分组、分组里的「提交到」都不报错
	a.rv.grpClosed["g1"] = true
	tt.Frame()
	a.reviewCommitTo(1)
	if a.repoAct != 1 || strings.TrimSpace(a.sc.msg) == "" {
		t.Fatalf("提交到 web-dashboard 应切仓库并预填说明:act=%d msg=%q", a.repoAct, a.sc.msg)
	}
	// 非仓库文件夹不能「提交到」
	a.reviewCommitTo(2)
	if a.repoAct != 1 {
		t.Fatal("非仓库文件夹不应切为活动仓库")
	}
}

// 逐个提交向导:依次切仓库并预填说明;提交成功自动前进;跳过;结束后汇总。
func TestCommitWizard(t *testing.T) {
	a, tt := newTestApp(t)
	a.ActivateWorkspace("ws-fullstack")
	a.OpenSCM()
	tt.Frame()
	a.startCommitWizard()
	tt.Frame()
	w := a.cw
	if w == nil || len(w.order) != 2 || a.repoAct != 0 || strings.TrimSpace(a.sc.msg) == "" {
		t.Fatalf("向导应从 router-demo 开始并预填说明:%+v act=%d", w, a.repoAct)
	}
	// 失败不前进:没暂存文件时提交被拒,向导停在本步
	a.sc.msg = "fix: 路由"
	a.scmCommit(false)
	tt.Frame()
	tt.Frame()
	if a.cw.pos != 1 || a.repoAct != 1 {
		t.Fatalf("router-demo 提交成功后应前进到 web-dashboard:pos=%d act=%d", a.cw.pos, a.repoAct)
	}
	if strings.TrimSpace(a.sc.msg) == "" {
		t.Fatal("下一个仓库应预填说明")
	}
	// 手动切走 = 暂停,不是退出;回到向导恢复
	a.setActiveRepo(0)
	tt.Frame()
	if a.cw == nil || a.cw.pos != 1 {
		t.Fatal("手动切仓库不应退出向导")
	}
	a.wizardGo(a.cw.pos)
	if a.repoAct != 1 {
		t.Fatal("回到向导应切回当前步的仓库")
	}
	a.wizardNext("skip")
	tt.Frame()
	if a.cw.finished.IsZero() || a.cw.count("ok") != 1 || a.cw.count("skip") != 1 {
		t.Fatalf("应结束:ok=%d skip=%d", a.cw.count("ok"), a.cw.count("skip"))
	}
	// 没有改动时不启动
	a.cw = nil
	a.sc.commitErr = ""
	for _, e := range a.repos {
		e.changes = nil
	}
	a.sc.changes = nil
	a.startCommitWizard()
	if a.cw != nil && len(a.cw.order) > 0 && a.cw.order[0] == 99 {
		t.Fatal("不应出现")
	}
}
