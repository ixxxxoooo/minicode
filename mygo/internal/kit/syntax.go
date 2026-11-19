package kit

import (
	"regexp"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 设计稿里 Rust 的极简分词(docs/design/js/ui.js 的 RX):注释 / 字符串 / 关键字 / 类型 / 数字 / 函数名。
// 真正的编辑器用库自带的 tokenizer;这里只给工具卡里的差异行和代码预览着色。
var rustRx = regexp.MustCompile(`(//.*$)|("(?:[^"\\]|\\.)*"|'.')|\b(use|pub|struct|impl|fn|let|mut|self|Self|return|if|else|match|for|in|assert_eq|mod)\b|\b([A-Z][A-Za-z0-9]*)\b|\b(\d+)\b|\b([a-z_]\w*)(\()`)

// RustSpans 把一行 Rust 切成带颜色的片段(等宽 11.5px)。base 是默认文字色。
func RustSpans(src string, base ui.Color, size float32) []ui.Span {
	mk := func(t string, c ui.Color, italic bool) ui.Span {
		return ui.Span{Text: t, Font: theme.CodeFont(), Size: size, Color: c, Italic: italic}
	}
	var out []ui.Span
	last := 0
	for _, m := range rustRx.FindAllStringSubmatchIndex(src, -1) {
		if m[0] > last {
			out = append(out, mk(src[last:m[0]], base, false))
		}
		switch {
		case m[2] >= 0:
			out = append(out, mk(src[m[2]:m[3]], T.SynCom, true))
		case m[4] >= 0:
			out = append(out, mk(src[m[4]:m[5]], T.SynStr, false))
		case m[6] >= 0:
			out = append(out, mk(src[m[6]:m[7]], T.SynKw, false))
		case m[8] >= 0:
			out = append(out, mk(src[m[8]:m[9]], T.SynType, false))
		case m[10] >= 0:
			out = append(out, mk(src[m[10]:m[11]], T.SynNum, false))
		default: // 函数名:只染名字,括号保持默认色
			out = append(out, mk(src[m[12]:m[13]], T.SynFn, false), mk(src[m[14]:m[15]], base, false))
		}
		last = m[1]
	}
	if last < len(src) {
		out = append(out, mk(src[last:], base, false))
	}
	if len(out) == 0 {
		out = append(out, mk(" ", base, false))
	}
	return out
}
