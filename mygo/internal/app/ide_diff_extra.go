package app

import (
	"strings"
	"unicode"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
	"github.com/ixxxxoooo/minicode/internal/vcs"
)

const pDiffIgnWS = "scm.ignoreWs"

// setupDiffView 配置差异标签的常驻块头、行选与忽略空白。
func (a *App) setupDiffView(tb *ideTab) {
	dv := tb.dv
	if dv == nil {
		return
	}
	// HunkActions=false 时库只走 Annotations,不会调 RenderHunkActions(diffside.go);注解条里复用 hunkBar。
	dv.HunkActions = false
	a.syncDiffAnns(tb)
	dv.RenderHunkActions = func(c *ui.Context, bar ui.Element, act *editor.DiffHunkActions) {
		a.hunkBar(c, bar, act, tb)
	}
	dv.RenderAnnotation = func(c *ui.Context, an editor.DiffAnnotation) {
		key, _ := an.Key.(string)
		idx := atoi(strings.TrimPrefix(key, "h"))
		a.hunkAnnBar(c, tb, idx)
	}
	if tb.dk == dkWork && tb.dv.Editable {
		dv.LineSelection = true
		dv.OnLineSelection = func(r editor.DiffLineRange) { a.diffLineSel(tb, r) }
	}
}

func (a *App) syncDiffAnns(tb *ideTab) {
	dv := tb.dv
	var anns []editor.DiffAnnotation
	for i, h := range dv.Hunks() {
		line := h.NewStart
		if line < 0 {
			line = h.OldStart
		}
		if line < 0 {
			line = 0
		}
		anns = append(anns, editor.DiffAnnotation{Side: editor.DiffNew, Line: line, Height: 26, Key: "h" + itoa(i)})
	}
	dv.Annotations = anns
}

func (a *App) hunkAnnBar(c *ui.Context, tb *ideTab, i int) {
	tk := a.tk
	bar := ui.Row(c).Height(theme.Px(24)).PaddingX(4).Gap(4).AlignItems(ui.Center).Background(tk.Raised).Border(1, tk.Bd2).Radius(tk.RSm())
	act := &editor.DiffHunkActions{Index: i, Hunk: dvHunk(tb.dv, i)}
	// 按钮必须在 bar 的 Children 里创建,否则成了注解容器的兄弟节点:纵向堆叠并盖住代码行。
	bar.Children(func() { a.hunkBar(c, bar, act, tb) })
}

func dvHunk(dv *editor.DiffState, i int) editor.DiffHunk {
	hs := dv.Hunks()
	if i < 0 || i >= len(hs) {
		return editor.DiffHunk{}
	}
	h := hs[i]
	return editor.DiffHunk{OldStart: h.OldStart, OldEnd: h.OldEnd, NewStart: h.NewStart, NewEnd: h.NewEnd}
}

// diffLineSel 在行号选区上显示行级暂存/还原。
func (a *App) diffLineSel(tb *ideTab, r editor.DiffLineRange) {
	a.diffSelTab, a.diffSelRange = tb, r
}

// diffLineBar 在差异标签顶栏下方画行级操作条。
func (a *App) diffLineBar(c *ui.Context, tb *ideTab) {
	if a.diffSelTab != tb || !tb.dv.Editable {
		return
	}
	r := a.diffSelRange
	if r.Start < 0 {
		return
	}
	t := a.tk
	ui.Row(c).Height(theme.Px(26)).PaddingX(12).Gap(6).AlignItems(ui.Center).Background(t.AcSoft).Children(func() {
		ui.Text(c, "已选第 "+itoa(r.Start+1)+"–"+itoa(r.End+1)+" 行").FontSize(theme.FsCap).TextColor(t.AcTx).Grow(1)
		if tb.dk == dkWork {
			kit.Btn(c, "暂存选中行", kit.BtnOpt{Small: true, Icon: "plus"}).OnClick(func() { a.stageLineRange(tb, r) })
		}
		if tb.dk != dkStaged {
			kit.Btn(c, "还原选中行", kit.BtnOpt{Variant: kit.BtnGhost, Small: true, Icon: "rewind"}).OnClick(func() { a.revertLineRange(tb, r) })
		}
		kit.Btn(c, "清除选择", kit.BtnOpt{Variant: kit.BtnGhost, Small: true}).OnClick(func() { a.diffSelTab = nil })
	})
}

func (a *App) stageLineRange(tb *ideTab, r editor.DiffLineRange) {
	repo, rel := a.repoAt(tb.path)
	if repo == nil {
		return
	}
	index, _ := repo.Index(rel)
	src := tb.dv.New.String()
	if r.Side == editor.DiffOld {
		src = tb.dv.Old.String()
	}
	content := vcs.MergeLineRange(index, src, r.Start, r.End)
	a.scmAct("暂存选中行", func(r vcs.Repo) error { return r.StageContent(rel, content) }, func() { a.toast = "已暂存选中行" })
}

func (a *App) revertLineRange(tb *ideTab, r editor.DiffLineRange) {
	nw := vcs.MergeLineRange(tb.dv.New.String(), tb.dv.Old.String(), r.Start, r.End)
	tb.dv.New.Reset(nw)
	a.writeWork(tb)
	a.toast = "已还原选中行"
}

func lineNormWS(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var b strings.Builder
	sp := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !sp {
				b.WriteByte(' ')
				sp = true
			}
			continue
		}
		sp = false
		b.WriteRune(r)
	}
	return b.String()
}

// alignWSForDiff 仅用于比较:空白-only 差异的行在 new 侧对齐成 old 行。
func alignWSForDiff(old, nw string) string {
	ol := strings.Split(old, "\n")
	nl := strings.Split(nw, "\n")
	for i := range nl {
		if i < len(ol) && lineNormWS(ol[i]) == lineNormWS(nl[i]) && ol[i] != nl[i] {
			nl[i] = ol[i]
		}
	}
	return strings.Join(nl, "\n")
}

func (a *App) diffIgnoreWS() bool { return a.prefs.Bool(pDiffIgnWS, false) }

func (a *App) toggleDiffIgnoreWS(tb *ideTab) {
	on := !a.diffIgnoreWS()
	a.prefs.Set(pDiffIgnWS, on)
	a.rebuildDiff(tb)
	log.Info("切换忽略空白差异", "on", on)
}

func (a *App) rebuildDiff(tb *ideTab) {
	if tb == nil || tb.dv == nil {
		return
	}
	old := tb.dv.Old.String()
	work, err := a.ide.fs.Read(tb.path)
	if err != nil {
		work = tb.dv.New.String()
	}
	if a.diffIgnoreWS() {
		work = alignWSForDiff(old, work)
	}
	tb.dv.New.Reset(work)
	a.syncDiffAnns(tb)
}
