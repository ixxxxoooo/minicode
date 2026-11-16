package app

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/egoist/mygo/ui/richtext"
)

// mdBlockPart 是正文里的一块 Markdown(段落 / 标题 / 列表项 / 围栏代码等),after 为块后到下一块之间的空白(含换行)。
type mdBlockPart struct {
	raw   string
	after string
}

// mdBodySplit 把 Markdown 正文按块切开,保留每块原始字节与块间空白。
type mdBodySplit struct {
	lead  string // 正文开头到第一块之间的空白
	parts []mdBlockPart
}

// mdPreserveBody 把所见即所得序列化结果与读入时的块对照,只替换语义上变了的块。
func mdPreserveBody(origBody string, orig mdBodySplit, newBody string) string {
	if origBody == "" && newBody == "" {
		return ""
	}
	if len(orig.parts) == 0 {
		out := mdPreserveRoundTripBody(origBody, newBody)
		return out
	}
	newBody = mdFixMarks(origBody, newBody)
	neu := mdSplitBodyBlocks(newBody)
	merged := mdMergeBlockParts(orig.parts, orig.lead, neu.parts, neu.lead, newBody, origBody)
	return merged
}

// mdPreserveRoundTrip 在所见即所得序列化后,对照原文保守补回围栏语言标注(一一对应时)。
func mdPreserveRoundTrip(origFull, newFull string) string {
	_, origBody := mdSplitFront(origFull)
	newFront, newBody := mdSplitFront(newFull)
	patched := mdPreserveRoundTripBody(origBody, newBody)
	if patched == newBody {
		return newFull
	}
	out := newFront
	if patched != "" {
		out += patched
		if !strings.HasSuffix(out, "\n") {
			out += "\n"
		}
	}
	return out
}

func mdPreserveRoundTripBody(origBody, newBody string) string {
	patched := mdPatchFenceLangs(origBody, newBody)
	if patched == newBody {
		return newBody
	}
	return patched
}

// mdBlockSemKey 用语义序列化结果比较两块是否「内容相同」(* 与 -、段内换行不算)。
func mdBlockSemKey(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	return strings.TrimSpace(richtext.ParseMarkdown(raw).Markdown())
}

// mdBlockWouldNormalize 判断单块在整段序列化后是否与原文不同。
func mdBlockWouldNormalize(raw string) bool {
	trim := strings.TrimSpace(raw)
	if trim == "" || mdIsThematicBreak(trim) { // 分隔线序列化会丢,但没改动的块保持原样,编辑别的段落不受影响,不算
		return false
	}
	ser := mdFixMarks(trim, strings.TrimSpace(richtext.ParseMarkdown(raw).Markdown()))
	return mdLooseKey(trim) != mdLooseKey(ser)
}

// mdIsThematicBreak 是不是分隔线(--- / *** / ___,至少三个同种字符)。
func mdIsThematicBreak(s string) bool {
	if len(s) < 3 || strings.Contains(s, "\n") {
		return false
	}
	c := s[0]
	if c != '-' && c != '*' && c != '_' {
		return false
	}
	return strings.Trim(strings.ReplaceAll(s, " ", ""), string(c)) == ""
}

var (
	mdQuoteLead = regexp.MustCompile(`(?m)^[ \t]*(>[ \t]?)+`)
	mdSpaceRun  = regexp.MustCompile(`\s+`)
	mdOrdLead   = regexp.MustCompile(`^(\d+)[.)] `)
	mdEscape    = regexp.MustCompile("\\\\([!-/:-@\\[-`{-~])")
)

// mdLooseKey 给「写法不同但内容没变」的块算同一个键,这些差别序列化后虽然字节不同,实际不会改变文档:
// 段落里的单个换行(软换行,等价于空格)、引用块每行的 ">"、代码围栏的语言标注(编辑时由 mdPatchFenceLangs 补回)。
// 列表符(* → -)、强调符号这类真会被改写的写法仍然保留差异。
func mdLooseKey(s string) string {
	if strings.HasPrefix(s, "```") || strings.HasPrefix(s, "~~~") {
		// 围栏块:忽略第一行的语言标注,正文逐字比较(代码里的空白有意义)
		if i := strings.Index(s, "\n"); i >= 0 {
			return s[:3] + "\n" + strings.TrimSpace(s[i+1:])
		}
		return s[:3]
	}
	s = mdQuoteLead.ReplaceAllString(s, "")
	s = mdEscape.ReplaceAllString(s, "$1")   // 反斜杠转义(\< \~ \_ \[ …)渲染结果相同
	s = strings.ReplaceAll(s, "`", "")       // 行内代码含反引号时分隔符个数会变(` ```x ` → ```` ```x ````),渲染相同
	s = mdOrdLead.ReplaceAllString(s, "1. ") // 有序列表按项切块后,单独一项的编号(2. / 3.)会被序列化成 1.,是切块的副作用
	return strings.TrimSpace(mdSpaceRun.ReplaceAllString(s, " "))
}

