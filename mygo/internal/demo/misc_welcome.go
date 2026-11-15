// 演示数据索引(欢迎页与命令面板)
//
//   - WelcomeStatusLine:新会话欢迎页问候下的状态行(proj 为空 / 演示扩展)。
//   - WelcomeRecents:欢迎页最近会话行(设计稿 status.js 有,Go 欢迎页可展开见下方)。
//   - PaletteExtraFiles:命令面板无查询时「文件」分组补项(沙盒路径)。
package demo

// WelcomeStatusLine 演示模式欢迎页状态(含 MCP 故障提示变体)。
func WelcomeStatusLine(variant string) string {
	switch variant {
	case "clean":
		return "main · 工作区干净 · 没有最近会话"
	case "mcp-fail":
		return "main · 3 项未提交改动 · MCP 有 1 个服务器连接失败"
	case "review":
		return "feat/charts · 3 项待审查改动 · 上次会话 5 分钟前"
	default:
		return "main · 3 项未提交改动 · 上次会话 5 分钟前"
	}
}

// WelcomeRecent 欢迎页下方最近会话(标题, 时间文案)。
func WelcomeRecent() [][2]string {
	return [][2]string{
		{"完整修复一个 Bug:路由大小写匹配", "刚刚"},
		{"权限请求:修改配置与执行命令", "5 分钟"},
		{"Markdown 与代码高亮:发布说明", "昨天"},
	}
}

// PaletteExtraFiles 补充命令面板「最近文件」(相对沙盒路径)。
func PaletteExtraFiles() []string {
	return []string{
		"router-demo/src/router.rs",
		"router-demo/src/router_test.rs",
		"router-demo/Cargo.toml",
		"web-dashboard/src/charts.ts",
		"docs-site/docs/routing.md",
		"design-assets/palette.md",
	}
}
