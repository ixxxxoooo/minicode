package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/demo"

	"github.com/ixxxxoooo/minicode/internal/vcs"
	"github.com/ixxxxoooo/minicode/internal/workspace"
)

// demoProject 是演示模式下沙盒项目的名字(设计稿里的 router-demo)。
const demoProject = "router-demo"

// projLabel 是标题栏、侧栏等处显示的当前项目名:真实模式取目录名,演示模式是沙盒名。
func (a *App) projLabel() string {
	if a.proj != "" {
		return filepath.Base(a.proj)
	}
	return demoProject
}

// SetProject 切到真实项目目录:IDE 文件树 / 编辑器改读磁盘,仓库(分支、改动、图谱)改读该目录的
// Git;不是 Git 仓库时分支为空、源代码管理显示「还不是 Git 仓库」。目录没变时什么都不做。
// 打开仓库要跑 git,放后台;窗口还没建好(启动阶段、测试)时同步执行。
func (a *App) SetProject(dir string) {
	if a.ws != nil { // 工作区里打开成员文件夹的会话:留在工作区;别处的会话:退出工作区
		if a.ws.Has(dir) {
			return
		}
		a.leaveWorkspaceKeep()
	}
	if dir == "" || dir == a.proj {
		return
	}
	a.proj = dir
	a.noteRecentProject(dir)
	log.Info("切换项目", "dir", dir)
	a.SetRepo(nil)
	a.sc = scmState{gen: a.sc.gen + 1}
	a.ide.useFS(a.wrapIDE(workspace.NewOS(dir)), map[string]string{}, a.headOfRepo)
	a.ide.termLines = nil
	a.lspReset()
	a.closeTerm() // 终端跟着项目走:旧目录里的 shell 结束,下次打开面板在新目录重开
	a.bg("open-repo", func() func() {
		cli, err := vcs.Open(dir)
		return func() {
			if a.proj != dir {
				return // 期间又切走了
			}
			if err != nil {
				log.Info("项目不是 Git 仓库", "dir", dir)
				return
			}
			a.SetRepo(cli)
		}
	})
}

// headOfRepo 取 HEAD 里某文件的内容(没有仓库或文件不在 HEAD 里时 ok=false)。
func (a *App) headOfRepo(p string) (string, bool) {
	if workspace.IsExternalPath(p) {
		return "", false
	}
	r, rel := a.repoAt(p)
	if r == nil {
		return "", false
	}
	return r.Show("HEAD", rel)
}

// welcomeStatus 是欢迎页问候下面的状态行。演示模式保持设计稿里的文案,真实模式读仓库。
func (a *App) welcomeStatus() string {
	if a.proj == "" {
		if a.demoWs() {
			return demo.WelcomeStatusLine(a.welcomeLine)
		}
		return a.branch + " · 3 项未提交改动 · 上次会话 5 分钟前"
	}
	if a.repo == nil {
		return "不是 Git 仓库"
	}
	a.scmTick()
	if !a.sc.loaded {
		return a.branch
	}
	seen := map[string]bool{}
	for _, c := range a.sc.changes {
		seen[c.Path] = true
	}
	if len(seen) == 0 {
		return a.branch + " · 工作区干净"
	}
	return fmt.Sprintf("%s · %d 项未提交改动", a.branch, len(seen))
}

// isCurProject 侧栏的某个项目是不是当前项目(演示按沙盒名,真实按目录)。
func (a *App) isCurProject(p core.ProjectInfo) bool {
	if a.proj != "" {
		return p.Path == a.proj
	}
	return p.ID == demoProject
}

// tildePath 把家目录前缀缩写成 ~(设置页里显示项目路径用)。
func tildePath(p string) string {
	if h, err := os.UserHomeDir(); err == nil && h != "" && strings.HasPrefix(p, h) {
		return "~" + strings.TrimPrefix(p, h)
	}
	return p
}
