package app

import (
	"testing"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

func TestUsageModelCostUnknown(t *testing.T) {
	if usageModelCost("deepseek-chat", 0) != "未知费用" {
		t.Fatal("DeepSeek 无费用应未知")
	}
	if usageModelCost("Sonnet 5.5", 0) != "$0.00" {
		t.Fatal("有价目表可为 0")
	}
	if usageModelCost("deepseek-chat", 1.2) != "$1.20" {
		t.Fatal("有费用仍显示金额")
	}
}

func TestUsageLimitsFromSessionMarksUnavailable(t *testing.T) {
	a, _ := newTestApp(t)
	a.st = core.NewState()
	a.st.Usage.LimitsState = core.LimitsUnavailable
	a.use.limitsState = core.LimitsChecking
	a.usageLimitsFromSession()
	if a.use.limitsAt == 0 {
		t.Fatal("查询结束应记下时间")
	}
}

func TestUsageUnavailableProviderMsg(t *testing.T) {
	a, _ := newTestApp(t)
	a.demoExtra = false
	a.use.limitsState = core.LimitsUnavailable
	a.use.limitsAt = time.Now().Unix()
	u := core.UsageReport{Days: []core.Day{{Cost: 1}}, Models: []core.Share{{Name: "deepseek-chat", Cost: 0, Fraction: 1}}}
	a.SetUsageSource(func(int) core.UsageReport { return u })
	got := a.usage()
	if len(got.Limits) != 0 {
		t.Fatal("不可用时不应有假限额")
	}
	if usageModelCost(got.Models[0].Name, got.Models[0].Cost) != "未知费用" {
		t.Fatal("汇总应标未知费用")
	}
}
