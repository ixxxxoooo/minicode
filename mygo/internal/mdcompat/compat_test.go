package mdcompat

import (
	"strings"
	"testing"
)

func TestPrepareGFMTableBlankLine(t *testing.T) {
	src := "上文不换行\n| a | b |\n| --- | --- |\n| 1 | 2 |"
	out := Prepare(src)
	if !strings.Contains(out, "上文不换行\n\n| a |") {
		t.Fatalf("表格前应插入空行:\n%s", out)
	}
}

func TestEmojiShortcode(t *testing.T) {
	got := Prepare("Hello :smile: world")
	if !strings.Contains(got, "😄") {
		t.Fatalf("应展开 :smile:: %q", got)
	}
	if strings.Contains(Prepare("```\n:smile:\n```"), "😄") {
		t.Fatal("围栏内不应展开 emoji")
	}
}

func TestCompatFeatures(t *testing.T) {
	cases := []struct {
		name string
		md   string
		want []string // 纯文本中应包含的子串
	}{
		{"表格", "| h | v |\n| --- | --- |\n| a | `b\\|c` |", []string{"a", "b|c"}},
		{"任务列表", "- [x] done\n- [ ] todo", []string{"done", "todo"}},
		{"删除线", "~~strike~~", []string{"strike"}},
		{"自动链接", "<https://example.com>", []string{"example"}},
		{"裸 URL", "see https://example.com/path", []string{"example", "path"}},
		{"脚注", "text[^1]\n\n[^1]: note", []string{"text", "note"}},
		{"围栏代码", "```go\nfmt.Println(\"hi\")\n```", []string{"fmt", "hi"}},
		{"缩进代码", "    indent", []string{"indent"}},
		{"引用嵌套", "> q\n> > nested", []string{"q", "nested"}},
		{"分割线", "a\n\n---\n\nb", []string{"a", "b"}},
		{"setext", "Title\n=====", []string{"Title"}},
		{"行内 HTML", "a <br> b <kbd>K</kbd>", []string{"a", "b", "K"}},
		{"HTML 块", "<div>block</div>", []string{"block"}},
		{"数学", "$x^2$", []string{"x"}},
		{"转义管道", "cell \\| here", []string{"cell", "here"}},
		{"混合列表", "1. one\n2. two\n   - sub", []string{"one", "two", "sub"}},
		{"frontmatter 保留", "---\ntitle: x\n---\n\nbody", []string{"body", "title"}},
		{"CRLF", "line\r\nnext", []string{"line", "next"}},
		{"全角", "中文，表格\n| 列 | 值 |\n| --- | --- |\n| 甲 | 乙 |", []string{"中文", "甲", "乙"}},
		{"表格套 HTML", "| h |\n| --- |\n| <b>x</b> |", []string{"x"}},
		{"空表头", "| | |\n| --- | --- |\n| a | b |", []string{"a", "b"}},
		{"列不齐", "| a | b | c |\n| --- | --- |\n| 1 | 2 |", []string{"1", "2"}},
		{"details", "<details><summary>T</summary>body</details>", []string{"body", "T"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plain := PlainText(Parse(tc.md))
			for _, w := range tc.want {
				if !strings.Contains(plain, w) {
					t.Fatalf("缺 %q\nplain=%q\nmd=%q", w, plain, tc.md)
				}
			}
		})
	}
}

func TestParseDetailsHelper(t *testing.T) {
	s, b, ok := ParseDetails("<details><summary>标题</summary><p>正文</p></details>")
	if !ok || s != "标题" || !strings.Contains(b, "正文") {
		t.Fatalf("details 解析失败: %q %q %v", s, b, ok)
	}
}

func TestVisibleTextFromHTML(t *testing.T) {
	got := VisibleTextFromHTML(`<p align="center">居中<b>粗</b></p>`)
	if !strings.Contains(got, "居中") || !strings.Contains(got, "粗") {
		t.Fatalf("应保留文本: %q", got)
	}
}
