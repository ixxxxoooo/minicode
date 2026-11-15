package core

// 用量与成本(整页「用量与成本」读它)。真实数据来自扫描 ~/.claude/projects 的会话记录,
// 演示数据由 demo.UsageReport 按确定性公式造出;界面只认这里的结构,两边互换。

// Day 是某一天的费用。
type Day struct {
	Label string // 如「11 月 15 日」
	Cost  float64
}

// Share 是一个占比项(按模型 / 按项目)。
type Share struct {
	Name     string
	Fraction float64 // 0..1,模型用
	Cost     float64
	Sessions int // 项目用
	Kind     int // 模型色:0 info / 1 紫 / 2 ok
}

// CostSession 是一个费用高的会话。
type CostSession struct {
	Title, Project, Model string
	Cost                  float64
}

// Limit 是账户限额的一项(5 小时窗口 / 每周…)。
type Limit struct {
	Name    string
	Percent int
	Reset   string
}

// Tip 是一条节省建议。
type Tip struct {
	Level       int // 0 好 / 1 提醒 / 2 信息
	Title, Desc string
}

// UsageReport 是一个时间范围内的用量汇总。
type UsageReport struct {
	RangeName     string
	Days          []Day
	Total         float64
	DeltaPercent  float64 // 较上一段的变化(正 = 增加)
	InputTokens   string  // 已格式化,如「5.0M」
	OutputTokens  string
	Sessions      int
	CacheHit      int // 缓存命中率 %
	Models        []Share
	Projects      []Share
	TopSessions   []CostSession
	Limits        []Limit
	LimitsUpdated string // 「3 分钟前」
	Tips          []Tip
}

// MaxDay 返回单日最高费用(画柱子比例用,至少一个极小正数避免除零)。
func (u UsageReport) MaxDay() float64 {
	m := 0.0001
	for _, d := range u.Days {
		if d.Cost > m {
			m = d.Cost
		}
	}
	return m
}

// UsageCSV 导出每日费用的 CSV(带表头)。
func UsageCSV(u UsageReport) string {
	b := []byte("日期,费用(美元)\n")
	for _, d := range u.Days {
		b = append(b, []byte(d.Label+","+FormatMoney(d.Cost)[1:]+"\n")...)
	}
	return string(b)
}

// FormatMoney 把金额格式化成「$12.40」。
func FormatMoney(v float64) string {
	n := int(v*100 + 0.5)
	s := "$" + itoa(n/100) + "."
	if c := n % 100; c < 10 {
		s += "0" + itoa(c)
	} else {
		s += itoa(c)
	}
	return s
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
