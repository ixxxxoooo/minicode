package app

import (
	"path/filepath"
	"sort"

	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/projects"
)

// 侧栏「选中上下文」:选中一个项目 / 工作区(或打开其下的会话),聊天栏与 IDE 就一起切到它。
// 工作区是平的:成员文件夹之间没有「主」,工作区的会话 = 所有成员文件夹的会话合在一起。

// WorkspaceFor 给启动会话用:cwd 属于当前正在使用的工作区就用它,否则退回「第一个文件夹正是 cwd」的工作区(旧行为)。
func (a *App) WorkspaceFor(dir string) (projects.Workspace, bool) {
	if a == nil {
		return projects.Workspace{}, false
	}
	if a.ws != nil && a.ws.Has(dir) {
		return *a.ws, true
	}
	return a.registry().ByPrimary(dir)
}

// sessionDir 会话所在的项目目录(索引里没有的会话取不到)。
func (a *App) sessionDir(id string) (string, bool) {
	if id == "" {
		return "", false
	}
	if _, p, ok := a.sessionByID(id); ok {
		return p.Path, true
	}
	return "", false
}

// wsStartDir 工作区里新会话的启动目录:当前会话正好在某个成员文件夹里就沿用它,否则取列表里第一个。
func (a *App) wsStartDir(w projects.Workspace) string {
	if d, ok := a.sessionDir(a.curSess); ok && w.Has(d) {
		return projects.Clean(d)
	}
	if a.proj != "" && w.Has(a.proj) {
		return projects.Clean(a.proj)
	}
	return w.Primary()
}

// wsProject 把工作区所有成员文件夹的会话合成一个「虚拟项目」(按最近更新排序),侧栏与方案 B 都用它列会话。
func (a *App) wsProject(w projects.Workspace) core.ProjectInfo {
	dir := a.wsStartDir(w)
	out := core.ProjectInfo{ID: "wsproj:" + w.ID, Name: w.Name, Path: dir}
	for _, f := range w.Folders {
		if p, ok := a.projectByPath(f.Path); ok {
			out.Sessions = append(out.Sessions, p.Sessions...)
			out.Count += p.Count
		}
	}
	sort.SliceStable(out.Sessions, func(i, j int) bool { return out.Sessions[i].Updated > out.Sessions[j].Updated })
	if out.Count == 0 {
		out.Count = len(out.Sessions)
	}
	return out
}

// startProject 工作区新会话要落在的「项目」(没有会话的文件夹也有一条空项目)。
func (a *App) startProject(w projects.Workspace) core.ProjectInfo {
	dir := a.wsStartDir(w)
	if p, ok := a.projectByPath(dir); ok {
		return p
	}
	return core.ProjectInfo{ID: "path:" + dir, Name: filepath.Base(dir), Path: dir}
}

// firstLive 取一组会话里最近的、没有归档的一条。
func firstLive(ss []core.SessionInfo) (core.SessionInfo, bool) {
	for _, s := range ss {
		if !s.Archived {
			return s, true
		}
	}
	return core.SessionInfo{}, false
}

// selectProject 选中一个项目:退出工作区,IDE(文件树 / 终端 / 仓库)跟着换到这个项目;
// 当前会话不在这个项目里,聊天栏就切到它最近的会话,一个都没有则新开一个。
func (a *App) selectProject(p core.ProjectInfo) {
	log.Info("选中项目", "dir", p.Path, "name", p.Name)
	real := a.proj != "" // 真实模式才有项目目录;演示的文件系统是固定沙盒,不跟着换
	if a.ws != nil {
		if a.demoWs() {
			a.leaveWorkspace()
		} else {
			a.leaveWorkspaceKeep()
		}
	}
	if real {
		a.SetProject(p.Path)
	}
	a.projOpen[p.ID] = true
	if d, ok := a.sessionDir(a.curSess); ok && projects.Clean(d) == projects.Clean(p.Path) {
		a.sideBF = p.ID
		return
	}
	if s, ok := firstLive(p.Sessions); ok {
		a.OpenSession(s.ID)
	} else {
		a.NewSessionIn(p)
	}
	a.sideBF = p.ID
}

// selectWorkspace 选中一个工作区:文件树 / 搜索 / 源代码管理变成多文件夹;
// 当前会话不在任何成员文件夹里,聊天栏切到工作区最近的会话(所有成员的会话合在一起),没有就新开一个。
func (a *App) selectWorkspace(w projects.Workspace) {
	log.Info("选中工作区", "id", w.ID, "name", w.Name, "folders", len(w.Folders))
	a.ActivateWorkspace(w.ID)
	if a.ws == nil || a.ws.ID != w.ID { // 进入失败(文件夹都找不到)
		return
	}
	a.projOpen[w.ID] = true
	a.sideBF = ""
	if d, ok := a.sessionDir(a.curSess); ok && w.Has(d) {
		return
	}
	if s, ok := firstLive(a.wsProject(w).Sessions); ok {
		a.OpenSession(s.ID)
		return
	}
	a.NewSessionIn(a.startProject(w))
}

// ctxIsProject 侧栏的某个项目是不是当前上下文(没进工作区,且当前会话在它里面)。
func (a *App) ctxIsProject(p core.ProjectInfo) bool {
	if a.ws != nil {
		return false
	}
	if d, ok := a.sessionDir(a.curSess); ok {
		return projects.Clean(d) == projects.Clean(p.Path)
	}
	return a.isCurProject(p)
}
