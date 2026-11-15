package app

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/demo"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/projects"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 添加项目 / 新建·编辑工作区 / 通用确认框(设计稿「项目与工作区」页)。

// projDlg 是这几个对话框共用的状态;kind 为空表示没有打开。
type projDlg struct {
	kind string // addProj / ws / confirm / branch-dirty

	// 添加项目
	path    string
	err     string // 红色错误
	hint    string // 黄色提示(带一个动作)
	hintBtn string
	hintDo  func()

	// 工作区编辑
	wsID    string
	name    string
	folders []projects.Folder
	loadMD  bool
	addPath string
	pickLst bool // 展开「从已有项目添加」列表

	// 确认框
	title, body, okText string
	danger              bool
	ok                  func()

	branchName string
	branchFn   func()
}

func (a *App) closeProjDlg() { a.pd = projDlg{} }

// openAddProject 打开「添加项目」。
func (a *App) openAddProject() { a.pd = projDlg{kind: "addProj"} }

// openWsDialog 打开工作区对话框:id 为空 = 新建;withFolder 非空时预置第一个文件夹(从项目行「加入工作区」来)。
func (a *App) openWsDialog(id, withFolder string) {
	d := projDlg{kind: "ws"}
	if id != "" {
		if w, ok := a.registry().Workspace(id); ok {
			d.wsID, d.name, d.folders, d.loadMD = w.ID, w.Name, w.Folders, w.LoadMD
		}
	} else if withFolder != "" {
		d.folders = []projects.Folder{{Path: withFolder}}
	}
	a.pd = d
}

// confirmDlg 打开确认框。
func (a *App) confirmDlg(title, body, okText string, danger bool, ok func()) {
	a.pd = projDlg{kind: "confirm", title: title, body: body, okText: okText, danger: danger, ok: ok}
}

// pickFolders 弹系统文件夹选择器(真实模式);演示模式没有,提示直接输入路径。
func (a *App) pickFolders(multi bool, apply func([]string)) {
	if a.win == nil || a.demoWs() {
		a.toast = "演示模式没有系统文件夹选择器,请直接输入路径"
		return
	}
	a.bg("pick-folder", func() func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Title: "选择文件夹", Directory: true, Multiple: multi, CreateDirectories: true, ButtonLabel: "选择", Parent: a.win})
		return func() {
			if err != nil {
				log.Warn("文件夹选择器出错", "err", err)
				return
			}
			if len(paths) > 0 {
				apply(paths)
			}
		}
	})
}

// ───────── 提交动作 ─────────

// submitAddProject 校验并添加项目,按错误种类给出「下一步」。
func (a *App) submitAddProject() {
	d := &a.pd
	d.err, d.hint, d.hintDo = "", "", nil
	p, err := a.registry().AddProject(d.path)
	switch err {
	case nil:
		a.closeProjDlg()
		a.toast = "已添加项目 " + filepath.Base(p)
		a.projOpen["path:"+p] = true
		log.Info("添加项目", "path", p)
	case projects.ErrDup:
		d.hint, d.hintBtn = "这个项目已经在列表里。直接切换过去?", "切换过去"
		d.hintDo = func() {
			a.closeProjDlg()
			if pr, ok := a.projectByPath(p); ok {
				a.NewSessionIn(pr)
			}
		}
	case projects.ErrNotDir:
		dir := filepath.Dir(p)
		d.hint, d.hintBtn = "这是一个文件。要用它所在的文件夹吗?", "使用 "+tildePath(dir)
		d.hintDo = func() { a.pd.path, a.pd.hint = dir, "" }
	case projects.ErrEmpty:
		d.err = "请输入文件夹路径,或点「浏览…」选择"
	default:
		d.err = err.Error() + "。检查一下路径拼写(~ 代表你的家目录)"
		if sug := a.suggestPath(d.path); sug != "" {
			d.hint, d.hintBtn = "你是不是想输入 "+sug+"?", "使用"
			d.hintDo = func() { a.pd.path, a.pd.err, a.pd.hint = sug, "", "" }
		}
	}
}

