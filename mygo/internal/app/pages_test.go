package app

import (
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
