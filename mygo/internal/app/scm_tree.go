package app

import (
	"path"
	"sort"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
	"github.com/ixxxxoooo/minicode/internal/vcs"
)

const pScmView = "scm.viewMode"

func (a *App) scmViewMode() string {
	m := a.prefs.Str(pScmView, "list")
	if m != "tree" {
		return "list"
	}
	return "tree"
}

func (a *App) toggleScmViewMode() {
	on := a.scmViewMode() == "tree"
	a.prefs.Set(pScmView, map[bool]string{true: "list", false: "tree"}[on])
	log.Info("切换 SCM 视图", "tree", !on)
}

func (a *App) scmGroupFiles(c *ui.Context, id string, cs []vcs.Change) {
	if a.scmViewMode() == "list" || id == "merge" {
		for _, ch := range cs {
			a.scmFile(c, id, ch)
		}
		return
	}
	s := &a.sc
	if s.dirFold == nil {
		s.dirFold = map[string]bool{}
	}
	byDir := map[string][]vcs.Change{}
	for _, ch := range cs {
		d := path.Dir(ch.Path)
		if d == "." {
			d = ""
		}
		byDir[d] = append(byDir[d], ch)
	}
	dirs := make([]string, 0, len(byDir))
	for d := range byDir {
		dirs = append(dirs, d)
	}
	sort.Slice(dirs, func(i, j int) bool {
		if dirs[i] == "" {
			return true
		}
		if dirs[j] == "" {
			return false
		}
		return dirs[i] < dirs[j]
	})
	t := a.tk
	for _, d := range dirs {
		key := id + ":" + d
		open := !s.dirFold[key]
		h := ui.ButtonBase(c).Key("scm-dir-" + key).Height(theme.Px(22)).PaddingX(4).PaddingStart(10).Gap(4).AlignItems(ui.Center).Radius(t.RSm()).Cursor(ui.CursorPointer).TextColor(t.Tx3)
		if h.Hovered() {
			h.Background(t.Fill1)
		}
		h.OnClick(func() { s.dirFold[key] = open })
		h.Children(func() {
			kit.Ic(c, map[bool]string{true: "chevD", false: "chevR"}[open], kit.S12)
			label := d
			if label == "" {
				label = "(根)"
			}
			ui.Text(c, label).FontSize(theme.FsCap).FontWeight(500).SingleLine().Grow(1)
			ui.Text(c, itoa(len(byDir[d]))).FontSize(theme.FsMicro).TextColor(t.Tx4)
		})
		if !open {
			continue
		}
		for _, ch := range byDir[d] {
			row := ui.ButtonBase(c).Key("scm-f-" + id + "-" + ch.Path).Height(theme.Px(22)).PaddingX(4).PaddingStart(22).Gap(5).AlignItems(ui.Center).Radius(t.RSm()).Cursor(ui.CursorPointer)
			row.Children(func() { a.scmFileInner(c, id, ch, row) })
		}
	}
}

// scmFileInner 画文件行内容(scmFile 与树形缩进复用)。
func (a *App) scmFileInner(c *ui.Context, grp string, ch vcs.Change, r ui.Element) {
	t := a.tk
	r.ContextMenu(func(m *ui.Menu) { a.scmFileMenu(m, grp, ch) })
	hov := r.Hovered()
	if grp != "merge" && a.ide.active == diffID(grpDiffKind(grp), a.treePath(ch.Path)) { // 正在看这个文件的差异:选中外观
		kit.SelStyle(r)
	} else if hov {
		r.Background(t.Fill1)
	}
	if r.Clicked() {
		a.scmOpenFile(grp, ch, false)
	}
	if r.DoubleClicked() {
		a.scmOpenFile(grp, ch, true)
	}
	col := t.Tx2
	if ch.Status == 'D' {
		col = t.Tx4
	}
	r.TextColor(col).Children(func() {
		a.fileIc(c, path.Base(ch.Path))
		nm := ui.Text(c, path.Base(ch.Path)).SingleLine().Shrink(0)
		if ch.Status == 'D' {
			nm.Strikethrough()
		}
		if f := path.Base(path.Dir(ch.Path)); f != "." && f != "" {
			ui.Text(c, f).FontSize(theme.FsMicro).TextColor(t.Tx4).SingleLine().Shrink(1).MinWidth(0)
		}
		ui.Spacer(c).Grow(1)
		if hov {
			kit.Ib(c, "file", kit.IbOpt{Small: true, Tip: "打开文件"}).OnClick(func() { a.ide.openFile(a.treePath(ch.Path), true) })
			a.scmFileHoverBtns(c, grp, ch)
			return
		}
		a.scmStat(c, ch)
	})
}

func (a *App) scmFileHoverBtns(c *ui.Context, grp string, ch vcs.Change) {
	if grp == "changes" {
		kit.Ib(c, "undo", kit.IbOpt{Small: true, Tip: "放弃更改"}).OnClick(func() { a.scmDiscardAsk([]vcs.Change{ch}) })
		kit.Ib(c, "plus", kit.IbOpt{Small: true, Tip: "暂存"}).OnClick(func() { a.scmStage(ch.Path) })
	} else {
		kit.Ib(c, "minus", kit.IbOpt{Small: true, Tip: "取消暂存"}).OnClick(func() { a.scmUnstage(ch.Path) })
	}
}
