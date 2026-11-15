package demo

import (
	"github.com/ixxxxoooo/minicode/internal/core"
)

// AcctMenu 演示模式账号菜单的展示数据(设计稿 shell.js 对齐,不写死在界面里)。
type AcctMenu struct {
	Initial   string // 头像一字
	Name      string
	Subtitle  string // 如「Claude Max · 官方账号」
	MonthCost string // 已格式化的本月费用
	FiveLabel string // 限额项名称,如「5 小时」
	FivePct   int    // 百分比;-1 = 不显示进度条
}

// AcctMenuDemo 返回演示账号菜单数据。
func AcctMenuDemo() AcctMenu {
	u := Usage(2)
	pct := 38
	if len(u.Limits) > 0 {
		pct = u.Limits[0].Percent
	}
	return AcctMenu{
		Initial:   "L",
		Name:      "demo",
		Subtitle:  "Claude Max · 官方账号",
		MonthCost: core.FormatMoney(u.Total),
		FiveLabel: "5 小时",
		FivePct:   pct,
	}
}