// suggestPath 路径不存在时,在同级目录(演示是样例列表)里找名字最接近(编辑距离 ≤ 2)的一个。
func (a *App) suggestPath(p string) string {
	c := projects.Clean(p)
	if c == "" {
		return ""
	}
	base := filepath.Base(c)
	var cands []string
	if a.demoWs() {
		for _, f := range demo.Discovered() {
			cands = append(cands, f.Path)
		}
	} else if es, err := os.ReadDir(filepath.Dir(c)); err == nil {
		for _, e := range es {
			if e.IsDir() && len(cands) < 400 {
				cands = append(cands, tildePath(filepath.Join(filepath.Dir(c), e.Name())))
			}
		}
	}
	best, bd := "", 3
	for _, cand := range cands {
		if d := editDistance(base, filepath.Base(projects.Clean(cand))); d > 0 && d < bd {
			best, bd = cand, d
		}
	}
	return best
}

// editDistance 是两个字符串的编辑距离(按字符)。
func editDistance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur := make([]int, len(rb)+1)
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(rb)]
}

// submitWorkspace 校验并保存工作区。
func (a *App) submitWorkspace() {
	d := &a.pd
	d.err = ""
	if len(d.folders) < 2 {
		d.err = "工作区至少要有 2 个文件夹(只有 1 个请直接添加为项目)"
		return
	}
	w, err := a.registry().SaveWorkspace(projects.Workspace{ID: d.wsID, Name: d.name, Folders: d.folders, LoadMD: d.loadMD})
	if err != nil {
		d.err = err.Error()
		return
	}
	a.closeProjDlg()
	a.projOpen[w.ID] = true
	if a.ws != nil && a.ws.ID == w.ID { // 正在用的工作区被改了:重新进入使文件树生效
		id := w.ID
		a.leaveWorkspace()
		a.ActivateWorkspace(id)
	}
	a.toast = map[bool]string{true: "已更新工作区「" + w.Name + "」", false: "已创建工作区「" + w.Name + "」"}[d.wsID != ""]
}

func (a *App) addWsFolder(path string) {
	d := &a.pd
	c, err := a.registry().Check(path)
	if err != nil {
		d.err = err.Error()
		return
	}
	for _, f := range d.folders {
		if f.Path == c {
			d.err = "这个文件夹已经在工作区里"
			return
		}
	}
	if len(d.folders) >= projects.MaxFolders {
		d.err = projects.ErrTooMany.Error()
		return
	}
	d.err = ""
	d.folders = append(d.folders, projects.Folder{Path: c})
	if d.name == "" && len(d.folders) == 1 {
		d.name = filepath.Base(c) + " 工作区"
	}
}

// ───────── 绘制 ─────────

// projDialogs 画当前打开的对话框(在 shell 里与命令面板同层)。
func (a *App) projDialogs(c *ui.Context) {
	switch a.pd.kind {
	case "addProj":
		a.addProjectDialog(c)
	case "ws":
		a.workspaceDialog(c)
	case "confirm":
		a.confirmDialog(c)
	case "branch-dirty":
		a.branchDirtyDialog(c)
	}
}

// dlgFrame 是这几个对话框共用的外壳:遮罩 + 卡片 + 标题行。
func (a *App) dlgFrame(c *ui.Context, key string, w float32, title string, body func()) {
	t := a.tk
	open := true
	ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
		backdrop.Background(t.Scrim).Justify(ui.Center).AlignItems(ui.Center)
		panel.Width(w).Background(t.Raised).Radius(t.RXl()).Column().Border(1, t.Bd1).Padding(14, 16).Gap(6)
		for _, sh := range t.Shadow3() {
			panel.Shadow(sh.X, sh.Y, sh.Blur, sh.Spread, sh.Color)
		}
		a.dialogIn(key, backdrop, panel)
		panel.Children(func() {
			ui.Row(c).AlignItems(ui.Center).Children(func() {
				ui.Text(c, title).FontSize(theme.FsHead).FontWeight(700).Grow(1)
				kit.Ib(c, "x", kit.IbOpt{Small: true, Tip: "关闭 Esc"}).OnClick(a.closeProjDlg)
			})
			body()
		})
	})
	if !open {
		a.closeProjDlg()
	}
}

// dlgInput 是单行输入框(带描边,聚焦时强调色)。
func (a *App) dlgInput(c *ui.Context, key string, v *string, ph string, autoFocus bool, submit func()) {
	t := a.tk
	box := ui.Row(c).Key(key+"-box").Grow(1).MinWidth(0).Height(28).PaddingX(8).AlignItems(ui.Center).Background(t.Fill1).Radius(t.RSm()).Border(1, t.Bd2)
	if box.FocusWithin() {
		box.BorderColor(t.Ac)
	}
	box.Children(func() {
		in := ui.TextInputBase(c.Key(key), v).Grow(1).MinWidth(0).FontSize(theme.FsBody).Placeholder(ph).PlaceholderColor(t.Tx4)
		if autoFocus {
			in.AutoFocus()
		}
		if submit != nil {
			in.OnSubmit(submit)
		}
	})
}

