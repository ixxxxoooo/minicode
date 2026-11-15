package app

import (
	"path"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
	"github.com/ixxxxoooo/minicode/internal/vcs"
)

// 差异标签:对比的三种来源与块级操作。
//
//	head   HEAD ↔ 工作区      会话流改动卡的「在工作区打开差异」;只读历史对照 + 还原
//	work   暂存区 ↔ 工作区    SCM「更改」里的文件;块头:暂存此块 / 还原此块 / 复制补丁
//	staged HEAD ↔ 暂存区      SCM「暂存的更改」里的文件;块头:取消暂存此块 / 复制补丁
//
// 暂存 / 取消暂存最终都落到 vcs.Repo.StageContent(把完整内容写成暂存区里的版本),
// 所以「只暂存这一块」与 git add -p 等价。工作区一侧是可编辑的:改了按 ⌘S 写回文件。

const (
	dkHead   = "head"
	dkWork   = "work"
	dkStaged = "staged"
)

// diffID 差异标签的 id。
func diffID(kind, p string) string {
	switch kind {
	case dkWork:
		return "diff:wt:" + p
	case dkStaged:
		return "diff:idx:" + p
	}
	return "diff:" + p
}

// openDiffKind 打开某个文件的差异标签(pin=false 为预览标签)。
func (a *App) openDiffKind(kind, p string, pin bool) {
	d := a.ide
	id := diffID(kind, p)
	if t := d.tab(id); t != nil {
		if pin {
			t.preview = false
		}
		d.active, d.sel = id, p
		return
	}
	head := func() string { s, _ := d.headOf(p); return s }
	work := func() string { s, _ := d.fs.Read(p); return s }
	index := func() string {
		if a.repo != nil {
			if s, ok := a.repo.Index(p); ok {
				return s
			}
		}
		return head()
	}
	var old, nw, label string
	switch kind {
	case dkWork:
		old, nw, label = index(), work(), "暂存区 ↔ 工作区"
	case dkStaged:
		old, nw, label = head(), index(), "HEAD ↔ 暂存区"
		if nw == old { // 演示仓库没有真正的暂存区内容:用工作区的版本代替(真实 git 里暂存的改动不会与 HEAD 相同)
			nw = work()
		}
	default:
		old, nw, label = head(), work(), "HEAD ↔ 工作区"
	}
	d.addDiffTab(id, p, label, old, nw, pin)
	if t := d.tab(id); t != nil {
		t.dk = kind
		t.dv.Editable = kind != dkStaged
		a.configureDiff(t)
	}
}

// configureDiff 给差异状态接上块头动作与词级高亮(每个差异标签一次)。
func (a *App) configureDiff(t *ideTab) {
	dv := t.dv
	dv.HunkActions = true
	dv.Inline = editor.DiffInlineWord
	dv.Indicators = editor.DiffBars
	dv.RenderHunkActions = func(c *ui.Context, bar ui.Element, act *editor.DiffHunkActions) {
		a.hunkBar(c, bar, act, t)
	}
}

// hunkBar 是改动块头上的动作:暂存此块 · 还原此块 · 复制补丁(暂存区视图是:取消暂存此块)。
func (a *App) hunkBar(c *ui.Context, bar ui.Element, act *editor.DiffHunkActions, t *ideTab) {
	tk := a.tk
	bar.Row().Gap(4)
	btn := func(key, icon, text string, danger bool, run func()) {
		b := ui.ButtonBase(c).Key("hk-"+key).Height(20).PaddingX(7).Gap(4).Radius(tk.RSm()).AlignItems(ui.Center).Background(tk.Raised).Border(1, tk.Bd2).
			Cursor(ui.CursorPointer).FontSize(theme.FsMicro).TextColor(tk.Tx2)
		if b.Hovered() {
			b.Background(tk.Fill1)
			if danger {
				b.TextColor(tk.Err)
			}
		}
		if b.Clicked() {
			run()
		}
		b.Children(func() {
			kit.Ic(c, icon, kit.S12)
			ui.Text(c, text)
		})
	}
	if t.dk == dkWork && a.repo != nil {
		btn("stage", "plus", "暂存此块", false, func() { a.stageHunk(t, act.Index) })
	}
	if t.dk == dkStaged {
		btn("unstage", "undo", "取消暂存此块", false, func() { a.unstageHunk(t, act.Index) })
	} else {
		btn("revert", "rewind", "还原此块", true, func() { a.revertDiffHunk(t, act.Index) })
	}
	btn("patch", "copy", "复制补丁", false, func() { c.WriteClipboard(a.hunkPatch(t, act.Hunk)) })
}

// stageHunk 只暂存这一块:Keep 之后 Old 就是「暂存区 + 这一块」,整份写成暂存区的版本。
func (a *App) stageHunk(t *ideTab, i int) {
	t.dv.Accept(i)
	content, p := t.dv.Old.String(), t.path
	a.scmAct("暂存此块", func(r vcs.Repo) error { return r.StageContent(p, content) }, func() { a.toast = "已暂存这一块" })
}

