package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/projects"
)

// sideBProject 方案 B 当前浏览的项目:当前会话所在的项目优先,其次是刚选中的(还没有会话),最后是当前项目目录。
func (a *App) sideBProject() core.ProjectInfo {
	if _, p, ok := a.sessionByID(a.curSess); ok && a.curSess != "" {
		return p
	}
	if a.sideBF != "" {
		for _, p := range a.sideProjects() {
			if p.ID == a.sideBF {
				return p
			}
		}
	}
	if a.proj != "" {
		if p, ok := a.projectByPath(a.proj); ok {
			return p
		}
	}
	return a.curProject()
}

// sideBCtx 方案 B 项目条显示的当前上下文:进了工作区就是工作区(会话是所有成员文件夹合并的),否则是项目。
func (a *App) sideBCtx() (name, icon string, p core.ProjectInfo) {
	if a.ws != nil {
		return a.ws.Name, "layers", a.wsProject(*a.ws)
	}
	p = a.sideBProject()
	return p.Name, "folder", p
}

// 下拉里列表区的最大高度:项目 / 工作区很多时在这里滚动,不让下拉无限变长;底部的添加入口固定。
const projPickMaxH = 300

// projPickPopup 方案 B 的项目下拉:工作区 + 项目各一行(成员文件夹不再逐个列出),列表限高可滚,
// 底部固定「添加项目… / 新建工作区…」。
func (a *App) projPickPopup(c *ui.Context, anchor ui.Element) {
	a.popOpen("projpick", func(open *bool) {
		ui.PopoverBase(c, anchor, open, func(panel ui.Element) {
			a.popPanel(panel, 280).AttachTo(anchor, ui.AnchorTopLeft, ui.AnchorBottomLeft).Margin(4, 0, 0, 0)
			ws := a.registry().Workspaces()
			kit.VScroll(c).Key("pp-scroll").MaxHeight(projPickMaxH).Children(func() {
				ui.Column(c).Children(func() {
					if len(ws) > 0 {
						kit.Cap(c, "工作区")
					}
					for _, w := range ws {
						on := a.activeWsID() == w.ID
						if a.mi(c, "ws-"+w.ID, "layers", w.Name, itoa(len(w.Folders))+" 个文件夹", on).Clicked() {
							a.popup = ""
							a.selectWorkspace(w)
						}
					}
					kit.Cap(c, "项目")
					for _, p := range a.sideProjects() { // 含工作区成员:单独选它 = 只看这个文件夹(退出工作区)
						a.projPickRow(c, p)
					}
				})
			})
			kit.Sep(c)
			if a.mi(c, "pp-add-proj", "folder", "添加项目…", "⌘ O", false).Clicked() {
				a.popup = ""
				a.openAddProject()
			}
			if a.mi(c, "pp-add-ws", "layers", "新建工作区…", "", false).Clicked() {
				a.popup = ""
				a.openWsDialog("", "")
			}
		})
	})
}

func (a *App) projPickRow(c *ui.Context, p core.ProjectInfo) {
	on := a.ctxIsProject(p)
	// 同一路径可能对应多个会话目录(如 MiniCode 的几个工作树),键里带上项目 ID,避免多行共用一个控件状态
	key := "pp-" + projects.Clean(p.Path) + "|" + p.ID
	right := ""
	if p.Count > 0 {
		right = itoa(p.Count)
	}
	// 同名项目(如多个 001)靠悬停提示里的完整路径区分
	if a.mi(c, key, "folder", p.Name, right, on).Tooltip(tildePath(p.Path)).Clicked() {
		a.popup = ""
		a.selectProject(p)
	}
}