func (a *App) dlgMsg(c *ui.Context) {
	t := a.tk
	d := &a.pd
	if d.err != "" {
		ui.Row(c).Gap(6).Padding(6, 8).Radius(t.RSm()).Background(t.ErrSoft).TextColor(t.Err).FontSize(theme.FsCap).AlignItems(ui.Center).Children(func() {
			kit.Ic(c, "alert", kit.S12)
			ui.Text(c, d.err).Grow(1)
		})
	}
	if d.hint != "" {
		ui.Row(c).Gap(6).Padding(6, 8).Radius(t.RSm()).Background(t.WarnSoft).TextColor(t.Warn).FontSize(theme.FsCap).AlignItems(ui.Center).Children(func() {
			kit.Ic(c, "info", kit.S12)
			ui.Text(c, d.hint).Grow(1).TextColor(t.Tx1)
			if d.hintDo != nil && kit.Btn(c, d.hintBtn, kit.BtnOpt{Small: true}).Clicked() {
				f := d.hintDo
				f()
			}
		})
	}
}

// foundItem 是「最近 · 发现」里的一行。
type foundItem struct {
	name, path, branch string
	repo, added        bool
}

// discoverProjects 列出可添加的项目:演示是内置样例,真实是当前项目的同级仓库。
func (a *App) discoverProjects() []foundItem {
	added := map[string]bool{}
	for _, p := range a.sideProjects() {
		added[projects.Clean(p.Path)] = true
	}
	var out []foundItem
	if a.demoWs() {
		for _, f := range demo.Discovered() {
			p := projects.Clean(f.Path)
			out = append(out, foundItem{name: f.Name, path: f.Path, branch: f.Branch, repo: f.Branch != "", added: added[p]})
		}
		return out
	}
	parent := filepath.Dir(a.proj)
	es, _ := os.ReadDir(parent)
	for _, e := range es {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") || len(out) >= 6 {
			continue
		}
		p := filepath.Join(parent, e.Name())
		if !dirHasGit(p) {
			continue
		}
		out = append(out, foundItem{name: e.Name(), path: tildePath(p), branch: gitBranchOf(p), repo: true, added: added[p]})
	}
	return out
}

// gitBranchOf 不起 git 进程,直接读 .git/HEAD 得到分支名(读不到返回空)。
func gitBranchOf(dir string) string {
	b, err := os.ReadFile(filepath.Join(dir, ".git", "HEAD"))
	if err != nil {
		return ""
	}
	s := strings.TrimSpace(string(b))
	if r, ok := strings.CutPrefix(s, "ref: refs/heads/"); ok {
		return r
	}
	if len(s) > 7 {
		return s[:7]
	}
	return s
}

func (a *App) addProjectDialog(c *ui.Context) {
	t := a.tk
	d := &a.pd
	a.dlgFrame(c, "addProj", 460, "添加项目", func() {
		ui.Text(c, "项目就是一个文件夹,Claude 会话在这个目录里运行。").FontSize(theme.FsCap).TextColor(t.Tx3)
		ui.Row(c).Gap(6).AlignItems(ui.Center).Margin(4, 0, 0, 0).Children(func() {
			a.dlgInput(c, "ap-path", &d.path, "~/Code/my-project", true, a.submitAddProject)
			kit.Btn(c, "浏览…", kit.BtnOpt{Icon: "folder"}).OnClick(func() {
				a.pickFolders(false, func(ps []string) { a.pd.path, a.pd.err, a.pd.hint = tildePath(ps[0]), "", "" })
			})
		})
		a.dlgMsg(c)
		items := a.discoverProjects()
		if len(items) > 0 {
			ui.Text(c, "最近 · 发现的项目").FontSize(theme.FsCap).TextColor(t.Tx3).Margin(8, 0, 0, 0)
			for i, f := range items {
				f := f
				r := kit.RowEl(c, projects.Clean(d.path) == projects.Clean(f.path) && d.path != "").Key("ap-f-" + itoa(i)).Height(36).Gap(8)
				if !f.added {
					r.OnClick(func() { d.path, d.err, d.hint = f.path, "", "" })
				}
				r.Children(func() {
					icon := "folder"
					if f.repo {
						icon = "branch"
					}
					kit.Ic(c, icon, kit.S12).TextColor(t.Tx3)
					ui.Column(c).Grow(1).MinWidth(0).Children(func() {
						ui.Text(c, f.name).SingleLine().TextColor(map[bool]ui.Color{true: t.Tx4, false: t.Tx1}[f.added])
						ui.Text(c, f.path).Font("monospace").FontSize(theme.FsMicro).TextColor(t.Tx4).SingleLine()
					})
					desc := "不是 Git 仓库"
					if f.repo {
						desc = "是 Git 仓库 · " + f.branch
					}
					if f.added {
						desc = "已添加"
					}
					ui.Text(c, desc).FontSize(theme.FsCap).TextColor(t.Tx3).Shrink(0)
				})
			}
		}
		ui.Row(c).Gap(8).AlignItems(ui.Center).Margin(8, 0, 0, 0).Children(func() {
			ui.Text(c, "已在列表里的项目会置灰并显示「已添加」").FontSize(theme.FsCap).TextColor(t.Tx3).Grow(1)
			kit.Btn(c, "取消", kit.BtnOpt{Variant: kit.BtnGhost}).OnClick(a.closeProjDlg)
			kit.Btn(c, "添加", kit.BtnOpt{Variant: kit.BtnPri}).OnClick(a.submitAddProject)
		})
	})
}

