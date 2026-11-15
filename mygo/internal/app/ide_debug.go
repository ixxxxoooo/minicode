package app

import (
	"strings"

	"github.com/egoist/mygo/ui/editor"
)

// EditorDemo 按「动作:参数」驱动编辑器进入某个状态(cmd/shot 截图用,多个动作用逗号隔开):
// cursor:行:列(1 起)、select:行:列-行:列、find:查询、scroll:行、peek、rename、acts、inline、refs、def、hover:行:列、open:路径。
func (a *App) EditorDemo(spec string) {
	for _, act := range strings.Split(spec, ",") {
		name, arg, _ := strings.Cut(act, ":")
		tb := a.ide.cur()
		pos := func(s string) editor.Pos {
			var l, c int
			parts := strings.Split(s, ":")
			if len(parts) == 2 {
				l, c = atoi(parts[0]), atoi(parts[1])
			}
			return editor.Pos{Line: max(l-1, 0), Col: max(c-1, 0)}
		}
		if name == "open" {
			a.ide.openFile(arg, true)
			continue
		}
		if tb == nil || tb.st == nil {
			continue
		}
		switch name {
		case "cursor":
			tb.st.GoTo(pos(arg))
		case "select":
			from, to, _ := strings.Cut(arg, "-")
			a0, b0 := pos(from), pos(to)
			tb.st.Select(tb.buf.Offset(a0), tb.buf.Offset(b0))
		case "find":
			tb.st.Find(arg)
		case "scroll":
			tb.st.SetScrollY(tb.st.LineTop(max(atoi(arg)-1, 0)))
		case "peek":
			a.openPeek()
		case "rename":
			a.openRename()
		case "acts":
			a.openActs()
		case "inline":
			a.openInline()
		case "refs":
			a.showRefs(a.cursorWord())
		case "def":
			a.goDef()
		case "hover":
			a.hoverAt(tb, editor.HoverRequest{Offset: tb.buf.Offset(pos(arg))})
		}
	}
}

func atoi(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			break
		}
		n = n*10 + int(r-'0')
	}
	return n
}
