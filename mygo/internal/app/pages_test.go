package app

import (
	"github.com/ixxxxoooo/minicode/internal/customize"
	"os"
	"strings"
	"testing"

	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/demo"
)

func TestUsagePage(t *testing.T) {
	a, tt := newTestApp(t)
	a.screen = ScreenUsage
	tt.Frame()
	// 默认 30 天;点「7 天」重算
	if u := a.usage(); len(u.Days) != 30 || core.FormatMoney(u.Total) != "$16.19" {
		t.Fatalf("30 天汇总不对:%d 天 %s", len(u.Days), core.FormatMoney(u.Total))
	}
	if err := tt.Click("7 天"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.use.rng != 0 || len(a.usage().Days) != 7 {
		t.Fatalf("应切到 7 天:%d", a.use.rng)
	}
	// 导出 CSV
	if err := tt.Click("导出 CSV"); err != nil { // 点击会走一帧,提示被界面消费,所以下面再直接调用一次读提示
		t.Fatal(err)
	}
	a.exportUsage(a.usage())
	if !strings.HasPrefix(a.toast, "已导出到 ") {
		t.Fatalf("导出提示不对:%q", a.toast)
	}
	b, err := os.ReadFile(strings.TrimPrefix(a.toast, "已导出到 "))
	if err != nil || !strings.HasPrefix(string(b), "日期,费用(美元)\n") || strings.Count(string(b), "\n") != 8 {
		t.Fatalf("CSV 内容不对:%v %q", err, b)
	}
}

func TestFormatMoney(t *testing.T) {
	for in, want := range map[float64]string{0: "$0.00", 1.5: "$1.50", 12.404: "$12.40", 0.07: "$0.07"} {
		if got := core.FormatMoney(in); got != want {
			t.Errorf("%v → %s,想要 %s", in, got, want)
		}
	}
	if u := demo.Usage(2); len(u.Days) != 15 || u.MaxDay() <= 0 {
		t.Fatalf("本月应 15 天:%d", len(u.Days))
	}
}

func TestAutomationPage(t *testing.T) {
	a, tt := newTestApp(t)
	a.screen = ScreenAutomation
	tt.Frame()
	if a.au.sel != 2 || a.au.name != "依赖升级检查" || len(a.auto.List()) != 4 {
		t.Fatalf("初始选中不对:sel=%d name=%q", a.au.sel, a.au.name)
	}
	// 点卡片切换选中,抽屉草稿随之换
	if err := tt.Click("周报生成"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.au.sel != 3 || a.au.name != "周报生成" || a.au.days != "周五" {
		t.Fatalf("应选中周报生成:sel=%d name=%q", a.au.sel, a.au.name)
	}
	// 筛选「已暂停」只剩备份检查
	if err := tt.Click("已暂停"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.au.filter != 2 {
		t.Fatal("应切到已暂停")
	}
	// 保存:坏时间被拒绝,好时间写回
	a.au.filter = 0
	a.au.tm = "9点"
	it, _ := a.auto.Get(3)
	a.autoSave(it)
	if a.au.errMsg == "" {
		t.Fatal("坏时间应报错")
	}
	a.au.tm, a.au.name = "18:00", "周报(新)"
	a.autoSave(it)
	if got, _ := a.auto.Get(3); got.Time != "18:00" || got.Name != "周报(新)" || a.au.errMsg != "" {
		t.Fatalf("保存没生效:%+v %q", got, a.au.errMsg)
	}
	// 新建 → 选中新项并聚焦名称
	if err := tt.Click("新建自动化"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if len(a.auto.List()) != 5 || a.au.sel != 5 || a.au.name != "新的自动化" {
		t.Fatalf("新建不对:n=%d sel=%d", len(a.auto.List()), a.au.sel)
	}
	// 翻周
	a.au.weekOf = 1
	tt.Frame()
	a.au.weekOf = -1
	tt.Frame()
	// 立即运行 + 删除
	a.auto.RunNow(5, a.now())
	if got, _ := a.auto.Get(5); len(got.Runs) != 1 {
		t.Fatal("立即运行应记一条记录")
	}
	if err := tt.Click("删除"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if len(a.auto.List()) != 4 {
		t.Fatalf("应已删除:%d", len(a.auto.List()))
	}
}

func TestCanvasPage(t *testing.T) {
	a, tt := newTestApp(t)
	a.screen = ScreenCanvas
	tt.Frame()
	tt.Frame() // 第二帧才量得到各块的位置
	s := a.cvStore()
	if len(s.Versions) != 3 || len(s.Doc.Blocks) != 7 {
		t.Fatalf("演示画布不对:%d 个版本 %d 个块", len(s.Versions), len(s.Doc.Blocks))
	}
	if _, ok := a.cv.pos["h-ms"]; !ok {
		t.Fatalf("应记下里程碑标题的位置:%v", a.cv.pos)
	}
	// 点大纲「待办」→ 滚动位置被设置(>0)
	if err := tt.Click("待办"); err != nil {
		t.Fatal(err)
	}
	if a.cv.outline != "h-todo" {
		t.Fatalf("大纲当前项应是待办:%q", a.cv.outline)
	}
	// 编辑 → 切预览时落成新版本
	s.Doc.Block("h-bg").Text = "改写后的导语"
	if err := tt.Click("预览"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !a.cv.preview || len(s.Versions) != 4 || s.Versions[0].By != "你" {
		t.Fatalf("切预览应提交一个版本:%d", len(s.Versions))
	}
	// 回退到 v1:没有里程碑与待办
	if err := s.Rewind(1, a.now()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if len(s.Doc.Blocks) != 3 || len(s.Versions) != 5 {
		t.Fatalf("回退后应只剩架构部分:%d 个块", len(s.Doc.Blocks))
	}
	// 导出 Markdown
	a.exportCanvas()
	if !strings.HasPrefix(a.toast, "已导出为 Markdown:") {
		t.Fatalf("导出提示不对:%q", a.toast)
	}
	// 让 Claude 修改 → 回到对话并填好输入框
	a.AskClaude("请修改画布:加一节风险")
	if a.screen != ScreenWork || a.draft != "请修改画布:加一节风险" {
		t.Fatalf("应回到对话并填入提示词:%v %q", a.screen, a.draft)
	}
}

func TestBrowserPage(t *testing.T) {
	a, tt := newTestApp(t)
	a.SeedDemoExtras()
	a.screen = ScreenBrowser
	tt.SetSize(1024, 330) // 矮窗口,正文才有可滚动的余量
	tt.Frame()
	tt.Frame()
	b := a.brw()
	if b.Page().Title() != "Routing" || a.brs.url != "docs.rs/axum/latest/axum/routing" {
		t.Fatalf("起始页不对:%q %q", b.Page().Title(), a.brs.url)
	}
	if _, ok := a.brs.pos[2]; !ok {
		t.Fatalf("应记下各节位置:%v", a.brs.pos)
	}
	// 大纲 → 滚动
	if err := tt.Click("大纲"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	// 大纲文字与正文标题同名,Click 会点到正文;改按抽屉里「大纲」标签下方的行位置点
	tab, ok := tt.Find("大纲")
	if !ok {
		t.Fatal("找不到大纲标签")
	}
	for dy := float32(30); dy < 200 && a.brs.st.Y <= 0; dy += 8 {
		tt.ClickAt(tab.X+10, tab.Y+dy)
		tt.Frame()
	}
	if a.brs.st.Y <= 0 {
		t.Fatalf("点大纲应滚动正文:%v", a.brs.st.Y)
	}
	// 链接 → 站内跳转,后退回来
	if err := tt.Click("链接"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if err := tt.Click("tower::Service"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if b.Page().Title() != "Trait Service" || !b.CanBack() || a.brs.st.Y != 0 {
		t.Fatalf("应跳到 tower 文章:%q", b.Page().Title())
	}
	b.Back()
	tt.Frame()
	if b.Page().Title() != "Routing" {
		t.Fatal("后退应回到 Routing")
	}
	// 抓不了的地址 → 错误卡,保留旧页
	a.browseTo("example.com")
	tt.Frame()
	if b.Err == nil || b.Page().Title() != "Routing" {
		t.Fatal("未接入的地址应报错并保留旧页")
	}
	// 快捷提示 → 回对话并带上页面
	a.screen = ScreenBrowser
	tt.SetSize(1024, 640)
	tt.Frame()
	tt.Frame()
	if err := tt.Click("总结要点"); err != nil {
		t.Fatal(err)
	}
	if a.screen != ScreenWork || !strings.Contains(a.draft, "docs.rs/axum/latest/axum/routing") || !strings.Contains(a.draft, "总结要点") {
		t.Fatalf("应回到对话并填好提示词:%q", a.draft)
	}
}

func TestCustomizePage(t *testing.T) {
	a, tt := newTestApp(t)
	a.SeedDemoExtras()
	a.screen = ScreenCustomize
	tt.Frame()
	tt.Frame()
	st := a.cuStore()
	if a.cuCat() != customize.MCP || a.cu.sel == nil || a.cu.sel.Name != "github" {
		t.Fatalf("默认应是 MCP 并选中第一项:%v", a.cu.sel)
	}
	// 授权:figma 的「授权」按钮
	if err := tt.Click("授权"); err != nil {
		t.Fatal(err)
	}
	if fg := st.List(customize.MCP, customize.All, "figma")[0]; fg.Status != customize.StOK {
		t.Fatalf("授权后应连通:%v", fg.Status)
	}
	// 选中 postgres(已停用):重试仍失败
	if err := tt.Click("postgres"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.cu.sel.Name != "postgres" {
		t.Fatalf("应选中 postgres:%v", a.cu.sel.Name)
	}
	if err := tt.Click("重试"); err != nil {
		t.Fatal(err)
	}
	if a.cu.sel.Status != customize.StErr {
		t.Fatal("已停用的服务器重试不应连通")
	}
	// 关键字筛选
	a.cu.q = "git"
	tt.Frame()
	if len(st.List(customize.MCP, a.cu.scope, a.cu.q)) != 1 || a.cu.sel.Name != "github" {
		t.Fatalf("筛选后选中项应退回 github:%v", a.cu.sel.Name)
	}
	a.cu.q = ""
	// 新建:名称冲突报错,合法则创建并选中
	if err := tt.Click("添加服务器"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.cu.dlg == nil {
		t.Fatal("应打开新建对话框")
	}
	a.cu.dlg.name, a.cu.dlg.extra = "github", "npx x"
	tt.Frame()
	if err := tt.Click("创建"); err != nil {
		t.Fatal(err)
	}
	if a.cu.dlg == nil || a.cu.dlg.err == "" {
		t.Fatal("重名应在对话框里报错")
	}
	a.cu.dlg.name = "linear"
	tt.Frame()
	if err := tt.Click("创建"); err != nil {
		t.Fatal(err)
	}
	if a.cu.dlg != nil || a.cu.sel == nil || a.cu.sel.Name != "linear" || st.Count(customize.MCP) != 5 {
		t.Fatalf("应创建并选中 linear:%v", a.cu.sel)
	}
	// 两步删除
	tt.Frame()
	tt.Frame()
	if err := tt.Click("删除"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if st.Count(customize.MCP) != 5 || a.cu.delArm == nil {
		t.Fatal("第一次点删除只应进入确认")
	}
	if err := tt.Click("确认删除"); err != nil {
		t.Fatal(err)
	}
	if st.Count(customize.MCP) != 4 {
		t.Fatal("确认后应删除")
	}
	// 命令:草稿改成重名,保存应失败且不改原条目
	a.SetCustomize("commands")
	tt.Frame()
	tt.Frame()
	a.cu.draft.Name = "/test"
	if err := tt.Click("保存"); err != nil {
		t.Fatal(err)
	}
	if a.cu.err == "" || a.cu.sel.Name != "/review" {
		t.Fatalf("重名保存应失败:%q %q", a.cu.err, a.cu.sel.Name)
	}
	a.cu.draft.Name = "/review2"
	tt.Frame()
	if err := tt.Click("保存"); err != nil {
		t.Fatal(err)
	}
	if a.cu.sel.Name != "/review2" || a.cu.err != "" {
		t.Fatalf("应保存成功:%q", a.cu.sel.Name)
	}
	// 在输入框试用 → 回对话并填好命令
	tt.Frame()
	if err := tt.Click("在输入框试用"); err != nil {
		t.Fatal(err)
	}
	if a.screen != ScreenWork || !strings.HasPrefix(a.draft, "/review2") {
		t.Fatalf("应回对话并填入命令:%v %q", a.screen, a.draft)
	}
	// 插件市场:安装
	a.screen = ScreenCustomize
	a.SetCustomize("plugins")
	a.cu.market = true
	tt.Frame()
	tt.Frame()
	if err := tt.Click("frontend-design"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if err := tt.Click("安装"); err != nil {
		t.Fatal(err)
	}
	if st.Count(customize.Plugins) != 5 || !a.cu.mk.Installed {
		t.Fatal("应安装 frontend-design")
	}
}
