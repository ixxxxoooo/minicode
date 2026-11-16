package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
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
	key := p + "@" + a.headKey()
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
		lg, _ := repo.FileLog(rel, 40)
		bl, err := repo.Blame(rel)
		return func() {
			g.busy = false
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
	end := tb.buf.Offset(editor.Pos{Line: line, Col: len([]rune(tb.buf.Line(line)))})
	key := tb.path + ":" + itoa(line) + ":" + txt + ":" + itoa(int(tb.buf.Version()))
	if tb.blameKey == key {
		return
	}
	tb.blameKey, tb.blameShown = key, true
	tb.st.SetInlays([]editor.Inlay{{Offset: end, Text: txt, Trailing: true, Ellipsis: "…"}})
}

// timelineRows 是左栏「时间线」:当前文件的提交历史,点一条在编辑区开该次提交里这个文件的差异。
func (a *App) timelineRows(c *ui.Context) {
	t := a.tk
	tb := a.ide.cur()
	if tb == nil || tb.diff || tb.virt != "" {
		ui.Text(c, "打开一个文件后显示它的历史").FontSize(theme.FsCap).TextColor(t.Tx4).Margin(2, 8, 6, 28)
		return
	}
	g := a.gitInfoFor(tb.path)
	if g == nil {
		ui.Text(c, "不在 Git 仓库里").FontSize(theme.FsCap).TextColor(t.Tx4).Margin(2, 8, 6, 28)
		return
	}
	// 顶部:还没提交的本地更改(含未保存的编辑),点开是 HEAD ↔ 工作区
	if a.ide.git[tb.path] != "" || tb.dirty() {
		r := ui.ButtonBase(c).Key("tl-local").Height(34).PaddingX(8).Gap(8).AlignItems(ui.Center).Radius(t.RSm()).Cursor(ui.CursorPointer).Margin(0, 4)
		if r.Hovered() {
			r.Background(t.Fill1)
		}
		if r.Clicked() {
			a.openDiffKind(dkHead, tb.path, false)
		}
		if r.DoubleClicked() {
			a.openDiffKind(dkHead, tb.path, true)
		}
		r.Children(func() {
			ui.Box(c).Size(7, 7).Radius(4).Border(1.5, t.Warn).Shrink(0)
			ui.Column(c).Grow(1).MinWidth(0).Children(func() {
				ui.Text(c, "本地更改").SingleLine().FontSize(theme.FsCap)
				ui.Text(c, map[bool]string{true: "含未保存的修改", false: "未提交"}[tb.dirty()]).SingleLine().FontSize(theme.FsMicro).TextColor(t.Warn)
			})
		})
	}
	if len(g.log) == 0 {
		msg := "这个文件还没有提交历史"
		if g.busy {
			msg = "读取中…"
		}
		ui.Text(c, msg).FontSize(theme.FsCap).TextColor(t.Tx4).Margin(2, 8, 6, 28)
		return
	}
	for _, cm := range g.log {
		cm := cm
		r := ui.ButtonBase(c).Key("tl-"+cm.Hash).Height(34).PaddingX(8).Gap(8).AlignItems(ui.Center).Radius(t.RSm()).Cursor(ui.CursorPointer).Margin(0, 4)
		if r.Hovered() {
			r.Background(t.Fill1)
		}
		if r.Clicked() {
			a.scmOpenCommitFile(cm, tb.path, false)
		}
		if r.DoubleClicked() {
			a.scmOpenCommitFile(cm, tb.path, true)
		}
		r.ContextMenu(func(m *ui.Menu) {
			if menuItem(m, "打开该次提交里的改动").Chosen() {
				a.scmOpenCommitFile(cm, tb.path, true)
			}
			if menuItem(m, "在图谱中查看").Chosen() {
				a.openGraph()
				a.graphTab.sel = cm.Hash
				a.scmCommitFiles(cm.Hash)
			}
			m.Separator()
			if menuItem(m, "复制哈希").Chosen() {
				a.scmCommitOp(cm, "copy")
			}
		})
		r.Children(func() {
			ui.Box(c).Size(7, 7).Radius(4).Background(t.Ac).Shrink(0)
			ui.Column(c).Grow(1).MinWidth(0).Children(func() {
				ui.Text(c, cm.Subject).SingleLine().FontSize(theme.FsCap)
				ui.Text(c, cm.Author+" · "+cm.When).SingleLine().FontSize(theme.FsMicro).TextColor(t.Tx4)
			})
			ui.Text(c, cm.Short()).Font("monospace").FontSize(theme.FsMicro).TextColor(t.Tx4).Shrink(0)
		})
	}
}
