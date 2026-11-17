package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
	"github.com/ixxxxoooo/minicode/internal/vcs"
	"github.com/ixxxxoooo/minicode/internal/workspace"
)

// 文件级 Git 信息:行内追溯(blame)与左栏「时间线」(文件历史)。
// 都按「文件 + 仓库 HEAD」缓存,在后台读,HEAD 变了(提交 / 切分支)就重读。

// gitInfo 是某个文件的追溯与历史缓存。
type gitInfo struct {
	key   string // 文件 + HEAD 哈希
	busy  bool
	err   error
	blame []vcs.BlameLine
	log   []vcs.Commit
}

func (a *App) headKey() string {
	if len(a.sc.log) > 0 {
		return a.sc.log[0].Hash
	}
	return ""
}

// gitInfoFor 取文件的追溯与历史;没有或过期就在后台读(读完后下一帧生效)。
func (a *App) gitInfoFor(p string) *gitInfo {
	if workspace.IsExternalPath(p) {
		return nil
	}
	repo, rel := a.repoAt(p)
	if repo == nil || p == "" {
		return nil
	}
	if a.gi == nil {
		a.gi = map[string]*gitInfo{}
	}
	lim := a.gitInfoLimit(p)
	key := p + "@" + a.headKey() + "#" + itoa(lim)
	g := a.gi[p]
	if g != nil && (g.key == key || g.busy) {
		return g
	}
	if g == nil {
		g = &gitInfo{}
		a.gi[p] = g
	}
	g.key, g.busy = key, true
	a.bg("git.file", func() func() {
		lg, lerr := repo.FileLog(rel, lim)
		bl, err := repo.Blame(rel)
		return func() {
			g.busy, g.err = false, lerr
			g.log = lg
			if err != nil || len(bl) == 0 {
				bl = nil // 没有追溯数据(演示仓库):由 blameLine 用历史兜底
			}
			g.blame = bl
		}
	})
	return g
}

// blameLine 给出第 i 行(0 起)的追溯。HEAD 与工作区不同的行(gitMarks)一律是「你 · 未提交」;
// 其余行取追溯数据,没有追溯数据(演示仓库)就归到这个文件最近的一次提交。
func (a *App) blameLine(tb *ideTab, g *gitInfo, i int) (vcs.BlameLine, bool) {
	if mm := tb.mm; mm != nil && mm.git != nil {
		if _, ok := mm.git[i]; ok {
			return vcs.BlameLine{Author: "你", When: "未提交", Summary: "未提交的改动", Uncommitted: true}, true
		}
	}
	if i >= 0 && i < len(g.blame) {
		return g.blame[i], true
	}
	if g.blame == nil && len(g.log) > 0 {
		c := g.log[0]
		return vcs.BlameLine{Hash: c.Hash, Author: c.Author, When: c.When, Summary: c.Subject}, true
	}
	return vcs.BlameLine{}, false
}

// blameInlay 在光标所在行末尾显示「作者 · 时间 · 摘要」(淡色、不可选)。
func (a *App) blameInlay(tb *ideTab) {
	if tb.st == nil || tb.buf == nil || tb.diff || tb.virt != "" {
		return
	}
	if !a.prefs.Bool(pBlameInline, true) {
		if tb.blameShown {
			tb.st.SetInlays(nil)
			tb.blameShown = false
		}
		return
	}
	g := a.gitInfoFor(tb.path)
	if g == nil {
		return
	}
	line := tb.st.CursorPos().Line
	bl, ok := a.blameLine(tb, g, line)
	if !ok {
		return
	}
	txt := "    " + bl.Author + " · " + bl.When + " · " + bl.Summary
	// Pos.Col 是字节列:这里必须用字节数。按字符数算时,含中文的行(每字 3 字节)会把追溯文字插到行中间
	end := tb.buf.Offset(editor.Pos{Line: line, Col: len(tb.buf.Line(line))})
	key := tb.path + ":" + itoa(line) + ":" + txt + ":" + itoa(int(tb.buf.Version()))
	if tb.blameKey == key {
		return
	}
	tb.blameKey, tb.blameShown = key, true
	tb.st.SetInlays([]editor.Inlay{{Offset: end, Text: txt, Trailing: true, Ellipsis: "…"}})
}

// timelineHeaderTools 时间线标题行右侧:刷新 / 来源(仅 Git)。
func (a *App) timelineHeaderTools(c *ui.Context) {
	p := a.timelineSubject()
	flt := kit.Ib(c, "filter", kit.IbOpt{Small: true, Tip: "来源: Git 历史(本地历史尚未接入)"})
	flt.Opacity(0.45).Cursor(ui.CursorDefault)
	kit.Ib(c, "refresh", kit.IbOpt{Small: true, Tip: "刷新"}).OnClick(func() {
		if p != "" {
			a.timelineReload(p)
			diag.L("ide").Info("刷新时间线", "path", p)
		}
	})
}

func (a *App) timelineStep(d int) {
	p := a.timelineSubject()
	if p == "" {
		return
	}
	g := a.gitInfoFor(p)
	if g == nil {
		return
	}
	n := len(g.log)
	if a.timelineHasLocal() {
		n++
	}
	if n == 0 {
		return
	}
	ix := a.ide.tlIx + d
	if ix < -1 {
		ix = -1
	}
	if ix >= n {
		ix = n - 1
	}
	a.ide.tlIx = ix
}

