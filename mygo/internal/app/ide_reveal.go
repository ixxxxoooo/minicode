package app

import "strings"

// autoReveal 切换编辑器标签后,在文件树里展开各级目录并选中该文件(设计稿「左栏 · 自动定位」)。
// 只在标签变化时执行一次,之后用户在树里自己点、自己折叠都不再被覆盖;偏好 explorer.* 可关或限制。
func (a *App) autoReveal() {
	if a.mode != ModeIDE || a.ide == nil {
		return
	}
	d := a.ide
	tb := d.cur()
	if tb == nil || tb.virt != "" || tb.diff || tb.path == "" || tb.path == d.revealed {
		return
	}
	if !a.prefs.Bool("explorer.autoReveal", true) {
		d.revealed = tb.path
		return
	}
	if a.prefs.Bool("explorer.autoRevealWhenVisible", false) {
		if a.NavID() != "files" || !a.sideOpen {
			return
		}
	}
	d.revealed = tb.path
	parts := strings.Split(tb.path, "/")
	for i := 1; i < len(parts); i++ {
		d.open[strings.Join(parts[:i], "/")] = true
	}
	d.sel = tb.path
	if a.prefs.Bool("explorer.autoRevealScroll", true) {
		d.revealTo = tb.path // 展开后还要把这一行滚进视口(下一帧行位置才算得出来)
	}
	uxLog.Info("自动定位当前文件", "path", tb.path, "scroll", d.revealTo != "")
}

// scrollRowIntoView 返回让某一行完全落进视口所需的滚动位置:已可见则原样返回,
// 在上方 / 下方时多留一行余量,避免目标行贴着边缘。
func scrollRowIntoView(y, viewH, rowTop, rowH float32) float32 {
	switch {
	case viewH <= 0:
		return y
	case rowTop < y:
		return max(0, rowTop-rowH)
	case rowTop+rowH > y+viewH:
		return rowTop + rowH*2 - viewH
	}
	return y
}

// treeRevealScroll 在文件树画完后调用:把待定位的行滚进可见范围。
// 行内容位置 = 标题行高 + 行序号 × 行高(树行等高),不依赖上一帧的 Bounds,
// 展开目录的同一帧就能算对;视口高度取自上一帧,第一帧没有就等下一帧。
func (a *App) treeRevealScroll() {
	d := a.ide
	if d.revealTo == "" {
		return
	}
	if d.treeHit < 0 { // 目标行不在树里(根被折叠 / 路径不存在):放弃,别无限等
		d.revealTo = ""
		return
	}
	if d.treeH <= 0 {
		a.frameSoon()
		return
	}
	if d.treeHit >= 0 && len(d.treeLineBuf) > 0 {
		d.treeList.ScrollIntoView(d.treeHit)
	}
	rowH := a.tk.Scheme.RowH
	top := 26 + float32(d.treeHit)*rowH
	d.treeSt.Y = scrollRowIntoView(d.treeSt.Y, d.treeH, top, rowH)
	uxLog.Info("文件树滚到当前文件", "path", d.revealTo, "row", d.treeHit)
	d.revealTo = ""
}
