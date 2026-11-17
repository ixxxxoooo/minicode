// Package mdcompat 在 MyGo 自带 Markdown 解析器之上做输入规范化与兼容性辅助(不替代 parser)。
package mdcompat

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	mdTableRow = regexp.MustCompile(`^\s*\|?.+\|.+\|?\s*$`)
	mdTableSep = regexp.MustCompile(`^\s*\|?\s*:?-{3,}:?\s*(\|\s*:?-{3,}:?\s*)+\|?\s*$`)
)

// Prepare 在交给 markdown.View / Stream 之前规范化助手与用户 Markdown 源码。
// 只做无损或明确安全的变换,不改变围栏代码块内的字节。
func Prepare(src string) string {
	if src == "" {
		return src
	}
	src = strings.ReplaceAll(src, "\r\n", "\n")
	src = strings.ReplaceAll(src, "\r", "\n")
	src = stripZeroWidth(src)
	src = expandEmojiShortcodes(src)
	src = ensureBlankBeforeGFMTable(src)
	return src
}

// ensureBlankBeforeGFMTable 在 GFM 表格前补空行,避免紧挨段落的 | 行被当成普通文本。
func ensureBlankBeforeGFMTable(src string) string {
	lines := strings.Split(src, "\n")
	var out []string
	inFence := false
	fence := ""
	for i, line := range lines {
		trim := strings.TrimLeft(line, " \t")
		if inFence {
			if strings.HasPrefix(trim, fence) && strings.TrimSpace(trim) == fence {
				inFence = false
			}
			out = append(out, line)
			continue
		}
		if f := openFence(trim); f != "" {
			inFence, fence = true, f
			out = append(out, line)
			continue
		}
		if i > 0 && mdTableRow.MatchString(line) && (i+1 < len(lines) && mdTableSep.MatchString(lines[i+1])) {
			prev := strings.TrimSpace(out[len(out)-1])
			if prev != "" {
				out = append(out, "")
			}
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func openFence(trim string) string {
	for _, ch := range []byte{'`', '~'} {
		n := 0
		for n < len(trim) && trim[n] == ch {
			n++
		}
		if n >= 3 {
			if ch == '`' && strings.Contains(trim[n:], "`") {
				return ""
			}
			return trim[:n]
		}
	}
	return ""
}

// emojiMap 常见短码 → Unicode;未知短码原样保留。
var emojiMap = map[string]string{
	"smile": "😄", "smiley": "😃", "grin": "😁", "joy": "😂", "wink": "😉",
	"thinking": "🤔", "thumbsup": "👍", "thumbsdown": "👎", "+1": "👍", "-1": "👎",
	"heart": "❤️", "fire": "🔥", "rocket": "🚀", "tada": "🎉", "warning": "⚠️",
	"white_check_mark": "✅", "x": "❌", "bug": "🐛", "eyes": "👀",
}

var emojiShort = regexp.MustCompile(`:([a-zA-Z0-9_+-]+):`)

func expandEmojiShortcodes(src string) string {
	lines := strings.Split(src, "\n")
	inFence := false
	fence := ""
	for i, line := range lines {
		trim := strings.TrimLeft(line, " \t")
		if inFence {
			if strings.HasPrefix(trim, fence) && strings.TrimSpace(trim) == fence {
				inFence = false
			}
		} else if f := openFence(trim); f != "" {
			inFence, fence = true, f
		} else {
			lines[i] = emojiShort.ReplaceAllStringFunc(line, func(m string) string {
				name := m[1 : len(m)-1]
				if e, ok := emojiMap[name]; ok {
					return e
				}
				return m
			})
		}
	}
	return strings.Join(lines, "\n")
}

// ContentMultiset 统计「内容性」字符(字母、数字、CJK 等)出现次数,用于容错测试。
func ContentMultiset(s string) map[rune]int {
	m := map[rune]int{}
	for _, r := range s {
		if isContentRune(r) {
			m[r]++
		}
	}
	return m
}

// LetterMultiset 只统计字母与 CJK,用于截断容错测试(忽略列表序号等数字标记)。
func LetterMultiset(s string) map[rune]int {
	m := map[rune]int{}
	for _, r := range s {
		if unicode.Is(unicode.Cf, r) {
			continue
		}
		if r >= 0x80 || unicode.IsLetter(r) {
			m[r]++
		}
	}
	return m
}

// MultisetContains 报告 sup 是否包含 sub 中的每个内容字符(次数不少于 sub)。
func MultisetContains(sup, sub map[rune]int) bool {
	for r, n := range sub {
		if sup[r] < n {
			return false
		}
	}
	return true
}

func isContentRune(r rune) bool {
	return r >= 0x80 || unicode.IsLetter(r) || unicode.IsNumber(r)
}

func stripZeroWidth(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == '\u200b' || r == '\u200c' || r == '\u200d' || r == '\ufeff' {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
