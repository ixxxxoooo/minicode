package demo

import (
	"github.com/ixxxxoooo/minicode/internal/core"
)

func init() {
	Register(Scenario{
		ID: "refactor", Title: "工具卡 · 编辑与改动汇总", Project: "router-demo",
		Label: "周一", Group: "更早", Clock: "周一",
		Build: buildScnRefactor,
	})
}

func buildScnRefactor() []Step {
	s := NewScript("refactor", "工具卡 · 编辑与改动汇总")
	u := s.User("重构 handlers 的错误处理:统一 Result、补上测试,并删掉废弃文件。")
	s.Think("多文件编辑 + 纯新增 / 纯删除 diff,最后应有改动汇总卡。", 4)

	s.Tool(core.NewToolCall("", "Edit", "编辑 src/handlers/error.rs").
		WithInput(map[string]any{"file_path": "src/handlers/error.rs"}).WithLocation("src/handlers/error.rs"),
		core.ToolCompleted, 610, &core.DiffBlock{
			Path: "src/handlers/error.rs", Old: tabCJKOld, New: tabCJKNew, Line: 8,
		})

	me := core.NewToolCall("", "MultiEdit", "编辑 src/server.rs").
		WithInput(map[string]any{"file_path": "src/server.rs"}).WithLocation("src/server.rs")
	s.Tool(me, core.ToolCompleted, 720,
		&core.DiffBlock{Path: "src/server.rs", Old: "use std::io;", New: "use std::io;\nuse crate::handlers::error::AppError;", Line: 1},
		&core.DiffBlock{Path: "src/server.rs", Old: "    handler(req)", New: "    handler(req).map_err(AppError::from)", Line: 52},
	)

	s.Tool(core.NewToolCall("", "Write", "写入 tests/error_handling.rs").
		WithInput(map[string]any{"file_path": "tests/error_handling.rs"}).WithLocation("tests/error_handling.rs"),
		core.ToolCompleted, 480, &core.DiffBlock{Path: "tests/error_handling.rs", Old: "", New: "#[test]\nfn maps_io_error() {\n    assert!(true);\n}\n", Line: 1})

	s.Tool(core.NewToolCall("", "Edit", "删除 src/legacy.rs").
		WithInput(map[string]any{"file_path": "src/legacy.rs"}).WithLocation("src/legacy.rs"),
		core.ToolCompleted, 350, &core.DiffBlock{
			Path: "src/legacy.rs", Old: "pub fn old_entry() {}\n", New: "", Line: 1,
		})

	s.Tool(core.NewToolCall("", "Edit", "重命名 src/temp.rs → src/cache.rs").
		WithInput(map[string]any{"file_path": "src/cache.rs"}).WithLocation("src/cache.rs"),
		core.ToolCompleted, 290, &core.DiffBlock{
			Path: "src/cache.rs", Old: "", New: "//! 自 temp.rs 迁入\n", Line: 1,
		})

	lineDiff := core.NewToolCall("", "Edit", "编辑 src/router.rs · 超长行").
		WithInput(map[string]any{"file_path": "src/router.rs"}).WithLocation("src/router.rs")
	s.Tool(lineDiff, core.ToolCompleted, 410, &core.DiffBlock{
		Path: "src/router.rs", Old: "    let row = \"short\";", New: ultraLongLine, Line: 120,
	})

	s.Say("已在 4 个文件里完成重构:新增测试文件、删除 legacy、统一错误类型。回合末尾应出现改动汇总卡。")
	s.End(u, 64, core.StopEndTurn, DefaultStats(5))
	return s.Steps()
}
