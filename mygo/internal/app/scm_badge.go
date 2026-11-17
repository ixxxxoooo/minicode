package app

import (
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/vcs"
)

// scmDemoBadge 是演示模式(没有打开真实项目)下「源代码管理」导航上显示的改动数,沿用设计稿示例。
const scmDemoBadge = 3

// scmBadge 返回导航上「源代码管理」入口要显示的改动提示。
// 用途:让用户不打开源代码管理面板,也能一眼看到工作区有未提交的改动。
// 返回值:n 为有改动的文件数(同一文件的暂存 + 工作区只算一个,多仓库时累加所有仓库);
// conflict 表示存在合并冲突(提示改用错误色)。n == 0 表示没有改动,不显示任何标识。
// 副作用:真实项目下每帧顺带触发一次 scmTick,保证切到别的分段时改动数也定时刷新。
func (a *App) scmBadge() (n int, conflict bool) {
	if a.proj == "" && !a.multiRoot() {
		return scmDemoBadge, false
	}
	a.scmTick()
	if a.multiRoot() {
		for i := range a.repos {
			_, _, _, c := a.repoSummary(i)
			n += c
		}
		conflict = hasConflict(a.sc.changes)
		return
	}
	seen := map[string]bool{}
	for _, c := range a.sc.changes {
		seen[c.Path] = true
	}
	return len(seen), hasConflict(a.sc.changes)
}

// hasConflict 判断改动列表里是否有合并冲突的文件。
func hasConflict(cs []vcs.Change) bool {
	for _, c := range cs {
		if c.Status == vcs.StatusConflict {
			return true
		}
	}
	return false
}

// scmTip 是「源代码管理」导航的悬停说明:有改动时带上数量。
func scmTip(n int, conflict bool) string {
	switch {
	case conflict:
		return "源代码管理 · " + itoa(n) + " 项改动(有合并冲突)"
	case n > 0:
		return "源代码管理 · " + itoa(n) + " 项改动"
	}
	return "源代码管理"
}

// scmDotKind 是导航圆点的颜色:冲突用错误色,普通改动用强调色。
func scmDotKind(conflict bool) kit.TagKind {
	if conflict {
		return kit.KErr
	}
	return kit.KAc
}
