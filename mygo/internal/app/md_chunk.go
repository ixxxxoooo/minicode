package app

import "strings"

// 长文本分块(会话流虚拟化的第二层):
// 一条助手正文可能有几千行。放在一行里,只要露出一角整段 Markdown 都要解析 + 排版;分成若干块、每块一行,
// 虚拟列表就只构建视口附近的块,屏幕外的块用占位高度。流式输出时已完成的块用静态渲染(有缓存),只有最后一块在重新解析。
const (
	mdChunkMin    = 4000 // 短于它不分块
	mdChunkTarget = 2400 // 每块的目标大小(字节),在其后的第一个安全边界切
)

// splitMarkdownChunks 把长 Markdown 按「安全边界」切成块,拼起来等于原文。
// 安全边界 = 围栏代码块之外、空行之后、下一行是新块起点(不缩进、不是列表项 / 引用 / 表格续行)的位置,
// 所以不会把列表、引用、代码块、表格从中间切开。边界只由它前面的内容决定,文本在末尾增长时已有的块不变(流式时键稳定)。
func splitMarkdownChunks(text string) []string {
	if len(text) <= mdChunkMin {
		return []string{text}
	}
	var chunks []string
	start, pos := 0, 0
	inFence := false
	fence := ""
	prevBlank := false
	for pos < len(text) {
		end := strings.IndexByte(text[pos:], '\n')
		var line string
		next := len(text)
		if end >= 0 {
			line = text[pos : pos+end]
			next = pos + end + 1
		} else {
			line = text[pos:]
		}
		trim := strings.TrimLeft(line, " ")
		if inFence {
			if strings.HasPrefix(trim, fence) && strings.TrimSpace(trim) == fence {
				inFence = false
			}
		} else {
			if f := fenceOf(trim, len(line)-len(trim)); f != "" {
				inFence, fence = true, f
			} else if prevBlank && pos-start >= mdChunkTarget && isBlockStart(line) {
				chunks = append(chunks, text[start:pos])
				start = pos
			}
		}
		prevBlank = strings.TrimSpace(line) == "" && !inFence
		pos = next
	}
	return append(chunks, text[start:])
}

// fenceOf 若这一行开了围栏代码块,返回围栏字符串(三个以上的 ` 或 ~)。indent > 3 是缩进代码块,不算围栏。
func fenceOf(trim string, indent int) string {
	if indent > 3 {
		return ""
	}
	for _, ch := range []byte{'`', '~'} {
		n := 0
		for n < len(trim) && trim[n] == ch {
			n++
		}
		if n >= 3 {
			if ch == '`' && strings.Contains(trim[n:], "`") { // 行内的 ```code``` 不是围栏
				return ""
			}
			return trim[:n]
		}
	}
	return ""
}

// isBlockStart 这一行适合当一个新块的开头:非空、不缩进,且不是列表项 / 引用 / 缩进续行。
func isBlockStart(line string) bool {
	if line == "" || line[0] == ' ' || line[0] == '\t' || line[0] == '>' {
		return false
	}
	if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") || strings.HasPrefix(line, "+ ") {
		return false
	}
	i := 0
	for i < len(line) && line[i] >= '0' && line[i] <= '9' {
		i++
	}
	if i > 0 && i < len(line)-1 && (line[i] == '.' || line[i] == ')') && line[i+1] == ' ' { // 有序列表项
		return false
	}
	return true
}