// mdNormalizedBlocks 统计「若编辑该段会被规范化」的非空行数(只计会变的块)。
func mdNormalizedBlocks(body string) int {
	sp := mdSplitBodyBlocks(body)
	n := 0
	for _, p := range sp.parts {
		if !mdBlockWouldNormalize(p.raw) {
			continue
		}
		for _, ln := range strings.Split(p.raw, "\n") {
			if strings.TrimSpace(ln) != "" {
				n++
			}
		}
	}
	return n
}

func mdJoinBlocks(sp mdBodySplit) string {
	var b strings.Builder
	b.WriteString(sp.lead)
	for i, p := range sp.parts {
		b.WriteString(p.raw)
		if p.after != "" {
			b.WriteString(p.after)
		} else if i < len(sp.parts)-1 {
			b.WriteString("\n\n")
		}
	}
	return b.String()
}

func mdSplitBodyBlocks(body string) mdBodySplit {
	var sp mdBodySplit
	if body == "" {
		return sp
	}
	lines := strings.Split(body, "\n")
	trailingNL := strings.HasSuffix(body, "\n")
	i := 0
	for i < len(lines) && lines[i] == "" {
		sp.lead += "\n"
		i++
	}
	for i < len(lines) {
		var raw []string
		switch {
		case mdFenceOpen(lines[i]):
			raw, i = mdConsumeFence(lines, i)
		case mdATXHeading(lines[i]):
			raw, i = mdConsumeUntilBlank(lines, i)
		case mdQuoteLine(lines[i]):
			raw, i = mdConsumeQuote(lines, i)
		case mdListItemLine(lines[i]):
			raw, i = mdConsumeListItem(lines, i)
		default:
			raw, i = mdConsumeParagraph(lines, i)
		}
		part := mdBlockPart{raw: strings.Join(raw, "\n")}
		after, ni := mdConsumeSep(lines, i)
		part.after = after
		i = ni
		sp.parts = append(sp.parts, part)
	}
	if trailingNL && len(sp.parts) > 0 && sp.parts[len(sp.parts)-1].after == "" {
		sp.parts[len(sp.parts)-1].after = "\n"
	}
	return sp
}

func mdConsumeSep(lines []string, i int) (after string, next int) {
	for i < len(lines) && lines[i] == "" {
		after += "\n"
		i++
	}
	return after, i
}

func mdFenceOpen(line string) bool {
	t := strings.TrimSpace(line)
	return strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~")
}

func mdConsumeFence(lines []string, i int) (raw []string, next int) {
	open := strings.TrimSpace(lines[i])
	fence := open[:3]
	raw = append(raw, lines[i])
	i++
	for i < len(lines) {
		t := strings.TrimSpace(lines[i])
		if strings.HasPrefix(t, fence) {
			rest := strings.TrimPrefix(t, fence)
			if strings.TrimSpace(rest) == "" {
				raw = append(raw, lines[i])
				i++
				break
			}
		}
		raw = append(raw, lines[i])
		i++
	}
	return raw, i
}

func mdATXHeading(line string) bool {
	t := strings.TrimSpace(line)
	if t == "" || t[0] != '#' {
		return false
	}
	n := 0
	for n < len(t) && t[n] == '#' {
		n++
	}
	return n > 0 && n <= 6 && (n == len(t) || t[n] == ' ')
}

func mdQuoteLine(line string) bool {
	t := strings.TrimLeft(line, " \t")
	return strings.HasPrefix(t, ">")
}

func mdConsumeQuote(lines []string, i int) (raw []string, next int) {
	for i < len(lines) {
		if lines[i] == "" {
			break
		}
		if !mdQuoteLine(lines[i]) && !mdBlankContinuation(lines[i]) {
			break
		}
		raw = append(raw, lines[i])
		i++
	}
	return raw, i
}

func mdBlankContinuation(line string) bool {
	return strings.TrimSpace(line) != "" && (strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "\t"))
}

