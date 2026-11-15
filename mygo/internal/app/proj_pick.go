package app

import (
	"path/filepath"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/projects"
)

// sideBProject 方案 B 当前浏览的项目(空 = 跟当前会话所在项目)。
func (a *App) sideBProject() core.ProjectInfo {
	if a.sideBF != "" {
		for _, p := range a.projects {
			if p.ID == a.sideBF {
				return p
			}
		}
	}
	return a.curProject()
}

// projPickPopup 方案 B 的项目下拉(含工作区)。
func (a *App) projPickPopup(c *ui.Context, anchor ui.Element) {
	a.popOpen("projpick", func(open *bool) {
		ui.PopoverBase(c, anchor, open, func(panel ui.Element) {
			a.popPanel(panel, 280).AttachTo(anchor, ui.AnchorTopLeft, ui.AnchorBottomLeft).Margin(4, 0, 0, 0)
			for _, w := range a.registry().Workspaces() {
				kit.Cap(c, "工作区 · "+w.Name)
				if a.mi(c, "ws-"+w.ID, "layers", w.Name, map[bool]string{true: "使用中", false: ""}[a.activeWsID() == w.ID], a.activeWsID() == w.ID).Clicked() {
					a.popup = ""
					a.ActivateWorkspace(w.ID)
					a.sideBF = ""
				}
				for _, f := range w.Folders {
					a.projPickRow(c, f.Path, filepath.Base(f.Path), "成员")
				}
			}
			kit.Cap(c, "项目")
			for _, p := range a.sideProjects() {
				if a.memberOfWorkspace(p.Path) {
					continue
				}
				a.projPickRow(c, p.Path, p.Name, "")
			}
		})
	})
}

func (a *App) projPickRow(c *ui.Context, path, name, tag string) {
	on := a.sideBProject().Path == path || a.proj == path
	key := "pp-" + projects.Clean(path)
	if a.mi(c, key, "folder", name, tag, on).Clicked() {
		a.popup = ""
		a.pickSideProject(path)
	}
}

func (a *App) pickSideProject(path string) {
	if p, ok := a.projectByPath(path); ok {
		a.sideBF = p.ID
	} else {
		a.sideBF = "path:" + path
	}
	if a.proj != "" {
		a.SetProject(path)
	}
	live, _ := sessionsSplit(a.sideBProject())
	for _, s := range live {
		if !s.Archived {
			a.OpenSession(s.ID)
			return
		}
	}
	a.toast = "这个项目还没有可打开的会话"
}
