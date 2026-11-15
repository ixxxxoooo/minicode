package demo

import "github.com/ixxxxoooo/minicode/internal/vcs"

// Repo 是演示项目 router-demo 的内存仓库:分支、暂存 / 更改、提交图谱都按设计稿造好。
func Repo() *vcs.Mem {
	m := vcs.NewMem()
	changes := []vcs.Change{
		{Path: "src/router.rs", Status: 'M', Staged: true, Add: 4, Del: 1},
		{Path: "tests/router_test.rs", Status: 'M', Add: 12},
		{Path: "tests/case_test.rs", Status: 'A', Add: 2},
	}
	log := []vcs.Commit{
		{Hash: "a3f9c21", Subject: "fix(router): 路径匹配忽略大小写与末尾斜杠", Author: "你", When: "刚刚", Parents: []string{"b7d02e4"}, Refs: []string{"HEAD", "main"}},
		{Hash: "b7d02e4", Subject: "test: 补充 router 的大小写用例", Author: "你", When: "2 小时", Parents: []string{"e19a6c0", "c44f8d1"}},
		{Hash: "c44f8d1", Subject: "feat(auth): 支持 Bearer 令牌", Author: "Lin", When: "昨天", Parents: []string{"d8830b7"}, Refs: []string{"feat/auth"}},
		{Hash: "d8830b7", Subject: "refactor: 抽出 handlers 模块", Author: "Lin", When: "昨天", Parents: []string{"e19a6c0"}},
		{Hash: "e19a6c0", Subject: "chore: 升级 axum 到 0.8", Author: "你", When: "周一", Parents: []string{"f5521ab"}, Refs: []string{"origin/main"}},
		{Hash: "f5521ab", Subject: "docs: 补充路由匹配说明", Author: "Wen", When: "11月8日"},
	}
	files := map[string][]vcs.Change{
		"a3f9c21": {{Path: "src/router.rs", Status: 'M', Add: 4, Del: 1}, {Path: "tests/router_test.rs", Status: 'A', Add: 12}},
		"b7d02e4": {{Path: "tests/router_test.rs", Status: 'M', Add: 9, Del: 2}},
		"c44f8d1": {{Path: "src/handlers/users.rs", Status: 'M', Add: 18, Del: 3}, {Path: "src/server.rs", Status: 'M', Add: 5}},
		"d8830b7": {{Path: "src/handlers/mod.rs", Status: 'A', Add: 1}, {Path: "src/handlers/users.rs", Status: 'A', Add: 3}},
		"e19a6c0": {{Path: "Cargo.toml", Status: 'M', Add: 1, Del: 1}},
		"f5521ab": {{Path: "README.md", Status: 'M', Add: 14}},
	}
	head := map[string]string{}
	for p := range GitStatus() {
		if s, ok := HeadOf(p); ok {
			head[p] = s
		}
	}
	m.SetState(changes, log, files, head, 1)
	return m
}
