package demo

import "github.com/ixxxxoooo/minicode/internal/core"

// Projects 是演示模式的侧栏数据(与设计稿一致)。
func Projects() []core.ProjectInfo {
	s := func(id, title, label string) core.SessionInfo {
		return core.SessionInfo{ID: id, Title: title, Label: label}
	}
	return []core.ProjectInfo{
		{ID: "router-demo", Name: "router-demo", Path: "/demo/router-demo", Count: 4, Sessions: []core.SessionInfo{
			s("fixbug", "完整修复一个 Bug:路由大小写匹配", "刚刚"),
			s("perm", "权限请求:修改配置与执行命令", "5 分钟"),
			s("files", "文件读写:梳理并整理路由层", "1 小时"),
			s("md", "Markdown 与代码高亮:发布说明", "昨天"),
		}},
		{ID: "web-dashboard", Name: "web-dashboard", Path: "/demo/web-dashboard", Count: 2, Sessions: []core.SessionInfo{
			s("browser", "内置浏览器:排查指标看板报错", "2 小时"),
			s("subagent", "子 Agent:代码审查与调研", "昨天"),
		}},
		{ID: "docs-site", Name: "docs-site", Path: "/demo/docs-site", Count: 6},
		{ID: "scenario-lab", Name: "scenario-lab", Path: "/demo/scenario-lab", Count: 61},
	}
}
