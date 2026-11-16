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

// 拖入 / 选择的文件引用(设计稿 drag-more 的 .ref 胶囊):不写进输入框正文,以胶囊显示在输入框顶部,
// 发送时按「@路径」放在消息最前面。这样输入框里不会同时出现一段 @路径 文字和一枚重复的胶囊。

// addRefs 记下一批引用(「@路径」形式);已有的不重复,返回新增了几个。
func (a *App) addRefs(refs ...string) int {
	n := 0
	for _, r := range refs {
		if r == "" || a.hasRef(r) {
			continue
		}
		a.refs = append(a.refs, r)
		n++
	}
	return n
}

// hasRef 引用是否已经在胶囊里,或已经写在正文里。
func (a *App) hasRef(r string) bool {
	for _, x := range a.refs {
		if x == r {
			return true
		}
	}
	return strings.Contains(" "+strings.ReplaceAll(a.draft, "\n", " ")+" ", " "+r+" ")
}

// removeRef 移除一枚引用胶囊。
func (a *App) removeRef(r string) {
	out := a.refs[:0]
	for _, x := range a.refs {
		if x != r {
			out = append(out, x)
		}
	}
	a.refs = out
}

// foldRefs 发送时把引用胶囊并进正文(放在最前,空格隔开)并清空胶囊;没有胶囊原样返回。
func (a *App) foldRefs(text string) string {
	if len(a.refs) == 0 {
		return text
	}
	head := strings.Join(a.refs, " ")
	a.refs = nil
	if strings.TrimSpace(text) == "" {
		return head
	}
	return head + " " + text
}

// composerRefChips 输入框上方的引用胶囊(与图片附件同一行区域):类型图标 + 文件名 + 移除。
func (a *App) composerRefChips(c *ui.Context) {
	if len(a.refs) == 0 {
		return
	}
	t := a.tk
	ui.Row(c).Padding(8, 12, 0, 12).Gap(6).Wrap().Children(func() {
		for _, r := range a.refs {
			r := r
			p := strings.TrimPrefix(r, "@")
			chip := ui.Row(c).Key("ref-"+p).Height(22).Padding(0, 3, 0, 7).Gap(4).Radius(t.RXs()).AlignItems(ui.Center).
				Background(t.InfoSoft).Shrink(0).Cursor(ui.CursorPointer)
			if chip.Hovered() {
				chip.Background(t.Info.Alpha(.2))
			}
			chip.Tooltip(p)
			chip.OnClick(func() { a.openPathRef(p) })
			chip.Children(func() {
				name := baseName(strings.TrimSuffix(p, "/"))
				if strings.HasSuffix(p, "/") {
					kit.Ic(c, "folder", kit.S12).TextColor(t.Info).Shrink(0)
				} else {
					a.fileIc(c, name)
				}
				ui.Text(c, name).FontSize(theme.FsCap).TextColor(t.Info).SingleLine().MaxWidth(180).Shrink(0)
				kit.Ib(c, "x", kit.IbOpt{Small: true, Tip: "移除引用"}).Size(16, 16).OnClick(func() { a.removeRef(r) })
			})
		}
	})
}
