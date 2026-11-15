package demo

import (
	"fmt"

	"github.com/ixxxxoooo/minicode/internal/vcs"
)

// routerPreserveLog 设计稿截图依赖的 6 条提交(顺序:新 → 旧),勿改哈希与摘要。
func routerPreserveLog() []vcs.Commit {
	return []vcs.Commit{
		{Hash: "a3f9c21", Subject: "fix(router): 路径匹配忽略大小写与末尾斜杠", Author: "你", When: "刚刚", Parents: []string{"b7d02e4"}, Refs: []string{"HEAD", "main", "origin/main"}},
		{Hash: "b7d02e4", Subject: "test: 补充 router 的大小写用例", Author: "你", When: "2 小时", Parents: []string{"e19a6c0", "c44f8d1"}},
		{Hash: "c44f8d1", Subject: "feat(auth): 支持 Bearer 令牌", Author: "Lin", When: "昨天", Parents: []string{"d8830b7"}, Refs: []string{"feat/auth"}},
		{Hash: "d8830b7", Subject: "refactor: 抽出 handlers 模块", Author: "Lin", When: "昨天", Parents: []string{"e19a6c0"}},
		{Hash: "e19a6c0", Subject: "chore: 升级 axum 到 0.8", Author: "你", When: "周一", Parents: []string{"f5521ab"}, Refs: []string{"origin/main"}},
		{Hash: "f5521ab", Subject: "docs: 补充路由匹配说明", Author: "Wen", When: "11月8日"},
	}
}

func routerPreserveCommitFiles() map[string][]vcs.Change {
	return map[string][]vcs.Change{
		"a3f9c21": {{Path: "src/router.rs", Status: 'M', Add: 4, Del: 1}, {Path: "tests/router_test.rs", Status: 'A', Add: 12}},
		"b7d02e4": {{Path: "tests/router_test.rs", Status: 'M', Add: 9, Del: 2}},
		"c44f8d1": {{Path: "src/handlers/users.rs", Status: 'M', Add: 18, Del: 3}, {Path: "src/server.rs", Status: 'M', Add: 5}},
		"d8830b7": {{Path: "src/handlers/mod.rs", Status: 'A', Add: 1}, {Path: "src/handlers/users.rs", Status: 'A', Add: 3}},
		"e19a6c0": {{Path: "Cargo.toml", Status: 'M', Add: 1, Del: 1}},
		"f5521ab": {{Path: "README.md", Status: 'M', Add: 14}},
	}
}

// gitExtendLog 在 f5521ab 之后追加更老的历史(共 220+ 条可见于 Mem.Log)。
func gitExtendLog(base []vcs.Commit, files map[string][]vcs.Change) []vcs.Commit {
	authors := []string{"你", "Lin", "Wen", "王明", "田中太郎", "김민수", "Zoe Chen"}
	whens := []string{"刚刚", "5 分钟", "1 小时", "3 小时", "昨天", "前天", "周一", "周二", "10月12日", "2024年8月3日", "去年"}
	parent := base[len(base)-1].Hash
	var older []vcs.Commit
	for i := 0; i < 214; i++ {
		h := fmt.Sprintf("%06x", 0x100000+i*17)
		subj := fmt.Sprintf("chore(history): 自动化提交 #%d", 214-i)
		if i%40 == 0 {
			subj = fmt.Sprintf("feat: 大功能批次 %d\n\n第二行说明\n- 子项 A\n- 子项 B", i/40)
		}
		if i == 50 {
			subj = "Merge branch 'feat/ui-redesign' (章鱼合并演示)"
		}
		cm := vcs.Commit{
			Hash: h, Subject: subj,
			Author: authors[i%len(authors)],
			When:   whens[i%len(whens)],
		}
		if i == 0 {
			cm.Parents = []string{parent}
		} else {
			cm.Parents = []string{older[len(older)-1].Hash}
		}
		if i == 30 {
			cm.Parents = []string{older[len(older)-1].Hash, fmt.Sprintf("%06x", 0x100000+28*17)}
			cm.Subject = "Merge branch 'feat/spike' and 'hotfix/cache'"
		}
		if i == 50 {
			cm.Parents = []string{older[len(older)-1].Hash, fmt.Sprintf("%06x", 0x100000+48*17), fmt.Sprintf("%06x", 0x100000+49*17)}
		}
		if i == 5 {
			cm.Refs = []string{"v0.3.0", "origin/release/0.3"}
		}
		if i == 12 {
			cm.Refs = []string{"feat/very/long/nested/branch/name-for-truncation"}
		}
		older = append(older, cm)
		if i == 80 {
			fc := make([]vcs.Change, 80)
			for j := 0; j < 80; j++ {
				fc[j] = vcs.Change{Path: fmt.Sprintf("demo/history/batch_%d.txt", j), Status: 'M', Add: 1}
			}
			files[h] = fc
		} else if i%17 == 0 {
			files[h] = []vcs.Change{{Path: "src/router.rs", Status: 'M', Add: 1}}
		}
	}
	out := append(append([]vcs.Commit(nil), base...), older...)
	return out
}

func gitRouterBranches() []vcs.Branch {
	return []vcs.Branch{
		{Name: "main", When: "当前"},
		{Name: "feat/auth", When: "2 小时前"},
		{Name: "feat/ui-redesign", When: "刚刚"},
		{Name: "feat/very/long/nested/branch/name-for-truncation", When: "昨天"},
		{Name: "feat/cache/warmup/probe", When: "3 天前"},
		{Name: "origin/dev", Remote: true, When: "昨天"},
		{Name: "origin/main", Remote: true, When: "刚刚"},
		{Name: "origin/feat/auth", Remote: true, When: "2 小时前"},
	}
}
