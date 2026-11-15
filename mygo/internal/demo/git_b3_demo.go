package demo

import (
	"github.com/ixxxxoooo/minicode/internal/vcs"
)

// SeedGitB3 为 B3 Git 缺口演示准备数据(储藏 diff、变基冲突、HEAD 比较等)。
func SeedGitB3(m *vcs.Mem) {
	if m == nil {
		return
	}
	m.SetStashHeld(0, []vcs.Change{
		{Path: "src/router.rs", Status: 'M', Add: 4, Del: 2},
		{Path: "tests/router_test.rs", Status: 'M', Add: 1},
	}, map[string][2]string{
		"src/router.rs": {
			"pub fn match_path(path: &str) -> bool {\n    path.eq_ignore_ascii_case(path)\n}\n",
			"pub fn match_path(path: &str) -> bool {\n    let p = path.trim_end_matches('/');\n    p.eq_ignore_ascii_case(p)\n}\n",
		},
	})
	m.SetRebaseConflict("变基时发生冲突:请先解决 src/router.rs,或中止变基")
}