func (a *App) timelineHasLocal() bool {
	p := a.timelineSubject()
	if p == "" {
		return false
	}
	tb := a.ide.tab(p)
	if tb == nil {
		tb = a.ide.cur()
	}
	if tb == nil || tb.path != p {
		if t := a.ide.cur(); t != nil && t.diff && t.path == p {
			tb = t
		}
	}
	if tb != nil && (a.ide.git[p] != "" || (!tb.diff && tb.dirty())) {
		return true
	}
	if t := a.ide.cur(); t != nil && t.diff && t.path == p && t.dk == dkHead {
		return true
	}
	return a.ide.git[p] != ""
}

// timelineRows 是左栏「时间线」:跟随活动编辑器对应文件;单击预览对比、双击固定。
func (a *App) timelineRows(c *ui.Context) {
	t := a.tk
	p := a.timelineSubject()
	if p == "" {
		ui.Text(c, "打开一个文件后显示它的历史").FontSize(theme.FsCap).TextColor(t.Tx4).Margin(2, 8, 6, 28)
		return
	}
	if !a.timelineFileGit(p) {
		ui.Text(c, "不在 Git 仓库里").FontSize(theme.FsCap).TextColor(t.Tx4).Margin(2, 8, 6, 28)
		return
	}
	g := a.gitInfoFor(p)
	if g == nil {
		ui.Text(c, "不在 Git 仓库里").FontSize(theme.FsCap).TextColor(t.Tx4).Margin(2, 8, 6, 28)
		return
	}
	pick := a.timelinePickHash()
	untracked := gitInfoUntracked(p, a.ide.git)
	rowIx := 0
	// 本地更改(工作区 / HEAD 对比)
	if a.timelineHasLocal() {
		on := pick == "__local__" || (a.ide.tlIx == 0 && pick == "")
		r := kit.RowEl(c, on).Key("tl-local").Height(theme.Px(34)).PaddingX(8).Gap(8).AlignItems(ui.Center).Radius(t.RSm()).Margin(0, 4)
		if r.Clicked() {
			a.openDiffKind(dkHead, p, false)
			a.ide.tlIx = 0
		}
		if r.DoubleClicked() {
			a.openDiffKind(dkHead, p, true)
		}
		r.Children(func() {
			if on {
				kit.SelBar(c)
			}
			ui.Box(c).Size(7, 7).Radius(4).Border(1.5, t.Warn).Shrink(0)
			ui.Column(c).Grow(1).MinWidth(0).Children(func() {
				ui.Text(c, "本地更改").SingleLine().FontSize(theme.FsCap)
				tb := a.ide.tab(p)
				dirty := tb != nil && tb.dirty()
				ui.Text(c, map[bool]string{true: "含未保存的修改", false: "未提交"}[dirty]).SingleLine().FontSize(theme.FsMicro).TextColor(t.Warn)
			})
		})
		rowIx++
	}
	if msg := timelineErrText(p, g, untracked); msg != "" && len(g.log) == 0 {
		ui.Text(c, msg).FontSize(theme.FsCap).TextColor(t.Tx4).Margin(2, 8, 6, 28)
		return
	}
	for _, cm := range g.log {
		cm, ix := cm, rowIx
		rowIx++
		on := pick == cm.Hash || a.ide.tlIx == ix
		r := kit.RowEl(c, on).Key("tl-"+cm.Hash).Height(theme.Px(34)).PaddingX(8).Gap(8).AlignItems(ui.Center).Radius(t.RSm()).Margin(0, 4)
		if r.Clicked() {
			a.openTimelineCommit(cm, p, false)
			a.ide.tlIx = ix
		}
		if r.DoubleClicked() {
			a.openTimelineCommit(cm, p, true)
			if t := a.ide.tab(timelineDiffID(p)); t != nil {
				t.preview = false
			}
		}
		r.ContextMenu(func(m *ui.Menu) {
			if menuItem(m, "与上一版本比较").Chosen() {
				a.openTimelineVsParent(cm, p, false)
			}
			if menuItem(m, "与工作区比较").Chosen() {
				a.openTimelineVsWork(cm, p, false)
			}
			m.Separator()
			if menuItem(m, "在源代码管理中查看").Chosen() {
				a.timelineViewInSCM(cm.Hash)
			}
			if menuItem(m, "在新标签页中打开").Chosen() {
				a.scmOpenCommitFile(cm, p, true)
			}
			m.Separator()
			if menuItem(m, "复制提交哈希").Chosen() {
				a.scmCommitOp(cm, "copy")
			}
		})
		r.Children(func() {
			if on {
				kit.SelBar(c)
			}
			ui.Box(c).Size(7, 7).Radius(4).Background(t.Ac).Shrink(0)
			ui.Column(c).Grow(1).MinWidth(0).Children(func() {
				ui.Text(c, cm.Subject).SingleLine().FontSize(theme.FsCap)
				ui.Text(c, cm.Author+" · "+cm.When).SingleLine().FontSize(theme.FsMicro).TextColor(t.Tx4)
			})
			ui.Text(c, cm.Short()).Font("monospace").FontSize(theme.FsMicro).TextColor(t.Tx4).Shrink(0)
		})
	}
	if g.busy && len(g.log) > 0 {
		ui.Text(c, "读取中…").FontSize(theme.FsMicro).TextColor(t.Tx4).Margin(4, 8, 4, 28)
	}
	if len(g.log) >= a.gitInfoLimit(p)-1 && !g.busy {
		more := ui.ButtonBase(c).Key("tl-more").Height(theme.Px(28)).PaddingX(8).Justify(ui.Center).TextColor(t.Ac).FontSize(theme.FsCap).Cursor(ui.CursorPointer).Margin(4, 4, 6, 4)
		if more.Clicked() {
			a.timelineLoadMore(p)
		}
		more.Children(func() {
			ui.Text(c, "加载更多…")
		})
	}
}
