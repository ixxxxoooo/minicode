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
			Tip:  "项目外文件,默认仅查看;点击解除只读后可编辑并保存",
		}).OnClick(func() { a.unlockReadOnly(tb) })
		return
	}
	kit.Chip(c, "工作区外", kit.ChipOpt{
		Line: true,
		Tip:  "不在当前项目里,没有 Git 信息",
	})
}

// unlockReadOnly 把只读打开的项目外文件(语言服务器跳转开的标准库 / 依赖等)切成可编辑。
// 改动会直接写回磁盘上的原文件,所以先确认一次;确认后标签与普通文件一样可编辑、可保存、可替换。
func (a *App) unlockReadOnly(tb *ideTab) {
	if tb == nil || !tb.diskRO || tb.st == nil {
		return
	}
	name := baseName(tb.path)
	a.confirmDlg("解除只读,开始编辑?", name+"\n这是项目外的文件(标准库 / 依赖等),保存会直接写入磁盘上的原文件。", "解除只读", false, func() {
		tb.diskRO = false
		tb.st.ReadOnly = false
		a.toast = "已解除只读:" + name + "(⌘S 保存)"
		extOpenLog.Info("解除只读", "path", tb.path)
	})
}
