package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/workspace"
)

func (a *App) isExternalTab(tb *ideTab) bool {
	return tb != nil && !tb.diff && tb.virt == "" && workspace.IsExternalPath(tb.path)
}

// externalChip 标签 / 面包屑旁:只读跳转开的依赖显示锁,用户显式打开的工作区外文件显示「工作区外」。
func (a *App) externalChip(c *ui.Context, tb *ideTab) {
	if tb != nil && tb.diskRO {
		kit.Chip(c, "只读", kit.ChipOpt{
			Icon: "lock",
			Line: true,
			Tip:  "项目外文件,仅查看;不支持保存、格式化与追溯",
		})
		return
	}
	kit.Chip(c, "工作区外", kit.ChipOpt{
		Line: true,
		Tip:  "不在当前项目里,没有 Git 信息",
	})
}
