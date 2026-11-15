package demo

import (
	"github.com/ixxxxoooo/minicode/internal/core"
)

func init() {
	Register(Scenario{
		ID: "files", Title: "工具卡 · 文件读写:路由层梳理", Project: "router-demo",
		Label: "刚刚", Group: "今天", Clock: "刚刚",
		Build: buildScnFiles,
	})
	Register(Scenario{
		ID: "tools-read", Title: "工具卡 · 读取与搜索样式大全", Project: "",
		Build: buildScnToolsRead,
	})
}

// buildScnFiles 读取与编辑混合的真实感回合(侧栏 id files)。
func buildScnFiles() []Step {
	s := NewScript("files", "工具卡 · 文件读写:路由层梳理")
	u := s.User("先梳理 @{src/router.rs} 和 handlers,把路由匹配和错误出口整理清楚。")
	s.Think("先读路由表和 handlers 入口,再搜 match_path 的调用点,最后小步编辑。", 3)

	s.Tool(core.NewToolCall("", "Read", "读取 src/router.rs").
		WithInput(map[string]any{"file_path": "src/router.rs", "offset": "40", "limit": "20"}).
		WithLocation("src/router.rs"), core.ToolCompleted, 420,
		&core.TextBlock{Text: readSnippet(40, 20)})

	grpRead := []*core.ToolCall{
		core.NewToolCall("gr1", "Grep", "搜索 match_path").WithInput(map[string]any{"pattern": "match_path", "path": "src/", "output_mode": "content"}),
		core.NewToolCall("gr2", "Read", "读取 src/handlers/mod.rs").WithInput(map[string]any{"file_path": "src/handlers/mod.rs"}).WithLocation("src/handlers/mod.rs"),
		core.NewToolCall("gr3", "Glob", "查找 **/*error*.rs").WithInput(map[string]any{"pattern": "src/**/*error*.rs"}),
	}
	for _, tc := range grpRead {
		s.Add(ms(80), core.EvAppend{Item: tc})
	}
	s.Add(ms(400), core.EvToolUpdate{ID: "gr1", Patch: core.ToolPatch{Status: core.Status(core.ToolCompleted), DurationMs: 380,
		AppendContent: []core.ToolContent{&core.TextBlock{Text: grepContentResult()}}}})
	s.Add(ms(80), core.EvToolUpdate{ID: "gr2", Patch: core.ToolPatch{Status: core.Status(core.ToolCompleted), DurationMs: 310,
		AppendContent: []core.ToolContent{&core.TextBlock{Text: readSnippet(1, 48)}}}})
	s.Add(ms(80), core.EvToolUpdate{ID: "gr3", Patch: core.ToolPatch{Status: core.Status(core.ToolCompleted), DurationMs: 290,
		AppendContent: []core.ToolContent{&core.TextBlock{Text: "src/handlers/error.rs"}}}})

	oldH := "pub mod routes;\n"
	newH := "pub mod routes;\npub mod error;\n"
	s.Tool(core.NewToolCall("", "Edit", "编辑 src/handlers/mod.rs").
		WithInput(map[string]any{"file_path": "src/handlers/mod.rs"}).WithLocation("src/handlers/mod.rs"),
		core.ToolCompleted, 520, &core.DiffBlock{Path: "src/handlers/mod.rs", Old: oldH, New: newH, Line: 1})

	s.Say("已阅读路由与 handlers 结构,并补上 `error` 模块入口。下一步可以把 `match_path` 的归一化与错误映射串起来。")
	s.End(u, 52, core.StopEndTurn, DefaultStats(3))
	return s.Steps()
}

