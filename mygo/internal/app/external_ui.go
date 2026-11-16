package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/workspace"
)

func (a *App) isExternalTab(tb *ideTab) bool {
	return tb != nil && !tb.diff && tb.virt == "" && workspace.IsExternalPath(tb.path)
}

// externalChip 标签 / 面包屑旁的「工作区外」标记。
func (a *App) externalChip(c *ui.Context) {
	kit.Chip(c, "工作区外", kit.ChipOpt{
		Line: true,
		Tip:  "不在当前项目里,没有 Git 信息",
	})
}