// repoNote 工作区对话框里文件夹行右侧的「分支 · 改动」说明。
func (a *App) repoNote(path string) string {
	if a.demoWs() {
		for _, f := range demo.WorkspaceFolders() {
			if f.Path == path {
				if f.Repo == nil {
					return "非仓库"
				}
				cs, _ := f.Repo.Status()
				n := map[string]bool{}
				for _, c := range cs {
					n[c.Path] = true
				}
				if len(n) == 0 {
					return f.Repo.Current() + " · 干净"
				}
				return f.Repo.Current() + " · " + itoa(len(n)) + " 项改动"
			}
		}
		return ""
	}
	if !dirHasGit(path) {
		return "非仓库"
	}
	return gitBranchOf(path)
}

func (a *App) workspaceDialog(c *ui.Context) {
	t := a.tk
	d := &a.pd
	title := "新建工作区"
	okText := "创建工作区"
	if d.wsID != "" {
		title, okText = "编辑工作区", "保存"
	}
	a.dlgFrame(c, "ws", 500, title, func() {
		ui.Text(c, "工作区 = 一组相关的文件夹。会话在「主文件夹」里运行,其余文件夹同时对 Claude 开放读写。").FontSize(theme.FsCap).TextColor(t.Tx3)
		ui.Text(c, "名称").FontSize(theme.FsCap).TextColor(t.Tx3).Margin(4, 0, 0, 0)
		ui.Row(c).Children(func() { a.dlgInput(c, "ws-name", &d.name, "全栈工作区", d.wsID == "", nil) })
		ui.Text(c, "文件夹("+itoa(len(d.folders))+" / "+itoa(projects.MaxFolders)+",顺序 = 文件树顺序)").FontSize(theme.FsCap).TextColor(t.Tx3).Margin(4, 0, 0, 0)
		if len(d.folders) == 0 {
			ui.Text(c, "还没有文件夹。从已有项目添加,或输入路径 / 浏览选择。").FontSize(theme.FsCap).TextColor(t.Tx4).Padding(8, 4)
		}
		for i, f := range d.folders {
			i, f := i, f
			ui.Row(c).Key("ws-f-" + itoa(i)).Height(36).Gap(8).AlignItems(ui.Center).PaddingX(4).Children(func() {
				kit.Ic(c, "folder", kit.S12).TextColor(t.Tx3)
				ui.Column(c).Grow(1).MinWidth(0).Children(func() {
					ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
						ui.Text(c, f.Label()).SingleLine().Shrink(1)
						if i == 0 {
							kit.Tag(c, "主文件夹", kit.KAc)
						}
					})
					ui.Text(c, tildePath(f.Path)).Font("monospace").FontSize(theme.FsMicro).TextColor(t.Tx4).SingleLine()
				})
				ui.Text(c, a.repoNote(f.Path)).FontSize(theme.FsCap).TextColor(t.Tx3).Shrink(0)
				if i > 0 {
					kit.Ib(c, "arrow", kit.IbOpt{Small: true, Tip: "设为主文件夹"}).OnClick(func() {
						d.folders[0], d.folders[i] = d.folders[i], d.folders[0]
					})
				}
				kit.Ib(c, "x", kit.IbOpt{Small: true, Tip: "移出工作区"}).OnClick(func() {
					d.folders = append(d.folders[:i:i], d.folders[i+1:]...)
					d.err = ""
				})
			})
		}
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			a.dlgInput(c, "ws-path", &d.addPath, "粘贴文件夹路径后回车添加", false, func() {
				a.addWsFolder(d.addPath)
				if a.pd.err == "" {
					a.pd.addPath = ""
				}
			})
			kit.Btn(c, "浏览…", kit.BtnOpt{Icon: "folder"}).OnClick(func() {
				a.pickFolders(true, func(ps []string) {
					for _, p := range ps {
						a.addWsFolder(p)
					}
				})
			})
		})
		ui.Row(c).Gap(6).Children(func() {
			kit.Btn(c, "从已有项目添加", kit.BtnOpt{Small: true, Icon: "plus"}).OnClick(func() { d.pickLst = !d.pickLst })
		})
		if d.pickLst {
			have := map[string]bool{}
			for _, f := range d.folders {
				have[f.Path] = true
			}
			n := 0
			for _, p := range a.sideProjects() {
				p := p
				cp := projects.Clean(p.Path)
				if have[cp] {
					continue
				}
				n++
				r := kit.RowEl(c, false).Key("ws-pick-" + p.ID).Height(26).Gap(6)
				r.OnClick(func() { a.addWsFolder(p.Path) })
				r.Children(func() {
					kit.Ic(c, "plus", kit.S12).TextColor(t.Ac)
					ui.Text(c, p.Name).Grow(1).SingleLine()
					ui.Text(c, tildePath(p.Path)).Font("monospace").FontSize(theme.FsMicro).TextColor(t.Tx4).SingleLine()
				})
			}
			if n == 0 {
				ui.Text(c, "没有可添加的项目了").FontSize(theme.FsCap).TextColor(t.Tx4).Padding(4, 8)
			}
		}
		a.dlgMsg(c)
		ui.Row(c).Gap(8).AlignItems(ui.Center).Margin(6, 0, 0, 0).Children(func() {
			if kit.Tog(c, d.loadMD, "ws-md") {
				d.loadMD = !d.loadMD
			}
			ui.Text(c, "让 Claude 同时读取附加文件夹里的 CLAUDE.md 与规则").FontSize(theme.FsCap).TextColor(t.Tx2).Grow(1)
		})
		if len(d.folders) >= 2 {
			ui.Row(c).Gap(6).Padding(6, 8).Radius(t.RSm()).Background(t.Fill1).FontSize(theme.FsCap).AlignItems(ui.Center).Children(func() {
				kit.Ic(c, "shield", kit.S12).TextColor(t.Tx3)
				ui.Text(c, "保存后 Claude 将获得 "+itoa(len(d.folders)-1)+" 个附加文件夹的读写权限(--add-dir)。可在 设置 → 权限与信任 里撤销。").TextColor(t.Tx2).Grow(1)
			})
		}
		ui.Row(c).Gap(8).AlignItems(ui.Center).Margin(6, 0, 0, 0).Children(func() {
			ui.Text(c, "保存到 ~/.minicode/workspaces.json,不会往项目里写文件").FontSize(theme.FsCap).TextColor(t.Tx3).Grow(1)
			kit.Btn(c, "取消", kit.BtnOpt{Variant: kit.BtnGhost}).OnClick(a.closeProjDlg)
			kit.Btn(c, okText, kit.BtnOpt{Variant: kit.BtnPri}).OnClick(a.submitWorkspace)
		})
	})
}

func (a *App) confirmDialog(c *ui.Context) {
	t := a.tk
	d := &a.pd
	a.dlgFrame(c, "confirm", 400, d.title, func() {
		ui.Text(c, d.body).FontSize(theme.FsBody).TextColor(t.Tx2).LineHeight(1.5)
		ui.Row(c).Gap(8).Justify(ui.End).Margin(10, 0, 0, 0).Children(func() {
			kit.Btn(c, "取消", kit.BtnOpt{Variant: kit.BtnGhost}).OnClick(a.closeProjDlg)
			v := kit.BtnPri
			if d.danger {
				v = kit.BtnDanger
			}
			if kit.Btn(c, d.okText, kit.BtnOpt{Variant: v}).Clicked() {
				f := d.ok
				a.closeProjDlg()
				if f != nil {
					f()
				}
			}
		})
	})
}
