package app

import (
	"fmt"
	"math"

	"github.com/ixxxxoooo/minicode/internal/core"
)

func usageDayMean(u core.UsageReport) float64 {
	if len(u.Days) == 0 {
		return 0
	}
	var sum float64
	for _, d := range u.Days {
		sum += d.Cost
	}
	return sum / float64(len(u.Days))
}

func usageDayBarTip(label string, cost, mean float64) string {
	cmp := "与均值持平"
	if mean > 0 {
		ratio := (cost - mean) / mean * 100
		if math.Abs(ratio) >= 0.5 {
			if ratio > 0 {
				cmp = fmt.Sprintf("高于均值 %.0f%%", ratio)
			} else {
				cmp = fmt.Sprintf("低于均值 %.0f%%", -ratio)
			}
		}
	}
	return fmt.Sprintf("%s · 费用 %s · %s", label, core.FormatMoney(cost), cmp)
}

func usageModelTip(name string, frac float64, cost float64, total float64) string {
	pct := int(frac*100 + 0.5)
	costTxt := usageModelCost(name, cost)
	if total > 0 && cost < 0.00005 {
		costTxt = fmt.Sprintf("约占 %d%% 总费用", pct)
	}
	return fmt.Sprintf("%s · %s · 占比 %d%%", name, costTxt, pct)
}

func usageProjectTip(name string, sessions int, cost float64) string {
	return fmt.Sprintf("%s · %d 个会话 · 费用 %s", name, sessions, core.FormatMoney(cost))
}

func usageLimitTip(name string, pct int, reset string) string {
	if reset == "" {
		return fmt.Sprintf("%s · 已用 %d%%", name, pct)
	}
	return fmt.Sprintf("%s · 已用 %d%% · %s", name, pct, reset)
}

func usageStatTip(value, label, sub string) string {
	if sub == "" {
		return fmt.Sprintf("%s · %s", value, label)
	}
	return fmt.Sprintf("%s · %s · %s", value, label, sub)
}
