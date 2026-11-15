package demo

// DemoProblem 供问题面板与编辑器波浪线共用(演示模式)。
type DemoProblem struct {
	Path, Level, Msg, Source, Word string // Path 为项目相对路径
	Line                           int    // 1 起
}

// Problems 返回演示诊断:错误 / 警告 / 信息 / 提示、同文件多行、同一行多条。
func Problems() []DemoProblem {
	return []DemoProblem{
		{"src/router.rs", "warn", "`fallback` 字段从未被读取", "rust-analyzer", "fallback", 6},
		{"src/router.rs", "warn", "未使用的变量:`h`", "rust-analyzer", "h", 11},
		{"src/router.rs", "info", "可改成 `&str` 避免分配", "rust-analyzer", "normalize", 29},
		{"src/router.rs", "hint", "添加 `#[allow(dead_code)]` 或删除", "rust-analyzer", "Router", 4},
		{"tests/router_test.rs", "err", "找不到函数 `normalize`,需要 `use crate::router::normalize`", "rust-analyzer", "normalize", 8},
		{"samples/langs/go_demo.go", "err", "cannot use \"type error for demo problems\" (untyped string constant) as int value in assignment", "gopls", "type", 21},
		{"samples/langs/go_demo.go", "warn", "Normalize 已声明但未使用(演示)", "gopls", "Normalize", 6},
		{"samples/lints/multi_diag.rs", "err", "expected `)`, found `{`", "rust-analyzer", "main", 1},
		{"samples/lints/multi_diag.rs", "warn", "unused variable: `fallback`", "rust-analyzer", "fallback", 2},
		{"samples/lints/multi_diag.rs", "warn", "unused variable: `h`", "rust-analyzer", "h", 3},
		{"samples/lints/multi_diag.rs", "info", "此处的 `normalize` 与外层 crate 中的函数同名", "rust-analyzer", "normalize", 4},
		{"edge/syntax_errors.rs", "err", "语法错误集合(演示)", "rust-analyzer", "broken", 1},
	}
}
