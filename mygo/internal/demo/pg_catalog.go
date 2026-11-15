package demo

// 五页演示数据的套名与切换入口(命令面板「演示:…」);各 pg_*.go 文件头有详细索引。

// CanvasSetNames 画布演示套(0 = 默认 Q3 单张)。
func CanvasSetNames() []string {
	return []string{
		"画布 · 默认(Q3 路由层)",
		"画布 · 列表(空画布 / 样式总览 / 状态稿)",
		"画布 · 超长 300 块",
		"画布 · 30 个版本",
		"画布 · Claude 修改中",
	}
}

func AutomationsSetNames() []string {
	return []string{
		"自动化 · 默认",
		"自动化 · 空列表",
		"自动化 · 拥挤周视图(20 条)",
		"自动化 · 各种失败",
		"自动化 · 超长运行历史",
		"自动化 · 同时段重叠",
	}
}

func BrowserSetNames() []string {
	return []string{
		"浏览器 · 默认(docs.rs)",
		"浏览器 · 扩展文章库",
		"浏览器 · 抓取失败样例",
		"浏览器 · 无摘要与极简页",
	}
}

func CustomizeSetNames() []string {
	return []string{
		"自定义 · 默认",
		"自定义 · 各类空态",
		"自定义 · 拥挤列表(40 项)",
		"自定义 · 错误与长文案",
	}
}

func UsageSetNames() []string {
	return []string{
		"用量 · 默认",
		"用量 · 新账户(空)",
		"用量 · 本周爆表",
		"用量 · 单日峰值",
		"用量 · 多模型与第三方",
		"用量 · 历史 180 天",
		"用量 · 限额将满/已用尽",
		"用量 · 缓存极低",
		"用量 · Top 榜很长",
	}
}