// buildScnToolsRead 集中展示读取 / 搜索类工具样式。
func buildScnToolsRead() []Step {
	s := NewScript("tools-read", "工具卡 · 读取与搜索样式大全")
	u := s.User("把读取、搜索、列目录、无匹配、大量结果和探索组(含失败)都演示一遍。")
	s.Think("按设计稿把 Read / Grep / Glob / LS 的展示态逐个造出来。", 2)

	patchTool(s, core.NewToolCall("r1", "Read", "读取 src/router.rs · 1–24 行").
		WithInput(map[string]any{"file_path": "src/router.rs", "offset": "1", "limit": "24"}).
		WithLocation("src/router.rs"), core.ToolCompleted, 360, &core.TextBlock{Text: readSnippet(1, 24)})

	patchTool(s, core.NewToolCall("g1", "Grep", "搜索 TODO").WithInput(map[string]any{
		"pattern": "TODO", "path": "src/", "output_mode": "content",
	}), core.ToolCompleted, 410, &core.TextBlock{Text: grepContentResult()})

	patchTool(s, core.NewToolCall("g2", "Grep", "搜索 legacy_handler").WithInput(map[string]any{
		"pattern": "legacy_handler", "path": ".", "output_mode": "files_with_matches",
	}), core.ToolCompleted, 390, &core.TextBlock{Text: grepFilesResult()})

	patchTool(s, core.NewToolCall("gl", "Glob", "查找 **/*.rs").WithInput(map[string]any{"pattern": "src/**/*.rs"}),
		core.ToolCompleted, 280, &core.TextBlock{Text: "src/router.rs\nsrc/server.rs\nsrc/handlers/mod.rs"})

	patchTool(s, core.NewToolCall("ls1", "LS", "列出 src/handlers").WithInput(map[string]any{"path": "src/handlers"}),
		core.ToolCompleted, 220, &core.TextBlock{Text: lsListing()})

	patchTool(s, core.NewToolCall("gn", "Grep", "搜索 __no_such_symbol__").WithInput(map[string]any{
		"pattern": "__no_such_symbol__", "path": "src/",
	}), core.ToolCompleted, 190, &core.TextBlock{Text: "No matches found"})

	patchTool(s, core.NewToolCall("gb", "Grep", "搜索 test_").WithInput(map[string]any{
		"pattern": "test_", "path": "src/", "output_mode": "files_with_matches",
	}), core.ToolCompleted, 650, &core.TextBlock{Text: longGrepResult()})

	// 探索组:连续读 / 搜,中间夹一条失败(含失败的组在设计稿里应默认展开)
	fail := core.NewToolCall("gf", "Read", "读取不存在的文件").WithInput(map[string]any{"file_path": "src/missing.rs"})
	s.Add(ms(100), core.EvAppend{Item: core.NewToolCall("ex1", "Read", "读取 Cargo.toml").
		WithInput(map[string]any{"file_path": "Cargo.toml"}).WithLocation("Cargo.toml")})
	s.Add(ms(60), core.EvAppend{Item: core.NewToolCall("ex2", "Grep", "搜索 edition").WithInput(map[string]any{"pattern": "edition", "path": "."})})
	s.Add(ms(60), core.EvAppend{Item: fail})
	s.Add(ms(300), core.EvToolUpdate{ID: "ex1", Patch: core.ToolPatch{Status: core.Status(core.ToolCompleted), DurationMs: 200,
		AppendContent: []core.ToolContent{&core.TextBlock{Text: "edition = \"2024\"\nname = \"router-demo\"\n"}}}})
	s.Add(ms(80), core.EvToolUpdate{ID: "ex2", Patch: core.ToolPatch{Status: core.Status(core.ToolCompleted), DurationMs: 180,
		AppendContent: []core.ToolContent{&core.TextBlock{Text: "Cargo.toml:2:edition = \"2024\""}}}})
	s.Add(ms(80), core.EvToolUpdate{ID: "gf", Patch: core.ToolPatch{Status: core.Status(core.ToolFailed), DurationMs: 90,
		AppendContent: []core.ToolContent{&core.TextBlock{Text: "File not found: src/missing.rs"}}}})

	s.Say("读取与搜索类工具卡的样式已齐:行号范围、内容 / 文件列表模式、Glob、LS、无匹配、大量命中与探索分组(含失败)。")
	s.End(u, 38, core.StopEndTurn, DefaultStats(7))
	return s.Steps()
}
