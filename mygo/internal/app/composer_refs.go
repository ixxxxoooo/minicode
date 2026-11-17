package app

import (
	"regexp"
	"strings"

	"github.com/egoist/mygo/ui"
)

// 文件 / 文件夹引用(拖入、添加、文件树「引用到对话」、长文本粘贴折叠)都写进输入框正文,
// 和打的字在一起,插在光标处,并用强调色 + 中等字重显示(与 @ 提及同款);只有图片是输入框上方的缩略图附件。
//
// @author ixxxxoooo

// draftRefRE 匹配正文里的「@路径」:前面是开头或空白,后面到下一个空白为止。
var draftRefRE = regexp.MustCompile(`(^|\s)(@[^\s@]+)`)

// insertRefs 把一批「@路径」排队插进输入框光标处(下一帧构建输入框时落地);正文里已有的不重复。
// 参数:refs 形如 "@src/a.go" / "@docs/"。返回:实际新增了几个。
func (a *App) insertRefs(refs ...string) int {
	have := a.draftNow()
	var add []string
	for _, r := range refs {
		if r == "" || strings.Contains(" "+strings.ReplaceAll(have, "\n", " ")+" ", " "+r+" ") {
			continue
		}
		dup := false
		for _, x := range add {
			dup = dup || x == r
		}
		if !dup {
			add = append(add, r)
		}
	}
	if len(add) == 0 {
		return 0
	}
	a.ux.pendingInsert += strings.Join(add, " ") + " "
	a.ux.focusDraft = true
	return len(add)
}

// takePendingInsert 取出排队的插入文字,并按光标前后补空格:前面不是空白就补一个空格隔开。
// 参数:body 输入框当前正文;start 光标 / 选区起点(rune 偏移)。返回:要插入的文字,没有排队时为空。
func (a *App) takePendingInsert(body string, start int) string {
	ins := a.ux.pendingInsert
	a.ux.pendingInsert = ""
	if ins == "" {
		return ""
	}
	rs := []rune(body)
	if start > 0 && start <= len(rs) {
		if p := rs[start-1]; p != ' ' && p != '\n' && p != '\t' {
			ins = " " + ins
		}
	}
	return ins
}

// refTextRanges 给正文里的「@路径」上样式:强调色 + 中等字重(范围按 rune 偏移,TextRanges 的要求)。
func refTextRanges(body string, color ui.Color) []ui.TextRange {
	var out []ui.TextRange
	for _, m := range draftRefRE.FindAllStringSubmatchIndex(body, -1) {
		s, e := m[4], m[5] // 第 2 个分组 = @路径 本身
		out = append(out, ui.TextRange{
			Start: len([]rune(body[:s])), End: len([]rune(body[:e])),
			Color: color, Weight: 500,
		})
	}
	return out
}

// refRangeColor 引用文字的颜色。
func (a *App) refRangeColor() ui.Color { return a.tk.Info }
