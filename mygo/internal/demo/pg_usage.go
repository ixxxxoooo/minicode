// 演示数据索引 · 用量
//
// 套 0:原 Usage(rng) 行为。
// 套 1:空 Days(新账户 loading 态)。
// 套 2–8:爆表周、单日尖峰、多模型含第三方、180 天、限额 94%/100%、缓存 12%、20 条 Top 会话。
//
// 切换:命令面板「演示:用量」。范围分段仍作用于套 0;其它套忽略 rng 或用固定天数。
package demo

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// UsageFor 按演示套与范围下标返回用量报告。
func UsageFor(set, rng int) core.UsageReport {
	switch set {
	case 1:
		return usageEmpty()
	case 2:
		return usageHeavyWeek()
	case 3:
		return usageSpikeDay()
	case 4:
		return usageMultiModel()
	case 5:
		return usageDays180()
	case 6:
		return usageLimitsHigh()
	case 7:
		return usageCacheLow()
	case 8:
		return usageTopLong()
	default:
		return Usage(rng)
	}
}

func usageEmpty() core.UsageReport {
	return core.UsageReport{RangeName: "新账户", Days: nil, Limits: nil, Tips: nil}
}

func fillModelsStandard(u *core.UsageReport) {
	u.Models = []core.Share{
		{Name: "Sonnet 5.5", Fraction: 0.72, Cost: u.Total * 0.72, Kind: 0},
		{Name: "Opus 5.5", Fraction: 0.21, Cost: u.Total * 0.21, Kind: 1},
		{Name: "Haiku 5.5", Fraction: 0.07, Cost: u.Total * 0.07, Kind: 2},
	}
}

func fillProjectsMany(u *core.UsageReport) {
	names := []string{"router-demo", "web-dashboard", "docs-site", "scenario-lab", "mini-core", "infra-tools", "design-docs", "playground"}
	for i, n := range names {
		u.Projects = append(u.Projects, core.Share{Name: n, Cost: u.Total * (0.18 - float64(i)*0.015), Sessions: 40 - i*3})
	}
}

func usageHeavyWeek() core.UsageReport {
	u := synthDays(7, "7 天", func(i int) float64 {
		return 1.2 + float64(i)*0.35
	})
	u.Total, u.DeltaPercent = sumDays(u.Days), 86
	u.InputTokens, u.OutputTokens, u.Sessions = "2.8M", "420k", 128
	u.CacheHit = 78
	fillModelsStandard(&u)
	fillProjectsMany(&u)
	u.Limits = []core.Limit{{Name: "5 小时窗口", Percent: 62, Reset: "1 小时 20 分后重置"}, {Name: "每周", Percent: 88, Reset: "周三重置"}}
	u.LimitsUpdated = "刚刚"
	u.Tips = []core.Tip{{Level: 1, Title: "本周费用偏高", Desc: "检查 scenario-lab 的长会话"}}
	return u
}

func usageSpikeDay() core.UsageReport {
	u := synthDays(30, "30 天", func(i int) float64 {
		if i == 29 {
			return 4.8
		}
		return 0.25 + math.Sin(float64(i))*0.1
	})
	u.Total = sumDays(u.Days)
	u.DeltaPercent = 240
	u.Sessions = 64
	u.CacheHit = 55
	fillModelsStandard(&u)
	u.TopSessions = []core.CostSession{{Title: "单日爆表 · 批量重构", Project: "router-demo", Model: "Opus 5.5", Cost: 3.2}}
	u.Limits = []core.Limit{{Name: "5 小时窗口", Percent: 100, Reset: "已用尽 · 22 分钟后重置"}}
	u.LimitsUpdated = "1 分钟前"
	return u
}

func usageMultiModel() core.UsageReport {
	u := synthDays(30, "30 天", func(i int) float64 { return dayCost(i) })
	u.Total = sumDays(u.Days)
	u.Models = []core.Share{
		{Name: "Sonnet 5.5", Fraction: 0.42, Kind: 0},
		{Name: "Opus 5.5", Fraction: 0.28, Kind: 1},
		{Name: "Haiku 5.5", Fraction: 0.12, Kind: 2},
		{Name: "MiniMax-M2.5", Fraction: 0.12, Kind: 0},
		{Name: "Kimi-K2", Fraction: 0.06, Kind: 0},
	}
	for i := range u.Models {
		u.Models[i].Cost = u.Total * u.Models[i].Fraction
	}
	u.CacheHit = 72
	u.InputTokens, u.OutputTokens, u.Sessions = "6.1M", "890k", 210
	fillProjectsMany(&u)
	u.Tips = []core.Tip{{Level: 0, Title: "第三方模型占 18%", Desc: "费用估算可能不完整"}}
	return u
}

func usageDays180() core.UsageReport {
	u := synthDays(180, "180 天", func(i int) float64 { return dayCost(i) })
	u.Total = sumDays(u.Days)
	u.DeltaPercent = 12
	u.Sessions = 890
	u.CacheHit = 81
	fillModelsStandard(&u)
	fillProjectsMany(&u)
	u.LimitsUpdated = "10 分钟前"
	return u
}

func usageLimitsHigh() core.UsageReport {
	u := synthDays(30, "30 天", func(i int) float64 { return dayCost(i) })
	u.Total = sumDays(u.Days)
	u.Limits = []core.Limit{
		{Name: "5 小时窗口", Percent: 94, Reset: "14 分钟后将达上限"},
		{Name: "每周", Percent: 100, Reset: "已用尽 · 周一 09:00 重置"},
		{Name: "Opus 每周", Percent: 78, Reset: "周四重置"},
	}
	u.LimitsUpdated = "2 分钟前"
	u.Tips = []core.Tip{{Level: 2, Title: "周限额已用尽", Desc: "切换到 Sonnet 或等待重置"}}
	return u
}

func usageCacheLow() core.UsageReport {
	u := synthDays(30, "30 天", func(i int) float64 { return dayCost(i) })
	u.Total = sumDays(u.Days)
	u.CacheHit = 12
	u.Tips = []core.Tip{{Level: 1, Title: "缓存命中率 12%", Desc: "频繁改上下文导致缓存浪费"}}
	return u
}

func usageTopLong() core.UsageReport {
	u := synthDays(30, "30 天", func(i int) float64 { return dayCost(i) })
	u.Total = sumDays(u.Days)
	fillModelsStandard(&u)
	fillProjectsMany(&u)
	for i := 0; i < 20; i++ {
		u.TopSessions = append(u.TopSessions, core.CostSession{
			Title:   fmt.Sprintf("高费用会话 #%02d · %s", i+1, []string{"重构", "审查", "测试", "调研"}[i%4]),
			Project: []string{"router-demo", "web-dashboard", "scenario-lab"}[i%3],
			Model:   []string{"Sonnet 5.5", "Opus 5.5", "Haiku 5.5"}[i%3],
			Cost:    0.4 + float64(20-i)*0.08,
		})
	}
	return u
}

func synthDays(n int, label string, fn func(int) float64) core.UsageReport {
	today := time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC)
	u := core.UsageReport{RangeName: label}
	for i := 0; i < n; i++ {
		d := today.AddDate(0, 0, -(n - 1 - i))
		c := math.Round(fn(i)*100) / 100
		u.Days = append(u.Days, core.Day{Label: strconv.Itoa(int(d.Month())) + " 月 " + strconv.Itoa(d.Day()) + " 日", Cost: c})
	}
	return u
}

func sumDays(days []core.Day) float64 {
	var t float64
	for _, d := range days {
		t += d.Cost
	}
	return math.Round(t*100) / 100
}
