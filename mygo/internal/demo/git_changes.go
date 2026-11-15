package demo

import (
	"fmt"

	"github.com/ixxxxoooo/minicode/internal/vcs"
)

// routerPreserveChanges 设计稿依赖的三项改动,勿删。
func routerPreserveChanges() []vcs.Change {
	return []vcs.Change{
		{Path: "src/router.rs", Status: 'M', Staged: true, Add: 4, Del: 1},
		{Path: "tests/router_test.rs", Status: 'M', Add: 12},
		{Path: "tests/case_test.rs", Status: 'A', Add: 2},
	}
}

func gitExtraChanges() []vcs.Change {
	cs := []vcs.Change{
		// 同一文件:暂存 + 工作区各一份
		{Path: "src/server.rs", Status: 'M', Staged: true, Add: 2, Del: 1},
		{Path: "src/server.rs", Status: 'M', Add: 5, Del: 0},
		{Path: "src/handlers/users.rs", Status: 'M', Staged: true, Add: 1, Del: 1},
		{Path: "src/handlers/users.rs", Status: 'M', Add: 3, Del: 2},
		{Path: "src/legacy.rs", Status: 'D', Staged: true, Del: 40},
		{Path: "tmp/scratch.txt", Status: 'D', Del: 2},
		{Path: "src/config_new.rs", Status: 'R', Staged: true, Add: 1, Del: 1}, // 自 src/config_old.rs
		{Path: "src/config_old.rs", Status: 'R', Staged: true, Del: 1},
		{Path: "src/config_copy.rs", Status: 'A', Staged: true, Add: 1}, // 复制(勿用字母 C,与冲突相同)
		{Path: "demo/untracked_notes.md", Status: 'U', Add: 3},
		{Path: "demo/license.bin", Status: 'U', Add: 1},
		{Path: "src/util.rs", Status: 'A', Staged: true, Add: 8},
		{Path: "Cargo.lock", Status: 'M', Staged: true, Add: 12, Del: 0},
		{Path: "demo/bin/app.ico", Status: 'M', Add: 0, Del: 0},
		{Path: "demo/massive_diff.rs", Status: 'M', Add: 2000, Del: 5000},
		{Path: "demo/eol_crlf.txt", Status: 'M', Add: 1, Del: 1},
		{Path: "demo/whitespace_only.rs", Status: 'M', Add: 0, Del: 0},
		{Path: "demo/word_highlight.rs", Status: 'M', Add: 2, Del: 2},
		{Path: "demo/wide_line.rs", Status: 'M', Add: 1, Del: 1},
		// 合并冲突(双方修改 / 双方新增 / 一删一改)
		{Path: "demo/conflicts/two.rs", Status: vcs.StatusConflict, Add: 4, Del: 2},
		{Path: "demo/conflicts/five.toml", Status: vcs.StatusConflict, Add: 10, Del: 6},
		{Path: "demo/conflicts/broken.md", Status: vcs.StatusConflict, Add: 1, Del: 1},
		{Path: "demo/conflicts/binary.dat", Status: vcs.StatusConflict},
		{Path: "demo/conflicts/both_added.json", Status: vcs.StatusConflict, Add: 2},
		{Path: "demo/conflicts/delete_modify.txt", Status: vcs.StatusConflict, Del: 3},
	}
	for i := 1; i <= 300; i++ {
		st := byte('M')
		staged := i%5 == 0
		if i%40 == 0 {
			st = 'A'
		}
		if i%73 == 0 {
			st = 'D'
		}
		cs = append(cs, vcs.Change{
			Path:   fmt.Sprintf("demo/bulk/file_%03d.txt", i),
			Status: st,
			Staged: staged,
			Add:    1 + i%7,
			Del:    i % 4,
		})
	}
	return cs
}
