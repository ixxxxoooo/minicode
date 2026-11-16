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

// wsSplit 画拆分后的两个编辑器组(各 ≥200px)。每个组 = 自己的标签条 + 面包屑 + 内容,
// 组容器从 0 基准分配空间并裁切(Basis(0) + MinHeight(0) + Clip),内容再长也不会越出组去压住别的组或标签条。
func (a *App) wsSplit(c *ui.Context, cur *ideTab) {
	d := a.ide
	if d.splitFrac < 0.2 || d.splitFrac > 0.8 {
		d.splitFrac = 0.5
	}
	frac := d.splitFrac
	focusPane := func(pane int8) func(ui.InputEvent) { // 在哪个组里按下 / 打字,哪个组就是焦点组(捕获阶段,不吃事件)
		return func(ev ui.InputEvent) {
			switch ev.Kind {
			case ui.InputPointerDown, ui.InputKeyDown, ui.InputText:
				d.paneSel = pane
			}
		}
	}
	child := func(pane int8) {
		a.etabsPane(c, pane)
		pcur := d.curOn(pane)
		if pcur == nil {
			a.wsEmpty(c)
			return
		}
		if !pcur.isVirt() && !pcur.diff {
			a.crumbs(c, pcur) // 面包屑属于各自的组(以前画在整个拆分区上方,只跟着焦点组走)
		}
		// 内容区:占满标签条与面包屑以外的高度并裁切;顶部的「预览 / 源码」等浮条是相对它定位的
		ui.Column(c).Grow(1).Basis(0).MinHeight(0).MinWidth(0).Clip().Children(func() {
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
		})
	}
	pane := func(i int8, g float32, vertical bool) {
		col := ui.Column(c).Key("split-pane-" + itoa(int(i))).Grow(g).Basis(0).MinWidth(0).MinHeight(0).Clip().ObserveInput(focusPane(i))
		if vertical {
			col.MinWidth(splitMin)
		} else {
			col.MinHeight(splitMin)
		}
		col.Children(func() { child(i) })
	}
	if d.splitDir == 2 {
		box := ui.Column(c).Key("split-box").Grow(1).MinHeight(0).MinWidth(0).AlignItems(ui.Stretch)
		if b := box.Bounds(); b.H > 0 {
			a.splitBox = b
		}
		box.Children(func() {
			pane(0, frac, false)
			a.splitHandle(c, false)
			pane(1, 1-frac, false)
		})
	} else {
		box := ui.Row(c).Key("split-box").Grow(1).MinWidth(0).MinHeight(0).AlignItems(ui.Stretch) // 不拉伸的话两个组只有内容那么高,居中悬在半空
		if b := box.Bounds(); b.W > 0 {
			a.splitBox = b
		}
		box.Children(func() {
			pane(0, frac, true)
			a.splitHandle(c, true)
			pane(1, 1-frac, true)
		})
	}
	_ = cur
}

// splitHandle 是两个组之间的分隔线(左右拆分拖横向,上下拆分拖纵向),双击还原 50%。
// 外观与其它分隔线一致:平时只是 1px 描边线,命中区是 7px 的透明带,悬停 / 拖动时亮出 2px 强调色线。
// 拖动按「每帧的位移 ÷ 拆分区尺寸」累加到比例上,与分隔线自己所在的坐标系无关。
func (a *App) splitHandle(c *ui.Context, vertical bool) {
	t := a.tk
	line := ui.Box(c).Key("split-line").Background(t.Bd1).Shrink(0)
	if vertical {
		line.Width(1).FillHeight()
	} else {
		line.Height(1).FillWidth()
	}
	line.Children(func() {
		h := ui.Box(c).Key("split-h").Absolute().Cursor(map[bool]ui.Cursor{true: ui.CursorResizeEW, false: ui.CursorResizeNS}[vertical])
		if vertical {
			h.Top(0).Bottom(0).Left(-3).Width(7)
		} else {
			h.Left(0).Right(0).Top(-3).Height(7)
		}
		dx, dy, dragging := h.Dragged()
		if dragging {
			total, delta := a.splitBox.W, dx
			if !vertical {
				total, delta = a.splitBox.H, dy
			}
			if total > splitMin*2 && delta != 0 {
				a.ide.splitFrac = clamp32(a.ide.splitFrac+delta/total, splitMin/total, 1-splitMin/total)
			}
		}
		if h.DoubleClicked() {
			a.ide.splitFrac = 0.5
		}
		lit := dragging || h.Hovered()
		h.Children(func() {
			ln := ui.Box(c).Background(t.Ac).Opacity(h.AnimateTo("lit", map[bool]float32{true: .85, false: 0}[lit], uiFade))
			if vertical {
				ln.Absolute().Top(0).Bottom(0).Left(2.5).Width(2)
			} else {
				ln.Absolute().Left(0).Right(0).Top(2.5).Height(2)
			}
		})
	})
}

