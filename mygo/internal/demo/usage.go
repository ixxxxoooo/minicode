package demo

import (
	"math"
	"strconv"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// 演示用的用量数据:与设计稿同一套确定性公式(越靠近今天越高,周末偏低)。

var usageRanges = []struct {
	Name string
	Days int
}{{"7 天", 7}, {"30 天", 30}, {"本月", 15}}

// UsageRanges 是范围切换的三个选项。
func UsageRanges() []string {
	out := make([]string, len(usageRanges))
	for i, r := range usageRanges {
		out[i] = r.Name
	}
	return out
}

func dayCost(k int) float64 {
	w := k % 7
	v := .22 + .42*math.Abs(math.Sin(float64(k)*1.37+1))
	if w == 5 || w == 6 {
		v -= .14
	}
	return math.Max(.08, v) * 1.2
}

// Usage 返回第 r 个范围(0 = 7 天 / 1 = 30 天 / 2 = 本月)的演示用量。
func Usage(r int) core.UsageReport {
	if r < 0 || r >= len(usageRanges) {
		r = 1
	}
	n := usageRanges[r].Days
	today := time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC)
	u := core.UsageReport{RangeName: usageRanges[r].Name}
	for i := 0; i < n; i++ {
		k := 30 - n + i
		d := today.AddDate(0, 0, -(n - 1 - i))
		c := math.Round(dayCost(k)*100) / 100
		u.Days = append(u.Days, core.Day{Label: strconv.Itoa(int(d.Month())) + " 月 " + strconv.Itoa(d.Day()) + " 日", Cost: c})
		u.Total += c
	}
	prev := u.Total * .91
	if r == 0 {
		prev = u.Total * 1.18
	}
	u.DeltaPercent = (u.Total - prev) / prev * 100
	u.InputTokens = strconv.FormatFloat(u.Total*.31, 'f', 1, 64) + "M"
	u.OutputTokens = strconv.Itoa(int(u.Total*17)) + "k"
	u.Sessions = int(u.Total * 5.2)
	u.CacheHit = 90
	for _, m := range []struct {
		n    string
		p    float64
		kind int
	}{{"Sonnet 5.5", .72, 0}, {"Opus 5.5", .21, 1}, {"Haiku 5.5", .07, 2}} {
		u.Models = append(u.Models, core.Share{Name: m.n, Fraction: m.p, Cost: u.Total * m.p, Kind: m.kind})
	}
	for _, p := range []struct {
		n string
		v float64
		s int
	}{{"router-demo", 5.2, 28}, {"web-dashboard", 3.1, 17}, {"docs-site", 1.6, 9}, {"scenario-lab", .9, 10}} {
		u.Projects = append(u.Projects, core.Share{Name: p.n, Cost: p.v * u.Total / 10.8, Sessions: p.s})
	}
	top := func(title, proj, model string, cost float64) core.CostSession {
		return core.CostSession{Title: title, Project: proj, Model: model, Cost: cost}
	}
	u.TopSessions = []core.CostSession{
		top("完整修复一个 Bug:路由大小写匹配", "router-demo", "Sonnet 5.5", 1.42),
		top("2000 轮长会话:滚动与加载", "scenario-lab", "Opus 5.5", .86),
		top("权限请求:修改配置与执行命令", "router-demo", "Sonnet 5.5", .55),
		top("子 Agent:代码审查与调研", "web-dashboard", "Sonnet 5.5", .48),
	}
	lim := func(n string, p int, r string) core.Limit { return core.Limit{Name: n, Percent: p, Reset: r} }
	u.Limits = []core.Limit{lim("5 小时窗口", 38, "2 小时 10 分后重置"), lim("每周", 12, "周四重置"), lim("Opus 每周", 6, "周四重置")}
	u.LimitsUpdated = "3 分钟前"
	tip := func(l int, t, d string) core.Tip { return core.Tip{Level: l, Title: t, Desc: d} }
	u.Tips = []core.Tip{
		tip(0, "缓存命中率 90%", "长会话没有浪费输入 token"),
		tip(1, "Opus 占费用 21%", "简单任务切到 Sonnet 能省约 $0.8"),
		tip(2, "2000 轮长会话较贵", "用 /compact 压缩后再继续"),
	}
	return u
}
