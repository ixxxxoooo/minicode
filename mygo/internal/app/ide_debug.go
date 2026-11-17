package app

import (
	"strings"

	"github.com/egoist/mygo/ui/editor"
	"github.com/egoist/mygo/ui/richtext"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/vcs"
)

// EditorDemo 按「动作:参数」驱动编辑器进入某个状态(cmd/shot 截图用,多个动作用逗号隔开):
// cursor:行:列(1 起)、select:行:列-行:列、find:查询、scroll:行、peek、rename、acts、inline、refs[:demo|empty|fold|loading|err|sel:N|normalize]、def、hover:行:列、open:路径、conflict(合并冲突标签)、review(审查标签;review:empty 空态)、expand:路径、newfile|newdir:文件夹:名字、renamef:路径:名字、foerr、discardask:路径、cmpfiles:提交、resetask:序号、delask:路径、rename:会话id、archived:项目id、scm、syncall、repobusy:N、search:查询、scope:<行>|scope:hover(所在范围高亮)|scope:文件夹(搜索范围)、pop:弹层键、wizard[:1|done](逐个提交向导,配合 -ws)、md:src|wysiwyg(Markdown 视图)、mdkeep(块写回演示样例)、wrap、format、drag:@路径(对话栏可放态)、mdedit:内容、goto:行、gotoline:行[:列]、multicursor、blockcomment、extmod[:auto|clean|dirty|deleted]、crumb[:file|sym|路径]、termrename、termtabs[:N](多终端长标题演示)、treefilter[:关键字](文件树过滤条);
// queue(排两条)、steer[:正文](登记引导)、ask[:题号](切问答卡题号,1 起)、latest[:fresh](滚离底部, fresh=有新内容圆点)、thinking(演示思考折叠跑马灯)、sub[:卡片id](子 Agent 标签)、wsdiag:scan|done|trunc|empty|err(工作区诊断各态)。
func (a *App) EditorDemo(spec string) {
	for _, act := range strings.Split(spec, ",") {
		name, arg, _ := strings.Cut(act, ":")
		if a.flowDebug(name, arg) {
			continue
		}
		if name == "wsdiag" {
			a.SetMode(ModeIDE)
			a.ide.bottomOpen, a.ide.bottomTab = true, "problems"
			a.ide.probScope = "ws"
			if arg == "" {
				arg = "scan"
			}
			a.wsDiagDemo = arg
			continue
		}
		tb := a.ide.cur()
		pos := func(s string) editor.Pos {
			var l, c int
			parts := strings.Split(s, ":")
			if len(parts) == 2 {
				l, c = atoi(parts[0]), atoi(parts[1])
			}
			return editor.Pos{Line: max(l-1, 0), Col: max(c-1, 0)}
		}
		switch name {
		case "motion":
			a.openSettings("motion")
			a.openMotionCatalog()
			continue
		case "closeall": // 关掉全部标签(看工作区起始页)
			for len(a.ide.tabs) > 0 {
				a.ide.closeTab(a.ide.tabs[0].id)
			}
			continue
		case "right": // 展开右栏
			a.rightOn = true
			continue
		case "panelmax":
			a.togglePanelMax()
			continue
		case "chatmax":
			a.toggleChatMax()
			continue
		}
		if name == "graph" {
			a.openGraph()
			if arg != "" {
				a.graphTab.sel = arg
				a.scmCommitFiles(arg)
			}
			continue
		}
		switch name {
		case "search": // search:查询:打开搜索并填入查询
			a.OpenSearch()
			a.SetSearch(arg, "", false)
			a.runSearch(true)
			continue
		case "scope": // scope:<行号>|hover:编辑器所在范围;否则多根搜索限定文件夹
			if arg == "hover" {
				a.dbgScopeHover()
				continue
			}
			if n, ok := parseScopeLineArg(arg); ok {
				a.dbgScopeLine(n)
				continue
			}
			a.setSearchScope(arg)
			a.runSearch(true)
			continue
		case "expand": // expand:<树路径>[;<树路径>…] 展开文件树里的文件夹并切到文件导航
			for _, q := range strings.Split(arg, ";") {
				a.ide.open[q] = true
			}
			a.nav, a.sideOpen = "files", true
			continue
		case "newfile", "newdir": // newfile:<文件夹>:<已输入的名字> 文件树行内新建(根用空)
			dir, buf, _ := strings.Cut(arg, ":")
			a.SetMode(ModeIDE)
			a.beginNew(dir, name == "newdir")
			a.fo.buf = buf
			continue
		case "renamef": // renamef:<树路径>:<已输入的名字> 文件树行内重命名
			p, buf, _ := strings.Cut(arg, ":")
			a.SetMode(ModeIDE)
			a.beginRenameFile(p)
			if buf != "" {
				a.fo.buf = buf
			}
			continue
		case "discardask": // discardask:<路径> 放弃更改确认(路径留空 = 全部更改)
			a.dbgSCM()
			_, wk := a.sc.split()
			if arg != "" {
				var one []vcs.Change
				for _, c := range wk {
					if c.Path == arg {
						one = append(one, c)
					}
				}
				wk = one
			}
			a.scmDiscardAsk(wk)
			continue
		case "cmpfiles": // cmpfiles:<提交> 「与工作区比较」的文件选择
			a.dbgSCM()
			for _, cm := range a.sc.log {
				if cm.Hash == arg {
					a.scmCompareWork(cm)
				}
			}
			continue
		case "resetask": // resetask:<第几个提交,0 起> 回退到这里的确认
			a.dbgSCM()
			if i := atoi(arg); i < len(a.sc.log) {
				n, _ := a.resetInfo(a.sc.log[i])
				a.askReset(a.sc.log[i], n)
			}
			continue
		case "foerr": // 提交当前行内输入(用来到达「已存在同名」的错误态)
			a.commitFileOp()
			continue
		case "delask": // delask:<树路径> 弹出「移到废纸篓」确认框
			a.SetMode(ModeIDE)
			a.askDelete(arg, strings.HasSuffix(arg, "/") || len(a.ide.listing(arg)) > 0)
			continue
		case "rename": // rename:<会话 id> 侧栏行内重命名
			a.nav, a.sideOpen = "", true
			a.beginRename(arg)
			continue
		case "archived": // archived:<项目 id> 展开「已归档」
			a.projOpen["arch:"+arg] = true
			continue
		case "scm": // 打开源代码管理
			a.OpenSCM()
			continue
		case "syncall": // 工作区:全部同步(演示:web-dashboard 第一次失败留「重试」,docs-site 有冲突被跳过)
			a.syncAllRepos(false)
			continue
		case "repobusy": // repobusy:N:第 N 个仓库行显示转圈
			if i := atoi(arg); a.multiRoot() && i < len(a.repos) {
				a.repos[i].busy = "同步"
			}
			continue
		case "pop": // pop:<键>:直接展开某个弹层(如 mcp / ctx-cmp / upd / notice)
			a.popup = arg
			if arg == "notice" {
				if a.nt == nil {
					a.nt = &noticeStore{}
				}
				a.nt.panelOn = true
				if len(a.nt.items) == 0 {
					a.noticeWarn("未安装 rust-analyzer", "语言智能暂不可用。", nil)
				}
			}
			continue
		case "upd":
			a.SeedUpdateDemo(arg, 42)
			continue
		case "autonew": // 自动化页的「新建自动化」对话框
			a.screen = ScreenAutomation
			a.openAutoNewDlg()
			continue
		case "sub": // sub:<卡片 id>:在工作区打开子 Agent 标签(空 = 第一张卡)
			a.SetMode(ModeIDE)
			id := arg
			if id == "" {
				for _, it := range a.st.Items {
					if s, ok := it.(*core.SubagentCard); ok {
						id = s.ID
						break
					}
				}
			}
			if id != "" {
				a.OpenWorkspaceTab(vSub, id)
				a.subNoFollow = false
				a.subSt.Y = 1e9
			}
			continue
		case "review": // 打开审查标签;review:empty = 空态
			if arg == "empty" {
				a.reviewDemoEmpty = true
			}
			a.OpenReview()
			continue
		case "wizard": // wizard / wizard:1(前进到第 2 个)/ wizard:done(全部处理完):多仓库逐个提交向导
			a.startCommitWizard()
			if a.cw != nil {
				switch arg {
				case "1":
					a.wizardNext("ok")
				case "done":
					for a.cw != nil && a.cw.finished.IsZero() {
						a.wizardNext("ok")
					}
				}
			}
			continue
		case "wizfail": // 演示:首个仓库提交失败(pre-commit)
			a.ActivateWorkspace("ws-fullstack")
			a.dbgSCM()
			if len(a.repos) > 0 {
				if mem, ok := a.repos[0].repo.(*vcs.Mem); ok {
					mem.SetFail("Commit", "pre-commit 钩子未通过", true)
				}
			}
			a.startCommitWizard()
			continue
		case "expandall": // 多仓「全部展开」SCM 模式
			a.prefs.Set(pScmExpandAll, true)
			a.OpenSCM()
			continue
		case "permcard": // 附加文件夹权限提示卡
			a.openWsDialog("ws-fullstack", "")
			a.wsPerm = wsPermPending{when: "save"}
			a.pd.kind = "ws-perm" // 覆盖 openWsDialog 的 ws
			continue
		case "treemenu": // treemenu:<树路径>:选中并打开「让 Claude…」
			a.SetMode(ModeIDE)
			a.nav, a.sideOpen = "files", true
			a.ide.sel = arg
			a.treeClaudeAsk(arg)
			continue
		}
		if a.gitDebug(name, arg) {
			continue
		}
		if name == "conflict" {
			a.demoConflict()
			continue
		}
		if name == "diff" || name == "staged" {
			k := map[string]string{"diff": dkWork, "staged": dkStaged}[name]
			a.openDiffKind(k, arg, true)
			continue
		}
		if name == "timeline" {
			a.ide.sections["时间线"], a.ide.sections["大纲"] = true, true
			a.nav, a.sideOpen = "files", true
			continue
		}
		if name == "open" {
			a.ide.openFile(arg, true)
			continue
		}
		if name == "pintab" { // pintab:路径1;路径2 固定多个标签(需先 open:)
			for _, p := range strings.Split(arg, ";") {
				p = strings.TrimSpace(p)
				if p == "" {
					continue
				}
				if t := a.ide.tabByPath(p); t != nil {
					a.setTabPinned(t, true)
				}
			}
			continue
		}
		if name == "openeditors" { // openeditors:展开「打开的编辑器」分区
			a.SetMode(ModeIDE)
			a.nav, a.sideOpen = "files", true
			a.ide.sections["打开的编辑器"] = true
			continue
		}
		if name == "goto" {
			a.SetMode(ModeIDE)
			if p, line, col, ok := parsePathLine(arg); ok {
				if p == "" {
					a.gotoLineInCurrent(line, col)
				} else {
					a.openFileAtLine(p, line, col)
				}
			}
			continue
		}
		if name == "gotoline" {
			a.SetMode(ModeIDE)
			a.openGotoLine()
			if arg != "" {
				a.ide.gotoLn.text = arg
			}
			continue
		}
		if name == "treefilter" { // treefilter[:关键字]:展开文件树过滤条并填入关键字
			a.SetMode(ModeIDE)
			a.ide.fltOpen, a.ide.flt = true, arg
			continue
		}
		if name == "multicursor" {
			a.SetMode(ModeIDE)
			a.dbgMultiCursor(tb)
			continue
		}
		if name == "blockcomment" {
			a.SetMode(ModeIDE)
			a.toggleBlockComment()
			continue
		}
		if name == "extmod" {
			a.dbgExtmod(arg)
			continue
		}
		if name == "crumb" {
			a.dbgCrumb(arg)
			continue
		}
		if name == "termrename" {
			a.SetMode(ModeIDE)
			a.ide.bottomOpen, a.ide.bottomTab = true, "term"
			if a.proj != "" && len(a.ide.tps) == 0 {
				a.newTerm()
			}
			a.beginTermRename(a.ide.tpSel)
			continue
		}
		if name == "termtabs" {
			a.SetMode(ModeIDE)
			n := 3
			if arg != "" {
				n = atoi(arg)
			}
			a.ensureTermTabsDemo(n)
			continue
		}
		switch name {
		case "md": // md:src / md:wysiwyg:当前 Markdown 标签的视图
			if tb != nil && tb.buf != nil {
				a.setMdMode(tb, arg == "src")
			}
			continue
		case "wrap": // 切换自动换行
			a.toggleWrap()
			continue
		case "format": // 格式化当前文档
			a.formatDoc()
			continue
		case "ref": // ref:路径:在输入框顶部加一枚引用胶囊(文件夹以 / 结尾),可连写多个用 ; 隔开
			for _, p := range strings.Split(arg, ";") {
				a.draft = appendRefAtoms(a.draft, "@"+p)
			}
			continue
		case "drag": // drag:@路径:对话栏显示「松手插入」可放态
			a.ux.demoHint = arg
			continue
		case "mdedit": // mdedit:在所见即所得里把整篇换成给定 Markdown(用 \n 表示换行),演示「改过」状态
			if tb != nil && tb.buf != nil && isMarkdownFile(tb.path) {
				md := a.mdEnsure(tb)
				if md.doc != nil {
					md.doc.SetText(richtext.ParseMarkdown(strings.ReplaceAll(arg, `\n`, "\n")))
					a.mdWrite(tb, md)
				}
			}
			continue
		case "mdkeep": // mdkeep:载入含段内换行 / 语言标注 / * 列表的样例并留在预览
			if tb != nil && tb.buf != nil && isMarkdownFile(tb.path) {
				sample := "# 快速开始\n\n第一段:设计稿先行。\n\n第二段:先改这一\n段。\n\n第三段:段内硬换行\n仍在本段。\n\n```rust\nfn main() {}\n```\n\n- 用 * 列表\n"
				tb.buf.Set(sample)
				tb.mdMode = mdWysiwyg
				tb.md = nil
				a.mdEnsure(tb)
			}
			continue
		}
		if tb == nil || tb.st == nil {
			continue
		}
		switch name {
		case "cursor":
			tb.st.GoTo(pos(arg))
		case "select":
			from, to, _ := strings.Cut(arg, "-")
			a0, b0 := pos(from), pos(to)
			tb.st.Select(tb.buf.Offset(a0), tb.buf.Offset(b0))
		case "find":
			tb.st.Find(arg)
		case "scroll":
			tb.st.SetScrollY(tb.st.LineTop(max(atoi(arg)-1, 0)))
		case "peek":
			a.openPeek()
		case "rename":
			a.openRename()
		case "acts":
			a.openActs()
		case "inline":
			a.openInline()
		case "refs":
			switch arg {
			case "demo":
				st := a.refsBeginSearch("normalize")
				a.refsFinishLocal(st, a.refsDemoItems("normalize"))
			case "empty":
				st := a.refsBeginSearch("NoSuchSymbol")
				a.refsFinishLocal(st, nil)
			case "fold":
				a.showRefs("normalize")
				a.refsCollapseAll()
			case "loading":
				st := a.refsBeginSearch("Count")
				st.loading = true
			case "err":
				st := a.refsBeginSearch("Count")
				st.err = "语言服务器未响应(演示错误态)"
				st.loading = false
			default:
				if strings.HasPrefix(arg, "sel:") {
					a.showRefs("normalize")
					if r := a.ide.refs; r != nil {
						n := atoi(strings.TrimPrefix(arg, "sel:"))
						r.rebuildRows()
						for i, row := range r.rows {
							if row.kind == refRowHit && row.ix == n {
								r.list.Selected = &i
								k := refItemKey(r.items[n], r.word)
								r.selKey = k
								break
							}
						}
					}
				} else if arg != "" {
					a.showRefs(arg)
				} else {
					a.showRefs(a.cursorWord())
				}
			}
		case "def":
			a.goDef()
		case "hover":
			a.hoverAt(tb, editor.HoverRequest{Offset: tb.buf.Offset(pos(arg))})
		}
	}
}

func atoi(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			break
		}
		n = n*10 + int(r-'0')
	}
	return n
}

// dbgSCM 调试:打开源代码管理并立刻读完仓库状态(截图要在第一帧就有数据)。
func (a *App) dbgSCM() {
	a.OpenSCM()
	if !a.sc.loaded {
		a.scmRefresh()
	}
}
