package app

import (
	"strings"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// usageModelCost 把模型费用格式化成展示文案;无价目表且费用为 0 时写「未知费用」。
func usageModelCost(name string, cost float64) string {
	if cost >= 0.00005 {
		return core.FormatMoney(cost)
	}
	low := strings.ToLower(name)
	for _, fam := range []string{"sonnet", "opus", "haiku"} {
		if strings.Contains(low, fam) {
			return core.FormatMoney(cost)
		}
	}
	if cost < 1e-9 {
		return "未知费用"
	}
	return core.FormatMoney(cost)
}
