package app

import (
	"regexp"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

var draftAtRE = regexp.MustCompile(`@([^\s@]+)`)

// draftRefPaths 从草稿里解析不重复的 @ 引用路径(不含 @ 前缀)。
func draftRefPaths(draft string) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range draftAtRE.FindAllStringSubmatch(draft, -1) {
		if len(m) < 2 {
			continue
		}
		p := strings.TrimSpace(m[1])
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}

// composerRefChips 输入框上方展示 @ 引用附件(与图片附件同一行区域)。
func (a *App) composerRefChips(c *ui.Context) {
	refs := draftRefPaths(a.draft)
	if len(refs) == 0 {
		return
	}
	t := a.tk
	ui.Row(c).Padding(6, 12, 0, 12).Gap(6).Wrap().Children(func() {
		for _, p := range refs {
			p := p
			chip := ui.Row(c).Height(24).PaddingX(4).Gap(4).Radius(t.RSm()).AlignItems(ui.Center).
				Border(1, t.Bd2).Background(t.Fill1).Shrink(0).Cursor(ui.CursorPointer)
			chip.OnClick(func() { a.openPathRef(p) })
			chip.Children(func() {
				name := baseName(strings.TrimSuffix(p, "/"))
				if strings.HasSuffix(p, "/") {
					kit.Ic(c, "folder", kit.S12).Shrink(0)
				} else {
					a.fileIc(c, name)
				}
				ui.Text(c, name).FontSize(theme.FsCap).TextColor(t.Tx2).SingleLine().MaxWidth(160).Shrink(0)
			})
		}
	})
}
