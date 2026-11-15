package app

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
	"github.com/ixxxxoooo/minicode/internal/vcs"
)

const stashFilePreview = 8

var stashLog = diag.L("scm.stash")

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
		a.toast = "当前不是 Git 仓库"
		delete(a.sc.stashOpen, i)
		return
	}
	if a.sc.stashBusy == nil {
		a.sc.stashBusy = map[int]bool{}
	}
	a.sc.stashBusy[i] = true
	a.bg("scm.stashfiles", func() func() {
		end := diag.SpanSlow("scm", "储藏文件列表", time.Second)
		fs, err := repo.StashFiles(i)
		end()
		return func() {
			delete(a.sc.stashBusy, i)
			if repo != a.repo {
				return
			}
			if err != nil {
				stashLog.Warn("读取储藏文件失败", "index", i, "err", err)
				a.toast = "读取储藏文件失败:" + err.Error()
				delete(a.sc.stashOpen, i)
				return
			}
			if a.sc.stashFiles == nil {
				a.sc.stashFiles = map[int][]vcs.Change{}
			}
			a.sc.stashFiles[i] = fs
			stashLog.Info("展开储藏", "index", i, "files", len(fs))
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
		end := diag.SpanSlow("scm", "储藏差异", time.Second)
		old, nw, ok := repo.StashFileContent(i, ch.Path)
		end()
		return func() {
			if !ok {
				a.toast = "无法读取储藏差异"
				stashLog.Warn("储藏差异读取失败", "index", i, "path", ch.Path)
				return
			}
			a.ide.addDiffTab(id, tp, "stash@{"+itoa(i)+"} · "+ch.Path, old, nw, true)
			stashLog.Info("打开储藏差异", "index", i, "path", ch.Path)
		}
	})
}

func (a *App) scmStashFileMenu(m *ui.Menu, i int, ch vcs.Change) {
	if menuItem(m, "打开差异").Chosen() {
		a.scmOpenStashDiff(i, ch)
	}
	if menuItem(m, "复制路径").Chosen() {
		a.copyOut(ch.Path, "路径")
	}
}

func (a *App) scmStashFileRows(c *ui.Context, i int) {
	t := a.tk
	s := &a.sc
	if s.stashBusy != nil && s.stashBusy[i] {
		ui.Text(c, "读取中…").FontSize(theme.FsCap).TextColor(t.Tx4).Padding(2, 24, 4, 8)
		return
	}
	fs, ok := s.stashFiles[i]
	if !ok {
		return
	}
	if len(fs) == 0 {
		ui.Text(c, "这条储藏没有文件改动").FontSize(theme.FsCap).TextColor(t.Tx4).Padding(2, 24, 4, 8)
		return
	}
	showAll := s.stashFilesAll != nil && s.stashFilesAll[i]
	limit := len(fs)
	if !showAll && len(fs) > stashFilePreview {
		limit = stashFilePreview
	}
	for _, ch := range fs[:limit] {
		ch := ch
		fr := ui.ButtonBase(c).Key("stashf-" + itoa(i) + ch.Path).Height(28).PaddingX(22).Gap(6).
			AlignItems(ui.Center).Radius(t.RSm()).Cursor(ui.CursorPointer).TextColor(t.Tx2)
		if fr.Hovered() {
			fr.Background(t.Fill1)
		}
		fr.ContextMenu(func(m *ui.Menu) { a.scmStashFileMenu(m, i, ch) })
		fr.OnClick(func() { a.scmOpenStashDiff(i, ch) })
		fr.Children(func() {
			kit.Ic(c, fileIcon(ch.Path), kit.S12)
			ui.Text(c, ch.Path).SingleLine().Grow(1).FontSize(theme.FsCap)
			ui.Text(c, statusLetter(ch.Status)).Font("monospace").FontSize(theme.FsMicro).TextColor(t.Tx4)
		})
	}
	if !showAll && len(fs) > stashFilePreview {
		more := len(fs) - stashFilePreview
		r := ui.ButtonBase(c).Key("stash-more-" + itoa(i)).Height(26).PaddingX(24).Gap(6).
			AlignItems(ui.Center).Radius(t.RSm()).Cursor(ui.CursorPointer).TextColor(t.Tx3)
		if r.Hovered() {
			r.Background(t.Fill1)
		}
		r.OnClick(func() {
			if s.stashFilesAll == nil {
				s.stashFilesAll = map[int]bool{}
			}
			s.stashFilesAll[i] = true
		})
		r.Children(func() {
			kit.Ic(c, "chevD", kit.S12)
			ui.Text(c, "还有 "+itoa(more)+" 个文件…").FontSize(theme.FsCap)
		})
	}
}
