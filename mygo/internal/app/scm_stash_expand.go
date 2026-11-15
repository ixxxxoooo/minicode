package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
	"github.com/ixxxxoooo/minicode/internal/vcs"
)

func (a *App) scmStashToggle(i int) {
	if a.sc.stashOpen == nil {
		a.sc.stashOpen = map[int]bool{}
	}
	if a.sc.stashOpen[i] {
		delete(a.sc.stashOpen, i)
		return
	}
	a.sc.stashOpen[i] = true
	repo := a.repo
	if repo == nil {
		return
	}
	a.bg("scm.stashfiles", func() func() {
		fs, err := repo.StashFiles(i)
		return func() {
			if repo != a.repo {
				return
			}
			if err != nil {
				a.toast = "读取储藏文件失败:" + err.Error()
				delete(a.sc.stashOpen, i)
				return
			}
			if a.sc.stashFiles == nil {
				a.sc.stashFiles = map[int][]vcs.Change{}
			}
			a.sc.stashFiles[i] = fs
		}
	})
}

func (a *App) scmOpenStashDiff(i int, ch vcs.Change) {
	repo := a.repo
	if repo == nil {
		return
	}
	tp := a.treePath(ch.Path)
	id := "diff:stash-" + itoa(i) + ":" + tp
	a.bg("scm.stashdiff", func() func() {
		old, nw, ok := repo.StashFileContent(i, ch.Path)
		return func() {
			if !ok {
				a.toast = "无法读取储藏差异"
				return
			}
			a.ide.addDiffTab(id, tp, "stash@{"+itoa(i)+"} · "+ch.Path, old, nw, true)
		}
	})
}

func (a *App) scmStashFileRows(c *ui.Context, i int) {
	t := a.tk
	fs := a.sc.stashFiles[i]
	for _, ch := range fs {
		ch := ch
		fr := ui.ButtonBase(c).Key("stashf-" + itoa(i) + ch.Path).Height(28).PaddingX(22).Gap(6).
			AlignItems(ui.Center).Radius(t.RSm()).Cursor(ui.CursorPointer).TextColor(t.Tx2)
		if fr.Hovered() {
			fr.Background(t.Fill1)
		}
		fr.OnClick(func() { a.scmOpenStashDiff(i, ch) })
		fr.Children(func() {
			kit.Ic(c, fileIcon(ch.Path), kit.S12)
			ui.Text(c, ch.Path).SingleLine().Grow(1).FontSize(theme.FsCap)
			ui.Text(c, statusLetter(ch.Status)).Font("monospace").FontSize(theme.FsMicro).TextColor(t.Tx4)
		})
	}
}
