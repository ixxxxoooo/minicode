package demo

import c "github.com/ixxxxoooo/minicode/internal/customize"

// Customize 是演示用的自定义数据(与设计稿 customize.js 的 DATA 一致)。
func Customize() *c.Store {
	tools := func(kv ...any) []c.Tool {
		var out []c.Tool
		for i := 0; i < len(kv); i += 2 {
			out = append(out, c.Tool{Name: kv[i].(string), On: kv[i+1].(bool)})
		}
		return out
	}
	inc := func(kind string, n int) c.Inc { return c.Inc{Kind: kind, N: n, On: true} }
	items := map[c.Cat][]*c.Item{
		c.Plugins: {
			{Name: "superpowers", Desc: "官方市场 · v4.2.0 · 12 个 Skills", Scope: c.User, On: true, Ver: "4.2.0", Upd: "4.3.0", Src: "claude-plugins-official",
				About: "一套工程方法论 Skills:先梳理需求、先写失败的测试、系统化调试。", Inc: []c.Inc{inc("Skills", 12), inc("斜杠命令", 3), inc("钩子", 1)}},
			{Name: "code-review", Desc: "官方市场 · v1.3.0 · 2 个命令", Scope: c.User, On: true, Ver: "1.3.0", Src: "claude-plugins-official",
				About: "按严重程度审查当前改动,并给出可直接应用的修改建议。", Inc: []c.Inc{inc("斜杠命令", 2), inc("子 Agent", 1)}},
			{Name: "commit-helper", Desc: "本地 · v0.2.0", Scope: c.Project, On: true, Ver: "0.2.0", Src: "本地目录 ~/plugins/commit-helper",
				About: "根据暂存区改动生成符合 Conventional Commits 的提交说明。", Inc: []c.Inc{inc("斜杠命令", 1), inc("钩子", 1)}},
			{Name: "design-kit", Desc: "官方市场 · v0.9.1", Scope: c.User, Ver: "0.9.1", Src: "claude-plugins-official",
				About: "设计令牌检查与组件审查。", Inc: []c.Inc{inc("Skills", 4)}},
		},
		c.MCP: {
			{Name: "github", Desc: "stdio · 14 个工具 · 已连接", Scope: c.User, On: true, Status: c.StOK, Transport: "stdio", Cmd: "npx -y @modelcontextprotocol/server-github",
				Env: []string{"GITHUB_TOKEN"}, Tok: "3.2k token · 1.6%", OnDemand: true,
				Tools: tools("create_issue", true, "list_pull_requests", true, "get_file_contents", true, "search_code", true, "create_branch", false, "merge_pull_request", false)},
			{Name: "filesystem", Desc: "stdio · 8 个工具 · 已连接", Scope: c.Project, On: true, Status: c.StOK, Transport: "stdio", Cmd: "npx -y @modelcontextprotocol/server-filesystem ~/Projects",
				Tok: "按需加载,不占上下文", OnDemand: true, Tools: tools("read_file", true, "write_file", true, "list_directory", true, "move_file", false)},
			{Name: "figma", Desc: "http · 需要授权", Scope: c.User, On: true, Status: c.StWarn, Transport: "http", Cmd: "https://mcp.figma.com/mcp", Tok: "连接后统计", OnDemand: true},
			{Name: "postgres", Desc: "stdio · 连接失败:ECONNREFUSED", Scope: c.Local, Status: c.StErr, Transport: "stdio", Cmd: "npx -y @modelcontextprotocol/server-postgres",
				Env: []string{"DATABASE_URL"}, Tok: "—", OnDemand: true},
		},
		c.Skills: {
			{Name: "brainstorming", Desc: "插件 · 动手前先梳理需求", Scope: c.User, On: true, Path: "插件 superpowers / brainstorming", Trig: []string{"想做一个新功能", "帮我设计"},
				Files: []c.File{{Name: "SKILL.md", Active: true}, {Name: "examples.md"}}},
			{Name: "test-driven-development", Desc: "插件 · 先写失败的测试", Scope: c.User, On: true, Path: "插件 superpowers / test-driven-development", Trig: []string{"补测试", "修这个 bug"},
				Files: []c.File{{Name: "SKILL.md", Active: true}}},
			{Name: "release-notes", Desc: "项目 · 生成发布说明", Scope: c.Project, On: true, Path: ".claude/skills/release-notes", Trig: []string{"发布说明", "changelog", "这个版本改了什么"},
				Files: []c.File{{Name: "SKILL.md", Active: true}, {Name: "template.md"}, {Name: "scripts", Dir: true}, {Name: "collect.sh", Indent: 1}}},
			{Name: "sql-review", Desc: "用户 · 审查 SQL", Scope: c.User, Path: "~/.claude/skills/sql-review", Trig: []string{"审查 SQL", "这条查询慢"},
				Files: []c.File{{Name: "SKILL.md", Active: true}}},
		},
		c.Commands: {
			{Name: "/review", Desc: "项目 · 审查当前改动", Scope: c.Project, On: true, Note: "审查当前 git diff", Hint: "[PR 编号]", Allow: []string{"Bash(git diff:*)", "Read", "Grep"},
				Body: "审查当前 git diff,按严重程度列出问题。\n如果提供了 $ARGUMENTS,只审查该 PR 的改动。\n每个问题给出文件、行号与建议的修改。"},
			{Name: "/test", Desc: "项目 · 跑测试并总结失败", Scope: c.Project, On: true, Note: "运行测试并总结失败", Hint: "[过滤条件]", Allow: []string{"Bash(cargo test:*)"}, Model: 1,
				Body: "运行 cargo test $ARGUMENTS,只总结失败的用例与最可能的原因。"},
			{Name: "/release", Desc: "用户 · 生成并发布版本", Scope: c.User, On: true, Note: "生成发布说明并打 tag", Hint: "[版本号]", Allow: []string{"Bash(git:*)", "Bash(gh:*)"},
				Body: "读取自上个 tag 以来的提交,生成发布说明,然后创建 $ARGUMENTS 的 tag 与 GitHub Release。"},
			{Name: "/clean", Desc: "用户 · 清理构建产物", Scope: c.User, On: true, Note: "清理 target 与 dist", Allow: []string{"Bash(rm:*)"}, Model: 3,
				Body: "清理 target 与 dist 目录,汇报释放的空间。"},
		},
		c.Agents: {
			{Name: "code-reviewer", Desc: "项目 · Sonnet · 只读", Scope: c.Project, On: true, Note: "审查代码的正确性与风格,不修改文件", Model: 1, Color: 1, Allow: []string{"Read", "Grep", "Glob"},
				Body: "你是严格的代码审查员。只读代码,不修改任何文件。\n按「正确性 → 安全 → 可读性」的顺序给出问题,每条带文件与行号。"},
			{Name: "researcher", Desc: "用户 · Haiku · 联网", Scope: c.User, On: true, Note: "联网调研并整理出处", Model: 3, Color: 3, Allow: []string{"Read", "WebFetch", "WebSearch"},
				Body: "你负责联网调研。每个结论都要附来源链接,区分事实与推测。"},
			{Name: "test-writer", Desc: "项目 · Sonnet · 可编辑", Scope: c.Project, On: true, Note: "为改动补充单元测试", Model: 1, Color: 2, Allow: []string{"Read", "Grep", "Glob", "Edit", "Write", "Bash"},
				Body: "为最近的改动补充单元测试,覆盖边界与失败路径。先写失败的测试,再确认通过。"},
		},
		c.Rules: {
			{Name: "~/.claude/CLAUDE.md", Desc: "用户 · 1.2 KB", Scope: c.User, On: true, Size: "1.2 KB", Tokens: 320, Body: "# 通用约定\n\n- 回答、注释一律中文\n- 先说明理解的需求,再动手\n- 不读取、不输出任何密钥"},
			{Name: "router-demo/CLAUDE.md", Desc: "项目 · 3.4 KB", Scope: c.Project, On: true, Size: "3.4 KB", Tokens: 900, Body: "# router-demo\n\n- 提交信息用 Conventional Commits\n- 不提交构建产物(target / dist)\n- 改完先 `cargo test`,再提交推送"},
			{Name: "CLAUDE.local.md", Desc: "本地 · 0.2 KB", Scope: c.Local, On: true, Size: "0.2 KB", Tokens: 60, Body: "# 本地笔记\n\n- 我的测试库端口是 5433"},
		},
		c.Hooks: {
			{Name: "PreToolUse · Bash", Desc: "项目 · 拦截含 rm -rf 的命令", Scope: c.Project, On: true, Event: 0, Match: "Bash", Cmd: "scripts/deny-rm-rf.sh", Timeout: 10},
			{Name: "PostToolUse · Edit", Desc: "项目 · 保存后运行 cargo fmt", Scope: c.Project, On: true, Event: 1, Match: "Edit|Write", Cmd: "cargo fmt --all", Timeout: 10, Trusted: true},
			{Name: "Stop", Desc: "用户 · 回合结束时发系统通知", Scope: c.User, On: true, Event: 2, Cmd: `osascript -e 'display notification "回合结束"'`, Timeout: 5, Trusted: true},
		},
	}
	market := []*c.Market{
		{Name: "frontend-design", By: "官方", Desc: "让界面设计有风格,而不是千篇一律", Stars: "4.2k"},
		{Name: "security-guidance", By: "官方", Desc: "写代码时提示常见的安全隐患", Stars: "2.8k"},
		{Name: "pr-review-toolkit", By: "官方", Desc: "一组专门审查 PR 的子 Agent", Stars: "3.1k"},
		{Name: "code-review", By: "官方", Desc: "按严重程度审查当前改动", Stars: "5.6k", Installed: true},
		{Name: "superpowers", By: "官方", Desc: "工程方法论 Skills 合集", Stars: "9.4k", Installed: true},
		{Name: "explanatory-output-style", By: "官方", Desc: "边做边解释关键决策", Stars: "1.1k"},
	}
	return c.NewStore(items, market)
}
