package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

const splitMin = float32(200)

// splitEditor 向右(1)或向下(2)拆分;已有拆分时切换方向。
func (a *App) splitEditor(dir int8) {
	d := a.ide
	if d.splitDir == dir {
		return
	}
	if d.splitFrac == 0 {
		d.splitFrac = 0.5
	}
	d.splitDir = dir
	d.paneSel = 1
	tb := d.curOn(0)
	if tb != nil && tb.buf != nil && !tb.diff && tb.virt == "" {
		a.openSharedInPane(1, tb)
	}
	uxLog.Info("拆分编辑器组", "dir", dir)
}

func (a *App) mergeSplit() {
	d := a.ide
	if d.splitDir == 0 {
		return
	}
	d.tabs2, d.active2 = nil, ""
	d.splitDir = 0
	d.paneSel = 0
}

// openSharedInPane 在指定组打开与已有标签同路径的文件,共享缓冲区。
func (a *App) openSharedInPane(pane int8, src *ideTab) {
	if src == nil || src.buf == nil {
		return
	}
	d := a.ide
	for _, t := range d.paneTabs(pane) {
		if t.path == src.path && !t.diff {
			*d.paneActive(pane) = t.id
			d.paneSel = pane
			return
		}
	}
	t := &ideTab{id: src.path, path: src.path, preview: false, buf: src.buf, saved: src.saved, eol: src.eol, eolDisk: src.eolDisk, hooked: false}
	t.buf = src.buf
	t.st = &editor.State{Buffer: src.buf, WrapGuides: nil}
	if head, ok := d.headOf(src.path); ok {
		t.st.DiffBase = editor.NewBuffer(head)
		t.headKey = src.headKey
	}
	d.addTabTo(pane, t)
	d.paneSel = pane
}

// openSide 在侧边组打开路径(preview=预览标签)。
func (a *App) openSide(p string, preview bool) {
	if a.ide.splitDir == 0 {
		a.splitEditor(1)
	}
	d := a.ide
	if t := d.tab(p); t != nil && t.buf != nil {
		a.openSharedInPane(1, t)
		return
	}
	d.addTabTo(1, a.mkFileTab(p, preview))
	d.paneSel = 1
}

func (a *App) mkFileTab(p string, preview bool) *ideTab {
	d := a.ide
	t := &ideTab{id: p, path: p, preview: preview}
	if k := viewerKind(p); k != "" {
		t.viewer, t.vw = k, &viewerState{}
		return t
	}
	d.loadText(t)
	return t
}

// wsSplit 画拆分后的两个编辑器组(各 ≥200px)。
func (a *App) wsSplit(c *ui.Context, cur *ideTab) {
	d := a.ide
	if d.splitFrac < 0.2 {
		d.splitFrac = 0.5
	}
	if d.splitFrac > 0.8 {
		d.splitFrac = 0.5
	}
	frac := d.splitFrac
	child := func(pane int8) {
		a.etabsPane(c, pane)
		pcur := d.curOn(pane)
		if pcur == nil {
			a.wsEmpty(c)
			return
		}
		switch {
		case pcur.isVirt():
			a.virtView(c, pcur)
		case pcur.viewer != "":
			a.viewerTab(c, pcur)
		case pcur.err != "":
			a.errView(c, pcur)
		case pcur.diff:
			a.diffTab(c, pcur)
		default:
			if pcur.buf != nil && isMarkdownFile(pcur.path) && !a.mdSource(pcur) {
				if pcur.reveal != nil {
					pcur.mdMode = mdSource
					a.codeView(c, pcur)
				} else {
					a.mdView(c, pcur)
				}
			} else {
				a.codeView(c, pcur)
			}
		}
	}
	if d.splitDir == 2 {
		ui.Column(c).Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).Grow(frac).MinHeight(splitMin).Children(func() { child(0) })
			a.splitHandle(c, false)
			ui.Column(c).Grow(1 - frac).MinHeight(splitMin).Children(func() { child(1) })
		})
	} else {
		ui.Row(c).Grow(1).MinWidth(0).Children(func() {
			ui.Column(c).Grow(frac).MinWidth(splitMin).Children(func() { child(0) })
			a.splitHandle(c, true)
			ui.Column(c).Grow(1 - frac).MinWidth(splitMin).Children(func() { child(1) })
		})
	}
	_ = cur
}

func (a *App) splitHandle(c *ui.Context, vertical bool) {
	t := a.tk
	el := ui.Box(c).Key("split-h").Background(t.Bd1).Cursor(map[bool]ui.Cursor{true: ui.CursorResizeEW, false: ui.CursorResizeNS}[vertical])
	if vertical {
		el.Width(4).FillHeight().Shrink(0)
	} else {
		el.Height(4).FillWidth().Shrink(0)
	}
	if el.Dragging() && vertical && a.wsW > splitMin*2 {
		if x, _, ok := el.PointerPosition(); ok {
			a.ide.splitFrac = clamp32(x/a.wsW, 0.2, 0.8)
		}
	}
}

func clamp32(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// etabsSplitBar 拆分态下顶部的全局条(合并 + 更多)。
func (a *App) etabsSplitBar(c *ui.Context) {
	ui.Row(c).Height(theme.HTab).PaddingX(8).Gap(4).AlignItems(ui.Center).Shrink(0).Children(func() {
		ui.Text(c, "编辑器组已拆分").Grow(1).FontSize(theme.FsCap).TextColor(a.tk.Tx3)
		kit.Ib(c, "merge", kit.IbOpt{Small: true, Tip: "合并拆分"}).OnClick(a.mergeSplit)
		kit.Ib(c, "more", kit.IbOpt{Small: true, Tip: "更多操作"}).Menu(a.tabStripMenu)
	})
}

// etabsPane 在拆分模式下为某一组画标签条。
func (a *App) etabsPane(c *ui.Context, pane int8) {
	d := a.ide
	tabs := d.paneTabs(pane)
	active := *d.paneActive(pane)
	ui.Row(c).Height(theme.HTab - 2).Gap(1).PaddingX(4).AlignItems(ui.Center).Children(func() {
		for _, tb := range tabs {
			if tb.id == active {
				d.paneSel = pane
			}
			a.etabPane(c, tb, pane)
		}
	})
}