// etabsSplitBar 拆分态下顶部的全局条(合并 + 更多)。
func (a *App) etabsSplitBar(c *ui.Context) {
	ui.Row(c).Height(theme.HTab).PaddingX(8).Gap(4).AlignItems(ui.Center).Shrink(0).Children(func() {
		ui.Text(c, "编辑器组已拆分").Grow(1).FontSize(theme.FsCap).TextColor(a.tk.Tx3)
		kit.Ib(c, "merge", kit.IbOpt{Small: true, Tip: "合并拆分"}).OnClick(a.mergeSplit)
		kit.Ib(c, "more", kit.IbOpt{Small: true, Tip: "更多操作"}).Menu(a.tabStripMenu)
	})
}

// etabsPane 在拆分模式下为某一组画标签条:标签 + 右侧「关闭此组」。条自己是 Shrink(0) 的固定高度,
// 横向放不下时裁切(不再溢出盖住下面的内容)。
func (a *App) etabsPane(c *ui.Context, pane int8) {
	t := a.tk
	d := a.ide
	tabs := d.paneTabs(pane)
	bar := ui.Row(c).Key("etabs-pane-" + itoa(int(pane))).Height(theme.HTab).Shrink(0).Gap(1).PaddingX(4).AlignItems(ui.Center).Clip()
	if t.Scheme.ID == "a" {
		bar.Background(t.Chrome).BorderWidth(0, 0, 1, 0).BorderColor(t.Bd2)
	}
	bar.Children(func() {
		// 注意:这里不能因为「这是该组的当前标签」就改 d.paneSel——两个组各有一个当前标签,
		// 每帧后画的次组都会把焦点组抢成 1,保存 / ⌘W / 状态栏 / 搜索定位全落到次组上。
		// 焦点组只由点击标签、点击组里的任意位置(wsSplit 的 ObserveInput)和键盘输入决定。
		ui.Row(c).Gap(1).AlignItems(ui.Center).Grow(1).MinWidth(0).Clip().Children(func() {
			for _, tb := range tabs {
				a.etabPane(c, tb, pane)
			}
		})
		kit.Ib(c, "x", kit.IbOpt{Small: true, Tip: "关闭这个编辑器组(合并拆分)"}).OnClick(func() { a.closePane(pane) })
	})
}

// closePane 关掉一个编辑器组:留下另一组。关闭的组里有只在这里开着、带未保存修改的标签时先确认。
func (a *App) closePane(pane int8) {
	d := a.ide
	lose := false
	for _, tb := range d.paneTabs(pane) {
		if a.closeLoses(tb) {
			lose = true
		}
	}
	do := func() {
		if pane == 0 { // 关主组:把次组的标签换过来再合并
			d.tabs, d.active = d.tabs2, d.active2
		}
		a.mergeSplit()
	}
	if lose {
		a.confirmDlg("不保存就关闭这个组?", "这个编辑器组里有未保存的修改,关闭后这些修改会丢失。", "不保存并关闭", true, do)
		return
	}
	do()
}

// closeTabOnPane 关某一组里的标签(没拆分时就是普通关闭)。以前标签上的 × 一律按主组处理,次组的标签关不掉。
func (a *App) closeTabOnPane(id string, pane int8) {
	d := a.ide
	switch {
	case d.splitDir == 0:
		a.closeWsTab(id)
	case pane == 1:
		d.closeOnPane(id, &d.tabs2, &d.active2)
	default:
		d.closeOnPane(id, &d.tabs, &d.active)
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