func mdConsumeParagraph(lines []string, i int) (raw []string, next int) {
	for i < len(lines) {
		if lines[i] == "" {
			break
		}
		if mdFenceOpen(lines[i]) || mdATXHeading(lines[i]) || mdQuoteLine(lines[i]) || mdListItemLine(lines[i]) {
			break
		}
		raw = append(raw, lines[i])
		i++
	}
	return raw, i
}

func mdConsumeUntilBlank(lines []string, i int) (raw []string, next int) {
	raw = append(raw, lines[i])
	i++
	for i < len(lines) && lines[i] != "" {
		raw = append(raw, lines[i])
		i++
	}
	return raw, i
}

func mdListItemLine(line string) bool {
	_, ok := mdListIndent(line)
	return ok
}

func mdListIndent(line string) (base int, ok bool) {
	i := 0
	for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
		i++
	}
	rest := strings.TrimLeft(line[i:], " \t")
	if rest == "" {
		return 0, false
	}
	if strings.HasPrefix(rest, "- [ ] ") || strings.HasPrefix(rest, "- [x] ") || strings.HasPrefix(rest, "- [X] ") {
		return i, true
	}
	if len(rest) >= 2 && (rest[0] == '-' || rest[0] == '*' || rest[0] == '+') && rest[1] == ' ' {
		return i, true
	}
	j := 0
	for j < len(rest) && unicode.IsDigit(rune(rest[j])) {
		j++
	}
	if j > 0 && j < len(rest) && rest[j] == '.' && (j+1 == len(rest) || rest[j+1] == ' ') {
		return i, true
	}
	return 0, false
}

func mdConsumeListItem(lines []string, i int) (raw []string, next int) {
	base, _ := mdListIndent(lines[i])
	raw = append(raw, lines[i])
	i++
	for i < len(lines) {
		if lines[i] == "" {
			// 空行:若下一行是缩进续行则仍属本项
			if i+1 < len(lines) && mdIndentedMore(lines[i+1], base) {
				raw = append(raw, lines[i])
				i++
				continue
			}
			break
		}
		if mdListItemLine(lines[i]) {
			ni, _ := mdListIndent(lines[i])
			if ni <= base {
				break
			}
		}
		if mdFenceOpen(lines[i]) || mdATXHeading(lines[i]) {
			break
		}
		if !mdIndentedMore(lines[i], base) && !mdListItemLine(lines[i]) {
			break
		}
		raw = append(raw, lines[i])
		i++
	}
	return raw, i
}

func mdIndentedMore(line string, base int) bool {
	n := 0
	for n < len(line) && (line[n] == ' ' || line[n] == '\t') {
		n++
	}
	return n > base && strings.TrimSpace(line) != ""
}

func mdMergeBlockParts(orig []mdBlockPart, origLead string, neu []mdBlockPart, neuLead, newBody, origBody string) string {
	if len(orig) == 0 {
		return mdPreserveRoundTripBody(origBody, newBody)
	}
	oKeys := make([]string, len(orig))
	nKeys := make([]string, len(neu))
	for i, p := range orig {
		oKeys[i] = mdBlockSemKey(p.raw)
	}
	for i, p := range neu {
		nKeys[i] = mdBlockSemKey(p.raw)
	}
	align := mdLCSAlign(oKeys, nKeys)
	if len(align) == 0 && (len(orig) > 1 || len(neu) > 1) {
		return mdPreserveRoundTripBody(origBody, newBody)
	}
	var out []mdBlockPart
	lead := origLead
	if lead == "" {
		lead = neuLead
	}
	oi, ni := 0, 0
	for _, step := range align {
		switch step.kind {
		case lcsOrigDel:
			oi++
		case lcsNeuIns:
			p := neu[ni]
			if oi > 0 && oi-1 < len(orig) {
				p.after = orig[oi-1].after
			}
			out = append(out, p)
			ni++
		case lcsMatch:
			op, np := orig[oi], neu[ni]
			if oKeys[oi] == nKeys[ni] {
				out = append(out, mdBlockPart{raw: op.raw, after: op.after})
			} else {
				after := op.after
				if after == "" {
					after = np.after
				}
				out = append(out, mdBlockPart{raw: np.raw, after: after})
			}
			oi++
			ni++
		}
	}
	for ni < len(neu) {
		out = append(out, neu[ni])
		ni++
	}
	return mdJoinBlocks(mdBodySplit{lead: lead, parts: out})
}

type lcsStepKind int

