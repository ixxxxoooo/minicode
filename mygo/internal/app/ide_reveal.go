package app

import "strings"

// autoReveal 切换编辑器标签后,在文件树里展开各级目录并选中该文件(设计稿「左栏 · 自动定位」)。
// 只在标签变化时执行一次,之后用户在树里自己点、自己折叠都不再被覆盖;偏好 explorer.autoReveal 可关。
func (a *App) autoReveal() {
	d := a.ide
	tb := d.cur()
	if tb == nil || tb.virt != "" || tb.diff || tb.path == "" || tb.path == d.revealed {
		return
	}
	d.revealed = tb.path
	if !a.prefs.Bool("explorer.autoReveal", true) {
		return
	}
	parts := strings.Split(tb.path, "/")
	for i := 1; i < len(parts); i++ {
		d.open[strings.Join(parts[:i], "/")] = true
	}
	d.sel = tb.path
	uxLog.Info("自动定位当前文件", "path", tb.path)
}
