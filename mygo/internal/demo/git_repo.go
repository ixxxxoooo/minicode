package demo

import (
	"fmt"

	"github.com/ixxxxoooo/minicode/internal/vcs"
)

// buildRouterRepo 组装 router-demo 的完整演示仓库。
func buildRouterRepo() *vcs.Mem {
	changes := append(routerPreserveChanges(), gitExtraChanges()...)
	files := routerPreserveCommitFiles()
	log := gitExtendLog(routerPreserveLog(), files)
	head := gitRouterHeadContent()
	for p := range GitStatusLite() {
		if _, ok := head[p]; !ok {
			if s, ok2 := HeadOf(p); ok2 {
				head[p] = s
			}
		}
	}
	m := vcs.NewMem()
	m.SetState(changes, log, files, head, 3)
	m.SetAheadBehind(3, 2)
	m.SetBranches(gitRouterBranches())
	m.SetStash(gitRouterStashes())
	m.SetIndex(gitRouterIndexContent())
	m.SetBlameData(gitRouterBlame())
	for p, lg := range gitRouterFileLog() {
		m.SetFileLogFor(p, lg)
	}
	return m
}

// GitStatusLite 树标记用的主要路径(全量状态以 Repo().Status() 为准)。
func GitStatusLite() map[string]string {
	m := map[string]string{
		"src/router.rs": "M", "tests/router_test.rs": "M", "tests/case_test.rs": "A",
		"src/server.rs": "M", "demo/conflicts/two.rs": "C", "demo/massive_diff.rs": "M",
	}
	for i := 1; i <= 300; i++ {
		p := fmt.Sprintf("demo/bulk/file_%03d.txt", i)
		st := "M"
		if i%40 == 0 {
			st = "A"
		}
		if i%73 == 0 {
			st = "D"
		}
		m[p] = st
	}
	return m
}
