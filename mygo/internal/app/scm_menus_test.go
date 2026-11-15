package app

import (
	"strings"
	"testing"
)

// SCM 文件行右键「放弃更改…」先确认,确认前什么都不丢。
func TestSCMFileMenuDiscardConfirm(t *testing.T) {
	a, tt := scmApp(t)
	_, before := a.sc.split()
	if err := tt.RightClick("case_test.rs"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("放弃更改…"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.pd.kind != "confirm" || !strings.Contains(a.pd.title, "case_test.rs") {
		t.Fatalf("应弹出确认框:%q %q", a.pd.kind, a.pd.title)
	}
	if _, wk := a.sc.split(); len(wk) != len(before) {
		t.Fatal("确认前不应放弃")
	}
	a.pd.ok()
	tt.Frame()
	if _, wk := a.sc.split(); len(wk) != len(before)-1 {
		t.Fatalf("确认后应放弃 1 个:%d", len(wk))
	}
}

func TestSCMDiscardAllAsk(t *testing.T) {
	a, tt := scmApp(t)
	_, wk := a.sc.split()
	a.scmDiscardAsk(wk)
	if !strings.Contains(a.pd.title, itoa(len(wk))+" 个文件") {
		t.Fatalf("应写明文件数:%q", a.pd.title)
	}
	a.pd.ok()
	tt.Frame()
	if _, wk := a.sc.split(); len(wk) != 0 {
		t.Fatalf("应全部放弃:%d", len(wk))
	}
}

// 回退到这里:当前提交 / 其它分支的提交置灰;确认后移除其后的提交。
func TestSCMResetToCommit(t *testing.T) {
	a, tt := scmApp(t)
	log := a.sc.log
	if _, why := a.resetInfo(log[0]); why == "" {
		t.Fatal("当前提交不能回退到自己")
	}
	n, why := a.resetInfo(log[2])
	if why != "" || n < 2 {
		t.Fatalf("第 3 个提交应可回退且至少移除 2 个:n=%d why=%q", n, why)
	}
	a.askReset(log[2], n)
	if !strings.Contains(a.pd.title, log[2].Short()) || !strings.Contains(a.pd.body, "reflog") {
		t.Fatalf("确认文案不对:%q / %q", a.pd.title, a.pd.body)
	}
	a.pd.ok()
	tt.Frame()
	tt.Frame()
	if a.sc.log[0].Hash != log[2].Hash {
		t.Fatalf("回退后顶端应是目标提交:%s", a.sc.log[0].Short())
	}
}

// 与工作区比较:先选文件(快速选择),选中后开「提交 ↔ 工作区」差异标签。
func TestSCMCompareWithWork(t *testing.T) {
	a, tt := scmApp(t)
	var cm = a.sc.log[0]
	for _, x := range a.sc.log {
		if x.Hash == "a3f9c21" {
			cm = x
		}
	}
	a.scmCompareWork(cm)
	tt.Frame()
	if a.qp == nil || a.qp.kind != "cmpfiles" || len(a.qpRows()) != 2 {
		t.Fatalf("应列出该提交的 2 个文件:%+v", a.qp)
	}
	a.qpRows()[0].run()
	tt.Frame()
	tb := a.ide.cur()
	if tb == nil || !tb.diff || !strings.Contains(tb.id, "cmp-a3f9c21") || tb.label != "a3f9c21 ↔ 工作区" {
		t.Fatalf("应打开比较标签:%+v", tb)
	}
}

func TestSearchFileMenuFold(t *testing.T) {
	a, tt := newTestApp(t)
	a.OpenSearch()
	a.SetSearch("match_path", "", false)
	tt.Frame()
	f := a.sr.res.Files[0].Path
	name := f[strings.LastIndex(f, "/")+1:]
	if err := tt.RightClick(name); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("折叠"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !a.sr.folded[f] {
		t.Fatal("右键「折叠」应折叠该文件")
	}
}
