package demo

import "github.com/ixxxxoooo/minicode/internal/vcs"

// Repo 是演示项目 router-demo 的内存仓库:分支、暂存 / 更改、提交图谱都按设计稿造好。
func Repo() *vcs.Mem {
	return buildRouterRepo()
}