// unstageHunk 取消暂存这一块:把这块在暂存区一侧还原,整份写回暂存区。
func (a *App) unstageHunk(t *ideTab, i int) {
	t.dv.Revert(i)
	content, p := t.dv.New.String(), t.path
	a.scmAct("取消暂存此块", func(r vcs.Repo) error { return r.StageContent(p, content) }, func() { a.toast = "已取消暂存这一块" })
}

// revertDiffHunk 还原这一块:把旧行放回工作区一侧并写回文件。
func (a *App) revertDiffHunk(t *ideTab, i int) {
	t.dv.Revert(i)
	a.writeWork(t)
	a.toast = "已还原这一块"
}

// writeWork 把差异标签里工作区一侧的内容写回文件,并让已打开的同一文件同步。
func (a *App) writeWork(t *ideTab) {
	d := a.ide
	src := t.dv.New.String()
	if err := d.fs.Write(t.path, src); err != nil {
		a.toast = "写入 " + t.path + " 失败:" + err.Error()
		return
	}
	d.invalidate()
	if ft := d.tab(t.path); ft != nil && ft.buf != nil && !ft.dirty() {
		ft.buf.Reset(src)
		ft.saved = src
		ft.mm = nil
	}
	a.sc.last = a.sc.last.Add(-scmRefreshEvery) // 让 SCM 尽快重读
}

// hunkPatch 把一块写成统一格式的补丁文本。
func (a *App) hunkPatch(t *ideTab, h editor.DiffHunk) string {
	oldL, newL := strings.Split(t.dv.Old.String(), "\n"), strings.Split(t.dv.New.String(), "\n")
	var b strings.Builder
	b.WriteString("--- a/" + t.path + "\n+++ b/" + t.path + "\n")
	b.WriteString("@@ -" + itoa(h.OldStart+1) + "," + itoa(h.OldEnd-h.OldStart) + " +" + itoa(h.NewStart+1) + "," + itoa(h.NewEnd-h.NewStart) + " @@\n")
	for i := h.OldStart; i < h.OldEnd && i < len(oldL); i++ {
		b.WriteString("-" + oldL[i] + "\n")
	}
	for i := h.NewStart; i < h.NewEnd && i < len(newL); i++ {
		b.WriteString("+" + newL[i] + "\n")
	}
	return b.String()
}

// diffTab 是差异标签的内容:顶栏(文件 / 来源 / 增删 / 上下一处 / 视图切换)+ 差异正文。
func (a *App) diffTab(c *ui.Context, tb *ideTab) {
	t := a.tk
	dv := tb.dv
	dv.Theme = a.editorTheme()
	dv.Font = ui.Font{Family: "monospace", Size: 12}
	narrow := a.wsNarrow()
	if narrow { // 工作区 < 520:强制统一视图
		dv.Layout = editor.DiffUnified
	}
	adds, dels := dv.Stats()
	hunks := dv.Hunks()
	ui.Row(c).Height(30).Shrink(0).PaddingX(12).Gap(8).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(t.Bd1).Children(func() {
		kit.Ic(c, "diff", kit.S12)
		ui.Text(c, path.Base(tb.path)).FontWeight(600).SingleLine().Shrink(0)
		ui.Text(c, tb.diffLabel()).TextColor(t.Tx3).SingleLine()
		kit.Tag(c, "+"+itoa(adds)+" −"+itoa(dels), kit.KOk)
		if tb.dv.Editable {
			kit.Tag(c, "可编辑", kit.KAc)
		}
		ui.Spacer(c).Grow(1)
		if len(hunks) > 0 {
			cur, _ := dv.CurrentHunk()
			ui.Text(c, itoa(cur+1)+" / "+itoa(len(hunks))).FontSize(theme.FsMicro).TextColor(t.Tx4).NoWrap()
		}
		if kit.Ib(c, "chevU", kit.IbOpt{Small: true, Tip: "上一处差异"}).Clicked() && len(hunks) > 0 {
			cur, _ := dv.CurrentHunk()
			dv.GoToHunk((cur - 1 + len(hunks)) % len(hunks))
		}
		if kit.Ib(c, "chevD", kit.IbOpt{Small: true, Tip: "下一处差异"}).Clicked() && len(hunks) > 0 {
			cur, _ := dv.CurrentHunk()
			dv.GoToHunk((cur + 1) % len(hunks))
		}
		if !narrow {
			items := []kit.SegItem{{Text: "左右"}, {Text: "统一"}}
			on := 0
			if dv.Layout == editor.DiffUnified {
				on = 1
			}
			if i := kit.Seg(c, items, on, kit.SegOpt{Key: "dvseg"}); i >= 0 {
				if i == 0 {
					dv.Layout = editor.DiffSplit
				} else {
					dv.Layout = editor.DiffUnified
				}
			}
		}
		kit.Btn(c, "打开文件", kit.BtnOpt{Variant: kit.BtnGhost, Small: true, Icon: "ext"}).OnClick(func() {
			a.jumpTo(tb.path, 0, 0, 0)
		})
	})
	editor.DiffView(c.Key("dv-"+tb.id), dv).Grow(1).MinHeight(0).FillWidth()
}
