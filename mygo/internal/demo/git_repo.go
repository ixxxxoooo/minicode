package demo

import (
	"fmt"
	"os"

	"github.com/ixxxxoooo/minicode/internal/vcs"
)

var stress = os.Getenv("MINI_DEMO_STRESS") == "1"

// SetStress 打开 / 关闭「极端数据」演示:300 文件 bulk、冲突、重命名、5 条储藏、220+ 提交等。
// 默认关闭,主演示仓库只有设计稿里的三项改动与 6 条提交,截图才能与设计稿逐项对照;
// 要看压力场景:MINI_DEMO_STRESS=1 启动(或 cmd/shot -stress),测试里调 SetStress(true)。
func SetStress(on bool) { stress, sandboxExtras = on, on }

// sandboxExtras 控制沙盒文件树里的压力目录(samples / edge / bulk / node_modules 等多语言与边界文件),
// 默认跟随极端数据开关;编辑器类测试在 TestMain 里单独打开。
var sandboxExtras = stress

// SetSandboxExtras 单独开关沙盒的压力目录(不影响 Git 极端数据)。
func SetSandboxExtras(on bool) { sandboxExtras = on }

// Stress 报告极端数据是否开启。
func Stress() bool { return stress }

// buildRouterRepo 组装 router-demo 的完整演示仓库。
func buildRouterRepo() *vcs.Mem {
	changes := routerPreserveChanges()
	files := routerPreserveCommitFiles()
	log := routerPreserveLog()
	if stress {
		changes = append(changes, gitExtraChanges()...)
		log = gitExtendLog(log, files)
	}
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
	stashes := gitRouterStashes()
	if !stress {
		stashes = stashes[:1]
	}
	m.SetStash(stashes)
	m.SetIndex(gitRouterIndexContent())
	m.SetBlameData(gitRouterBlame())
	for p, lg := range gitRouterFileLog() {
		m.SetFileLogFor(p, lg)
	}
	return m
}

// GitStatusLite 树标记用的主要路径(全量状态以 Repo().Status() 为准)。
func GitStatusLite() map[string]string {
	m := map[string]string{"src/router.rs": "M", "tests/router_test.rs": "M", "tests/case_test.rs": "A"}
	if !stress {
		return m
	}
	m["src/server.rs"], m["demo/conflicts/two.rs"], m["demo/massive_diff.rs"] = "M", "C", "M"
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
