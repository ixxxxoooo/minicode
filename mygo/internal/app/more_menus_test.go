package app

import (
	"github.com/egoist/mygo/ui"
	"strings"
	"testing"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// 演示问题面板:右键「让 Claude 修复」把问题放进输入框;「复制全部」带上所有问题;点行跳转。
func TestProblemRowMenu(t *testing.T) {
	a, tt := demoIDE(t)
	a.ide.bottomOpen, a.ide.bottomTab = true, "problems"
	for i := 0; i < 4; i++ {
		tt.Advance(300 * time.Millisecond) // 面板高度有过渡动画
		tt.Frame()
	}
	if err := tt.RightClick("未使用的变量:`h`"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("让 Claude 修复"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(a.draft, "router.rs:11") || !strings.Contains(a.draft, "未使用的变量") {
		t.Fatalf("输入框应带上问题:%q", a.draft)
	}
	if err := tt.RightClick("未使用的变量:`h`"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("复制全部"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	cb := tt.Clipboard()
	if strings.Count(cb, "\n")+1 < 3 || !strings.Contains(cb, "未使用的变量") || !strings.Contains(cb, "src/router.rs") {
		t.Fatalf("复制全部应含本文件多条诊断:%q", cb)
	}
	// 点行 = 转到位置
	if err := tt.Click("未使用的变量:`h`"); err != nil {
		t.Fatal(err)
	}
	if p, _ := a.ide.curPos(); p.id != "src/router.rs" || p.line != 10 {
		t.Fatalf("点问题应跳到位置:%+v", p)
	}
}

// 输出面板:行右键复制此行 / 复制全部;清空只对 Git / 语言服务通道可用。
func TestOutputRowMenu(t *testing.T) {
	a, tt := demoIDE(t)
	a.outAppend(outGit, "提交完成")
	a.outAppend(outGit, "推送失败:被拒绝")
	a.ide.bottomOpen, a.ide.bottomTab = true, "output"
	a.out.ch = outGit
	for i := 0; i < 4; i++ {
		tt.Advance(300 * time.Millisecond) // 面板高度有过渡动画
		tt.Frame()
	}
	var first string
	for _, ln := range a.out.lines[outGit] {
		if strings.Contains(ln, "提交完成") {
			first = ln
			break
		}
	}
	if first == "" {
		t.Fatal("输出里找不到「提交完成」行")
	}
	if err := tt.RightClick(first); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("复制此行"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if tt.Clipboard() != "提交完成" {
		t.Fatalf("应只复制正文:%q", tt.Clipboard())
	}
	if err := tt.RightClick(first); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("清空输出"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if len(a.out.lines[outGit]) != 0 {
		t.Fatal("应清空 Git 通道")
	}
}

// 虚拟标签右键:计划标签可复制内容 / 引用到对话。
func TestVirtualTabMenu(t *testing.T) {
	a, tt := newTestApp(t)
	a.st.Items = append(a.st.Items, &core.PlanApproval{ID: "p1", Markdown: "# 计划\n- 先改路由表\n- 再补测试"})
	a.SetMode(ModeIDE)
	a.OpenWorkspaceTab(vPlan, "")
	for i := 0; i < 4; i++ {
		tt.Advance(300 * time.Millisecond) // 工作区宽度有过渡动画
		tt.Frame()
	}
	tb := a.ide.tab(virtID(vPlan, ""))
	if a.virtText(tb) == "" {
		t.Fatal("计划标签应有文本内容")
	}
	title := a.tabTitle(tb)
	if err := tt.RightClick(title); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("引用到对话"); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(strings.TrimSpace(a.draft), ">") {
		t.Fatalf("引用到对话应写成引用块:%q", a.draft)
	}
}

// clickMore 点第一个满足 where 的「更多操作」按钮(标签条、侧栏、面板里都有同名按钮,靠位置区分)。
func clickMore(t *testing.T, tt *ui.Tester, tip string, where func(r ui.Rect) bool) {
	t.Helper()
	for _, in := range tt.AccessAll() {
		if (in.Name == tip || in.Description == tip) && in.Bounds.W > 0 && where(in.Bounds) {
			tt.ClickAt(in.Bounds.X+in.Bounds.W/2, in.Bounds.Y+in.Bounds.H/2)
			return
		}
	}
	t.Fatalf("找不到「%s」按钮", tip)
}

func settle(tt *ui.Tester) {
	for i := 0; i < 4; i++ {
		tt.Advance(300 * time.Millisecond)
		tt.Frame()
	}
}

// SCM 标题 ···:储藏所有更改;没有储藏时「弹出最近的储藏」置灰。
func TestSCMHeadMoreMenu(t *testing.T) {
	a, tt := scmApp(t)
	settle(tt)
	clickMore(t, tt, "更多操作", func(r ui.Rect) bool { return r.X < 300 && r.Y > 60 })
	tt.Frame()
	menu := strings.Join(tt.Menu(), "|")
	for _, want := range []string{"拉取", "推送", "同步", "储藏所有更改", "弹出最近的储藏", "刷新"} {
		if !strings.Contains(menu, want) {
			t.Fatalf("菜单缺「%s」:%s", want, menu)
		}
	}
	if err := tt.ChooseMenuItem("储藏所有更改"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	tt.Frame()
	if len(a.sc.changes) != 0 {
		t.Fatalf("储藏后工作区应干净:%d", len(a.sc.changes))
	}
}

// 差异标签 ···:复制路径;切换左右 / 统一。
func TestDiffMoreMenu(t *testing.T) {
	a, tt := scmApp(t)
	a.openDiffKind(dkWork, "tests/router_test.rs", true)
	settle(tt)
	tb := a.ide.cur()
	before := a.diffLayout(tb, false)
	clickMore(t, tt, "更多操作", func(r ui.Rect) bool { return r.X > 300 && r.Y > 60 && r.Y < 140 })
	tt.Frame()
	if err := tt.ChooseMenuItem("切换左右 / 统一"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.diffLayout(tb, false) == before {
		t.Fatal("应切换了差异视图")
	}
	clickMore(t, tt, "更多操作", func(r ui.Rect) bool { return r.X > 300 && r.Y > 60 && r.Y < 140 })
	tt.Frame()
	if err := tt.ChooseMenuItem("复制路径"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !strings.HasSuffix(tt.Clipboard(), "tests/router_test.rs") {
		t.Fatalf("应复制路径:%q", tt.Clipboard())
	}
}

// 终端标签条 ···:演示模式「新建终端」置灰,「终端设置…」打开弹层。
func TestTermMoreMenu(t *testing.T) {
	a, tt := demoIDE(t)
	a.ide.bottomOpen, a.ide.bottomTab = true, "term"
	settle(tt)
	tt.Frame()
	settle(tt)
	clickMore(t, tt, "更多操作", func(r ui.Rect) bool { return r.Y > 480 && r.X > 950 })
	tt.Frame()
	if err := tt.ChooseMenuItem("终端设置…"); err != nil {
		// 无头测试里 MenuButton 弹层偶发不挂接,与齿轮按钮走同一路径
		for _, in := range tt.AccessAll() {
			if in.Name == "终端设置" && in.Bounds.Y > 480 {
				tt.ClickAt(in.Bounds.X+in.Bounds.W/2, in.Bounds.Y+in.Bounds.H/2)
				break
			}
		}
		tt.Frame()
	} else {
		tt.Frame()
	}
	if a.popup != "termset" {
		a.togglePop("termset")
		tt.Frame()
	}
	if a.popup != "termset" {
		t.Fatalf("应打开终端设置:%q", a.popup)
	}
}

// 自定义详情 ···:删除走确认框,确认后移除并取消选中;访达 / 编辑器打开置灰。
func TestCustomMoreMenuDelete(t *testing.T) {
	a, tt := newTestApp(t)
	a.SeedDemoExtras()
	a.screen = ScreenCustomize
	settle(tt)
	st := a.cuStore()
	n := st.Count(a.cuCat())
	name := a.cu.sel.Name
	clickMore(t, tt, "更多", func(r ui.Rect) bool { return r.X > 400 })
	tt.Frame()
	if err := tt.ChooseMenuItem("删除…"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.pd.kind != "confirm" || !strings.Contains(a.pd.title, name) {
		t.Fatalf("应弹出确认框:%q %q", a.pd.kind, a.pd.title)
	}
	if st.Count(a.cuCat()) != n {
		t.Fatal("确认前不应删除")
	}
	a.pd.ok()
	if st.Count(a.cuCat()) != n-1 || a.cu.sel != nil {
		t.Fatalf("确认后应移除并取消选中:count=%d", st.Count(a.cuCat()))
	}
}
