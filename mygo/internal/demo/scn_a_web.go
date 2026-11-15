package demo

import (
	"github.com/ixxxxoooo/minicode/internal/core"
)

func init() {
	Register(Scenario{
		ID: "axum", Title: "工具卡 · 调研:axum 与 actix", Project: "router-demo",
		Label: "10月28日", Group: "更早", Clock: "10月28日",
		Build: buildScnAxum,
	})
	Register(Scenario{
		ID: "browser", Title: "工具卡 · 浏览器与 MCP 工具链", Project: "web-dashboard",
		Label: "2 小时", Group: "今天",
		Build: buildScnBrowser,
	})
	Register(Scenario{
		ID: "tools-web", Title: "工具卡 · 联网与杂项工具", Project: "",
		Build: buildScnToolsWeb,
	})
}

func buildScnAxum() []Step {
	s := NewScript("axum", "工具卡 · 调研:axum 与 actix")
	u := s.User("对比 axum 与 actix 的路由写法,写一份简短结论(含表格)。")
	s.Think("先项目内搜索,再 WebSearch / WebFetch 补官方文档。", 3)

	s.Tool(core.NewToolCall("", "Grep", "搜索 Router::").WithInput(map[string]any{"pattern": "Router::", "path": "src/"}),
		core.ToolCompleted, 400, &core.TextBlock{Text: grepContentResult()})

	search := "1. **axum** — 基于 tower/hyper,宏路由与 `Router::nest` 组合清晰。\n2. **actix** — 基于 actor,`web::scope` 与资源模型成熟。\n3. **路径大小写** — 两者默认都区分,需在中间件归一化。\n"
	s.Tool(core.NewToolCall("", "WebSearch", "搜索 axum vs actix routing").WithInput(map[string]any{
		"query": "axum vs actix routing comparison 2025",
	}), core.ToolCompleted, 1800, &core.TextBlock{Text: search})

	fetch := "# Routing\n\naxum `Router` chains matchers; actix uses `App::service` scopes.\n"
	s.Tool(core.NewToolCall("", "WebFetch", "抓取 docs.rs/axum").WithInput(map[string]any{
		"url": "https://docs.rs/axum/latest/axum/routing/",
	}), core.ToolCompleted, 2200, &core.TextBlock{Text: fetch})

	conclusion := "### 结论\n\n| 维度 | axum | actix |\n| --- | --- | --- |\n| 路由组合 | `Router::nest` / `merge` | `scope` / `resource` |\n| 中间件 | `tower::Layer` | `wrap` |\n| 学习曲线 | 低(类型驱动) | 中(actor 模型) |\n\n建议本项目继续 axum,并在中间件里统一路径归一化。"
	s.Say(conclusion)
	s.End(u, 95, core.StopEndTurn, DefaultStats(4))
	return s.Steps()
}

func buildScnBrowser() []Step {
	s := NewScript("browser", "工具卡 · 浏览器与 MCP 工具链")
	u := s.User("指标看板报错,用浏览器 MCP 打开页面、截图并调 GitHub MCP 查 issue。")
	s.Think("Playwright 截图 + GitHub MCP,串成一条工具链。", 2)

	okMCP := core.NewToolCall("", "mcp__github__get_issue", "github · get_issue").
		WithInput(map[string]any{"owner": "demo", "repo": "web-dashboard", "issue_number": 42})
	s.Tool(okMCP, core.ToolCompleted, 890, &core.TextBlock{Text: "{\"title\":\"Dashboard 500 on /metrics\",\"state\":\"open\"}"})

	failMCP := core.NewToolCall("", "mcp__github__create_issue", "github · create_issue").
		WithInput(map[string]any{"owner": "demo", "repo": "web-dashboard", "title": "test"})
	s.Tool(failMCP, core.ToolFailed, 420, &core.TextBlock{Text: "403 Forbidden: token lacks repo scope"})

	shot := core.NewToolCall("", "mcp__playwright__browser_take_screenshot", "playwright · browser_take_screenshot").
		WithInput(map[string]any{"url": "http://localhost:3000/metrics", "fullPage": true})
	s.Tool(shot, core.ToolCompleted, 3100,
		&core.TextBlock{Text: "已打开指标看板,捕获整页截图。"},
		&core.ImageBlock{Label: "metrics-error.png", Path: "/demo/screens/metrics-error.png"},
	)

	s.Say("GitHub MCP 读到 open issue,Playwright 已附图;创建 issue 因权限失败应显示错误态。")
	s.End(u, 48, core.StopEndTurn, DefaultStats(6))
	return s.Steps()
}

func buildScnToolsWeb() []Step {
	s := NewScript("tools-web", "工具卡 · 联网与杂项工具")
	u := s.User("补 WebFetch、WebSearch、ToolSearch、LSP、NotebookEdit 与未知工具名。")

	s.Tool(core.NewToolCall("", "WebSearch", "搜索 MCP 协议").WithInput(map[string]any{"query": "model context protocol tools"}),
		core.ToolCompleted, 1500, &core.TextBlock{Text: "Model Context Protocol — specification 2025-03-26"})

	s.Tool(core.NewToolCall("", "WebFetch", "抓取 example.com").WithInput(map[string]any{"url": "https://example.com"}),
		core.ToolCompleted, 900, &core.TextBlock{Text: "Example Domain\n\nThis domain is for use in documentation."})

	s.Tool(core.NewToolCall("", "ToolSearch", "查找工具 diff").WithInput(map[string]any{"query": "diff viewer"}),
		core.ToolCompleted, 320, &core.TextBlock{Text: "Bash, Read, Grep, WebFetch"})

	s.Tool(core.NewToolCall("", "LSP", "代码智能 · goto_definition").WithInput(map[string]any{
		"operation": "goto_definition", "file_path": "src/router.rs", "line": 41,
	}), core.ToolCompleted, 540, &core.TextBlock{Text: "src/router.rs:12 — fn normalize(path: &str) -> String"})

	s.Tool(core.NewToolCall("", "NotebookEdit", "编辑 notebook 单元").WithInput(map[string]any{
		"notebook_path": "notebooks/ci.ipynb", "cell_id": "c1",
	}), core.ToolCompleted, 610, &core.DiffBlock{
		Path: "notebooks/ci.ipynb", Old: `"cargo test"`, New: `"cargo test --test integration"`, Line: 3,
	})

	s.Tool(core.NewToolCall("", "CustomEnterpriseTool", "同步设计令牌").WithInput(map[string]any{"mode": "pull"}),
		core.ToolCompleted, 0, &core.TextBlock{Text: "OK — 12 tokens updated"})

	s.Say("杂项工具落在 KindOther / MCP / Fetch 等步骤行上,未知工具名走默认样式。")
	s.End(u, 32, core.StopEndTurn, DefaultStats(2))
	return s.Steps()
}
