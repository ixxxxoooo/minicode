package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

// undoTurnChanges 撤销本回合改动卡里列出的文件(逐文件还原第一个未撤销块)。
func (a *App) undoTurnChanges() {
	a.rv.init()
	files := a.changedFiles()
	if len(files) == 0 {
		a.toast = "没有可撤销的改动"
		return
	}
	n := 0
	for _, f := range files {
		for i := range f.blocks {
			k := hunkKey(f.path, i)
			if a.rv.hunk != nil && a.rv.hunk[k] == "rev" {
				continue
			}
			a.revertHunk(f, i)
			n++
			break
		}
	}
	if n == 0 {
		a.toast = "改动已全部撤销"
		return
	}
	a.toast = "已撤销 " + itoa(n) + " 个文件的改动"
	log.Info("改动卡撤销", "files", n)
}

// changesCardUndoBtn 改动卡头部的撤销按钮。
func (a *App) changesCardUndoBtn(c *ui.Context) {
	kit.Btn(c, "撤销", kit.BtnOpt{Variant: kit.BtnGhost, Small: true, Icon: "undo"}).OnClick(func() {
		a.undoTurnChanges()
	})
}
