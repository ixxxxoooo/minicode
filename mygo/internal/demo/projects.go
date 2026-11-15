package demo

import "github.com/ixxxxoooo/minicode/internal/core"

// Projects 是演示模式的侧栏数据(与设计稿一致)。
func Projects() []core.ProjectInfo {
	s := func(id, title, label string, g ...string) core.SessionInfo {
		si := core.SessionInfo{ID: id, Title: title, Label: label, Group: "今天", Clock: label}
		if len(g) == 2 {
			si.Group, si.Clock = g[0], g[1]
		}
		return si
	}
	ps := []core.ProjectInfo{
		{ID: "router-demo", Name: "router-demo", Path: "/demo/router-demo", Count: 4, Sessions: []core.SessionInfo{
			s("fixbug", "完整修复一个 Bug:路由大小写匹配", "刚刚"),
			s("perm", "权限请求:修改配置与执行命令", "5 分钟"),
			s("files", "文件读写:梳理并整理路由层", "1 小时"),
			s("md", "Markdown 与代码高亮:发布说明", "昨天", "昨天", "18:20"),
			s("ci", "补充集成测试并修复 CI", "昨天", "昨天", "11:02"),
			s("refactor", "重构 handlers 模块的错误处理", "周一", "更早", "周一"),
			s("long", "2000 轮长会话:滚动与加载", "11月2日", "更早", "11月2日"),
			s("axum", "调研:axum 与 actix 的路由差异", "10月28日", "更早", "10月28日"),
		}},
		{ID: "web-dashboard", Name: "web-dashboard", Path: "/demo/web-dashboard", Count: 2, Sessions: []core.SessionInfo{
			s("browser", "内置浏览器:排查指标看板报错", "2 小时"),
			s("subagent", "子 Agent:代码审查与调研", "昨天"),
		}},
		{ID: "docs-site", Name: "docs-site", Path: "/demo/docs-site", Count: 6},
		{ID: "scenario-lab", Name: "scenario-lab", Path: "/demo/scenario-lab"},
	}
	ps = append(ps, ExtraProjects()...)
	// 把登记的场景并入侧栏:id 已在基础列表里的只换剧本;其余按 Project 归入对应项目(默认场景实验室)
	have := map[string]bool{}
	for _, p := range ps {
		for _, x := range p.Sessions {
			have[x.ID] = true
		}
	}
	for _, sc := range Scenarios() {
		if have[sc.ID] {
			continue
		}
		have[sc.ID] = true
		proj := sc.Project
		if proj == "" {
			proj = "scenario-lab"
		}
		for i := range ps {
			if ps[i].ID == proj {
				ps[i].Sessions = append(ps[i].Sessions, sc.sessionInfo())
			}
		}
	}
	for i := range ps {
		if len(ps[i].Sessions) > ps[i].Count {
			ps[i].Count = len(ps[i].Sessions)
		}
	}
	return ps
}
