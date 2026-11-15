package app

import (
	"strings"

	"github.com/egoist/mygo/ui/editor"
	"github.com/egoist/mygo/ui/richtext"
)

// EditorDemo 按「动作:参数」驱动编辑器进入某个状态(cmd/shot 截图用,多个动作用逗号隔开):
// cursor:行:列(1 起)、select:行:列-行:列、find:查询、scroll:行、peek、rename、acts、inline、refs、def、hover:行:列、open:路径、conflict(合并冲突标签)、review(审查标签)、expand:路径、rename:会话id、archived:项目id、scm、syncall、repobusy:N、search:查询、scope:文件夹、pop:弹层键、wizard[:1|done](逐个提交向导,配合 -ws)、md:src|wysiwyg(Markdown 视图)、wrap、format、drag:@路径(对话栏可放态)、mdedit:内容。
func (a *App) EditorDemo(spec string) {
	for _, act := range strings.Split(spec, ",") {
		name, arg, _ := strings.Cut(act, ":")
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
		case "scope": // scope:文件夹名:多根搜索限定范围
			a.setSearchScope(arg)
			a.runSearch(true)
			continue
		case "expand": // expand:<树路径>[;<树路径>…] 展开文件树里的文件夹并切到文件导航
			for _, q := range strings.Split(arg, ";") {
				a.ide.open[q] = true
			}
			a.nav, a.sideOpen = "files", true
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
		case "pop": // pop:<键>:直接展开某个弹层(如 fp-graph-repo、fp-search-scope)
			a.popup = arg
			continue
		case "review": // 打开审查标签
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
		case "drag": // drag:@路径:对话栏显示「松手引用」可放态
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
			a.showRefs(a.cursorWord())
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
