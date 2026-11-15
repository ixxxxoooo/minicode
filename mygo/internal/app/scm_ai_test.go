package app

import (
	"errors"
	"strings"
	"testing"
)

func TestCleanCommitMsg(t *testing.T) {
	cases := map[string]string{
		"```\nfix(router): 路由忽略大小写\n```": "fix(router): 路由忽略大小写",
		"提交说明: \"feat: 新增 normalize\"":   "feat: 新增 normalize",
		"fix: a\n\n\n\n- 要点一\n- 要点二\n":   "fix: a\n\n- 要点一\n- 要点二",
		"  docs: 更新说明  ":                 "docs: 更新说明",
	}
	for in, want := range cases {
		if got := cleanCommitMsg(in); got != want {
			t.Errorf("清洗 %q → %q,想要 %q", in, got, want)
		}
	}
}

func TestSCMAIMessage(t *testing.T) {
	a, tt := scmApp(t)
	var prompt string
	a.SetTextGenerator(func(dir, p string) (string, error) {
		prompt = p
		return "```\nfix(router): 路由匹配忽略大小写\n```", nil
	})
	a.scmGenMsg()
	tt.Frame()
	if a.sc.msg != "fix(router): 路由匹配忽略大小写" || a.sc.genning {
		t.Fatalf("应填入清洗后的 AI 说明:%q genning=%v", a.sc.msg, a.sc.genning)
	}
	if !strings.Contains(prompt, "router.rs") || !strings.Contains(prompt, "Conventional Commits") || !strings.Contains(prompt, "+") {
		t.Fatalf("提示里应有文件清单与差异节选:%s", prompt)
	}

	// 失败 → 退回本地草稿并提示
	a.sc.msg = ""
	a.SetTextGenerator(func(dir, p string) (string, error) { return "", errors.New("没有登录") })
	a.scmGenMsg()
	if a.sc.msg == "" || !strings.Contains(a.toast, "本地草稿") {
		t.Fatalf("失败应退回草稿:msg=%q toast=%q", a.sc.msg, a.toast)
	}

	// 关掉 AI 开关 → 不调用生成器
	a.sc.msg = ""
	called := false
	a.SetTextGenerator(func(dir, p string) (string, error) { called = true; return "x", nil })
	a.prefs.Set("scm.aiMsg", false)
	a.scmGenMsg()
	if called || a.sc.msg == "" {
		t.Fatalf("关闭开关应只用本地草稿 called=%v msg=%q", called, a.sc.msg)
	}
}

func TestGraphTab(t *testing.T) {
	a, tt := scmApp(t)
	a.openGraph()
	tt.Frame()
	if tb := a.ide.cur(); tb == nil || tb.virt != vGraph {
		t.Fatal("应打开 Git 图谱虚拟标签")
	}
	if !tt.HasText("Git 图谱") || !tt.HasText("补充路由匹配说明") {
		t.Fatalf("图谱标签应列出提交:%v", tt.Texts())
	}
	// 搜索:命中判断(不匹配的行变淡,不删行)
	m := graphMatch{"lin"}
	hits := 0
	for _, cm := range a.sc.log {
		if m.hit(cm) {
			hits++
		}
	}
	if hits != 2 {
		t.Fatalf("按作者 Lin 搜索应命中 2 条,实际 %d", hits)
	}
	if !(graphMatch{"origin/main"}).hit(a.sc.log[4]) || (graphMatch{"zzz"}).hit(a.sc.log[0]) {
		t.Fatal("引用名搜索 / 未命中判断不对")
	}
	// 选中一条:读出涉及文件并在详情里显示
	a.graphTab.sel = a.sc.log[1].Hash
	a.scmCommitFiles(a.sc.log[1].Hash)
	tt.Frame()
	tt.Frame()
	if !tt.HasText("tests/router_test.rs") {
		t.Fatalf("详情应列出该提交的文件:%v", tt.Texts())
	}
}