const (
	lcsMatch lcsStepKind = iota
	lcsOrigDel
	lcsNeuIns
)

type lcsStep struct {
	kind lcsStepKind
}

func mdLCSAlign(a, b []string) []lcsStep {
	n, m := len(a), len(b)
	if n == 0 && m == 0 {
		return nil
	}
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}
	var steps []lcsStep
	i, j := 0, 0
	for i < n && j < m {
		if a[i] == b[j] {
			steps = append(steps, lcsStep{kind: lcsMatch})
			i++
			j++
		} else if dp[i+1][j] >= dp[i][j+1] {
			steps = append(steps, lcsStep{kind: lcsOrigDel})
			i++
		} else {
			steps = append(steps, lcsStep{kind: lcsNeuIns})
			j++
		}
	}
	for i < n {
		steps = append(steps, lcsStep{kind: lcsOrigDel})
		i++
	}
	for j < m {
		steps = append(steps, lcsStep{kind: lcsNeuIns})
		j++
	}
	match := 0
	for _, s := range steps {
		if s.kind == lcsMatch {
			match++
		}
	}
	denom := len(a)
	if len(b) > denom {
		denom = len(b)
	}
	if denom > 0 && match*2 < denom {
		return nil
	}
	return steps
}

func mdPatchFenceLangs(orig, neu string) string {
	olangs := fenceLangs(orig)
	nlangs := fenceLangs(neu)
	if len(olangs) == 0 || len(olangs) != len(nlangs) {
		return neu
	}
	var b strings.Builder
	oi := 0
	fence := ""
	for _, ln := range strings.Split(neu, "\n") {
		trim := strings.TrimSpace(ln)
		if fence != "" {
			if strings.HasPrefix(trim, fence) {
				fence = ""
			}
			b.WriteString(ln)
			b.WriteByte('\n')
			continue
		}
		if strings.HasPrefix(trim, "```") || strings.HasPrefix(trim, "~~~") {
			fence = trim[:3]
			lang := ""
			if oi < len(nlangs) && nlangs[oi] == "" && oi < len(olangs) && olangs[oi] != "" {
				lang = olangs[oi]
			}
			oi++
			if lang != "" {
				if idx := strings.IndexByte(ln, '`'); idx >= 0 {
					rest := ln[idx:]
					if strings.HasPrefix(rest, "```") {
						b.WriteString("```" + lang)
						if tail := strings.TrimPrefix(rest, "```"); tail != "" && !strings.HasPrefix(strings.TrimSpace(tail), lang) {
							b.WriteString(tail)
						}
						b.WriteByte('\n')
						continue
					}
				}
			}
		}
		b.WriteString(ln)
		b.WriteByte('\n')
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func fenceLangs(body string) []string {
	var langs []string
	fence := ""
	for _, ln := range strings.Split(body, "\n") {
		trim := strings.TrimSpace(ln)
		if fence != "" {
			if strings.HasPrefix(trim, fence) {
				fence = ""
			}
			continue
		}
		if strings.HasPrefix(trim, "```") {
			fence = "```"
			lang := strings.TrimSpace(strings.TrimPrefix(trim, "```"))
			langs = append(langs, lang)
		} else if strings.HasPrefix(trim, "~~~") {
			fence = "~~~"
			lang := strings.TrimSpace(strings.TrimPrefix(trim, "~~~"))
			langs = append(langs, lang)
		}
	}
	return langs
}

// mdFixMarks 把序列化器擅自改写成 HTML 的强调标记换回 Markdown。
// 库(richtext/markdown.go)序列化后会读回校验,块里只要有一个分隔符读不回原样,就把这一块里所有加粗 / 斜体 / 删除线
// 整块写成 <strong> / <em> / <del>——「**看门狗会记栈**:…」这种中文紧跟全角标点的写法就会这样,编辑该段落后原文被换成 HTML 标签。
// 原文里本来就没有这个标签、结果里却多出来的,是库改写的,换回 ** / * / ~~(原文里的 ** 本来就被解析成了加粗)。
func mdFixMarks(orig, out string) string {
	for _, m := range [][3]string{{"strong", "**", "**"}, {"em", "*", "*"}, {"del", "~~", "~~"}} {
		open, closeTag := "<"+m[0]+">", "</"+m[0]+">"
		if strings.Contains(orig, open) || !strings.Contains(out, open) {
			continue
		}
		out = strings.ReplaceAll(strings.ReplaceAll(out, open, m[1]), closeTag, m[2])
	}
	return out
}
