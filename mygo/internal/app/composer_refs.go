package app

import (
	"regexp"
	"strings"
)

var draftAtRE = regexp.MustCompile(`@([^\s@]+)`)

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
