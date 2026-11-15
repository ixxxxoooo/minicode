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
