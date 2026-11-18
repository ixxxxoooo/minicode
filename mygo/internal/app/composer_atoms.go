package app

import (
	"regexp"
	"strings"

	"github.com/egoist/mygo/ui/richtext"
)

// 输入框正文里的「原子」(参照 BaoCode 的 composer embed / Cursor 的 @ 胶囊):文件、文件夹、折叠的长文本
// 在草稿字符串里是一段私有区标记 ` 类型 \x1f 路径 `,在输入框里整体画成一枚胶囊——
// 光标跳过它、退格整块删掉、可以夹在文字中间;发送时才展开成 `@路径`(Claude 认得的写法)。
//
// @author ixxxxoooo

const (
	atomOpen  = ""
	atomClose = ""
	atomSep   = "\x1f"
)

// 原子的类型。paste 后面跟字数,如 "paste:2100"。
const (
	atomFile  = "file"
	atomDir   = "dir"
	atomPaste = "paste"
	atomImg   = "img" // 图片在正文里的位置标记:只是界面上的行内胶囊,图片本身仍走输入框上方的缩略图附件,发送时不进文字
)

// atomRE 匹配草稿里的一个原子,捕获「类型\x1f路径」。
var atomRE = regexp.MustCompile(atomOpen + `([^` + atomClose + `]*)` + atomClose)

// atomString 把类型与路径编成草稿里的原子字符串。
func atomString(kind, path string) string { return atomOpen + kind + atomSep + path + atomClose }

// atomParts 拆开原子数据「类型\x1f路径」;格式不对时 ok 为 false。
func atomParts(data string) (kind, path string, ok bool) {
	kind, path, ok = strings.Cut(data, atomSep)
	return kind, path, ok && path != ""
}

// refAtom 把「@路径」引用(文件夹以 / 结尾)变成原子字符串;不是 @ 开头时返回空。
func refAtom(ref string) string {
	if !strings.HasPrefix(ref, "@") || len(ref) < 2 {
		return ""
	}
	p := strings.TrimPrefix(ref, "@")
	if strings.HasSuffix(p, "/") {
		return atomString(atomDir, strings.TrimSuffix(p, "/"))
	}
	return atomString(atomFile, p)
}

// appendRefAtoms 把一批「@路径」引用以原子(胶囊)的形式接到草稿末尾:已有的不重复,文字与胶囊之间补空格。
func appendRefAtoms(draft string, refs ...string) string {
	out := draft
	for _, r := range refs {
		at := refAtom(r)
		if at == "" || strings.Contains(out, at) {
			continue
		}
		if out != "" && !strings.HasSuffix(out, " ") && !strings.HasSuffix(out, "\n") {
			out += " "
		}
		out += at + " "
	}
	return out
}

// atomsToSend 把草稿里的原子展开成发给 Claude 的文字:`@路径`(文件夹 `@路径/`),前后没有空白就补一个。
func atomsToSend(s string) string {
	if !strings.Contains(s, atomOpen) {
		return s
	}
	var b strings.Builder
	last := 0
	for _, m := range atomRE.FindAllStringSubmatchIndex(s, -1) {
		b.WriteString(s[last:m[0]])
		kind, path, ok := atomParts(s[m[2]:m[3]])
		last = m[1]
		if !ok {
			continue
		}
		if kind == atomImg { // 位置标记不发给模型(图片另走附件)
			continue
		}
		if kind == atomDir && !strings.HasSuffix(path, "/") {
			path += "/"
		}
		if cur := b.String(); cur != "" && !strings.HasSuffix(cur, " ") && !strings.HasSuffix(cur, "\n") {
			b.WriteByte(' ')
		}
		if strings.ContainsAny(path, " \t") { // 路径带空格:用 @"…" 引起来(Claude Code 认这种写法),否则会在空格处被截断
			b.WriteString("@\"" + path + "\"")
		} else {
			b.WriteString("@" + path)
		}
		if last < len(s) && s[last] != ' ' && s[last] != '\n' {
			b.WriteByte(' ')
		}
	}
	b.WriteString(s[last:])
	return b.String()
}

// draftToText 把正文(段落以 \n 分隔,原子以标记串出现)变成富文本:原子是一个 U+FFFC 附件字符。
// 参数:mk 按「类型、路径」造出这枚胶囊的附件。
func draftToText(body string, mk func(kind, path string) *richtext.Attachment) richtext.Text {
	var paras []richtext.Paragraph
	for _, line := range strings.Split(body, "\n") {
		var runs []richtext.Run
		last := 0
		for _, m := range atomRE.FindAllStringSubmatchIndex(line, -1) {
			if m[0] > last {
				runs = append(runs, richtext.Run{Text: line[last:m[0]]})
			}
			last = m[1]
			if kind, path, ok := atomParts(line[m[2]:m[3]]); ok {
				runs = append(runs, richtext.Run{Text: string(richtext.ObjectReplacement), Attributes: richtext.Attributes{Attachment: mk(kind, path)}})
			}
		}
		if last < len(line) {
			runs = append(runs, richtext.Run{Text: line[last:]})
		}
		paras = append(paras, richtext.Paragraph{Runs: runs})
	}
	return richtext.TextOf(paras...)
}

// pillData 是胶囊附件的 Data:类型与路径。
type pillData struct{ Kind, Path string }

// textToDraft 是 draftToText 的逆运算:段落以 \n 连接,附件还原成原子标记串。
func textToDraft(t richtext.Text) string {
	var lines []string
	for _, p := range t.Paragraphs() {
		var b strings.Builder
		for _, r := range p.Runs {
			if at := r.Attributes.Attachment; at != nil {
				if d, ok := at.Data.(pillData); ok {
					b.WriteString(atomString(d.Kind, d.Path))
				}
				continue
			}
			b.WriteString(r.Text)
		}
		lines = append(lines, b.String())
	}
	return strings.Join(lines, "\n")
}
