// Package search 是项目内搜索 / 替换内核(与界面无关,走 workspace.FS 所以演示与真实项目通用)。
// 按行匹配,正则不跨行;每处匹配一条;列按字符(rune)计。
package search

import (
	"context"
	"errors"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/ixxxxoooo/minicode/internal/workspace"
)

// 限制:结果上限、单文件大小上限。
const (
	MaxMatches = 20000
	MaxFile    = 2 << 20
)

// Query 是一次搜索。
type Query struct {
	Text           string
	Case           bool // 区分大小写
	Word           bool // 全词
	Regex          bool
	Include        string // 逗号分隔的 glob,空 = 全部
	Exclude        string
	NoIgnore       bool // 不跳过 target / node_modules 等产物目录
	PreserveCase   bool // 替换时保留大小写(仅替换用)
	ExtraIgnoreDir []string
}

// Match 是一处命中。Line 从 0 起;Col / End 是该行内的字符列(半开区间)。
type Match struct {
	Path     string
	Line     int
	Col, End int
	Text     string // 整行(不含换行)
}

// FileResult 是一个文件里的全部命中。
type FileResult struct {
	Path    string
	Matches []Match
}

// Result 是搜索结果。
type Result struct {
	Files     []FileResult
	Total     int
	Truncated bool // 到达 MaxMatches 上限
}

// Compile 把查询编译成正则:字面量转义;全词加 \b;不区分大小写加 (?i)。
func Compile(q Query) (*regexp.Regexp, error) {
	if q.Text == "" {
		return nil, errors.New("空查询")
	}
	pat := q.Text
	if !q.Regex {
		pat = regexp.QuoteMeta(pat)
	}
	if q.Word {
		pat = `\b(?:` + pat + `)\b`
	}
	if !q.Case {
		pat = "(?i)" + pat
	}
	return regexp.Compile(pat)
}

var defaultIgnore = map[string]bool{"target": true, "node_modules": true, "dist": true, "build": true, "vendor": true}

// Run 搜索整个项目;ctx 取消时返回已找到的部分与 ctx.Err()。
func Run(ctx context.Context, fs workspace.FS, q Query) (Result, error) {
	re, err := Compile(q)
	if err != nil {
		return Result{}, err
	}
	inc, exc := parseGlobs(q.Include), parseGlobs(q.Exclude)
	var res Result
	var walk func(dir string) error
	walk = func(dir string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		es, err := fs.List(dir)
		if err != nil {
			return nil // 读不了的目录跳过
		}
		for _, e := range es {
			if res.Truncated {
				return nil
			}
			if strings.HasPrefix(e.Name, ".") {
				continue
			}
			if e.Dir {
				if !q.NoIgnore && defaultIgnore[e.Name] {
					continue
				}
				if err := walk(e.Path); err != nil {
					return err
				}
				continue
			}
			if len(inc) > 0 && !anyMatch(inc, e.Path) || anyMatch(exc, e.Path) {
				continue
			}
			fr := searchFile(fs, e.Path, re, MaxMatches-res.Total)
			if len(fr.Matches) == 0 {
				continue
			}
			res.Files = append(res.Files, fr)
			res.Total += len(fr.Matches)
			if res.Total >= MaxMatches {
				res.Truncated = true
			}
		}
		return nil
	}
	err = walk("")
	return res, err
}

// readable 读文件并过滤掉太大 / 二进制 / 非 UTF-8 的。
func readable(fs workspace.FS, p string) (string, bool) {
	src, err := fs.Read(p)
	if err != nil || len(src) > MaxFile {
		return "", false
	}
	head := src
	if len(head) > 8192 {
		head = head[:8192]
	}
	if strings.IndexByte(head, 0) >= 0 || !utf8.ValidString(src) {
		return "", false
	}
	return src, true
}

func searchFile(fs workspace.FS, p string, re *regexp.Regexp, room int) FileResult {
	fr := FileResult{Path: p}
	src, ok := readable(fs, p)
	if !ok {
		return fr
	}
	for i, line := range strings.Split(src, "\n") {
		line = strings.TrimSuffix(line, "\r")
		for _, loc := range re.FindAllStringIndex(line, -1) {
			if loc[1] == loc[0] { // 空匹配(如 a* )不算
				continue
			}
			fr.Matches = append(fr.Matches, Match{Path: p, Line: i, Col: utf8.RuneCountInString(line[:loc[0]]), End: utf8.RuneCountInString(line[:loc[1]]), Text: line})
			if len(fr.Matches) >= room {
				return fr
			}
		}
	}
	return fr
}

// ───────── 替换 ─────────

// Replace 只替换 picks 列出的那些命中(被用户忽略的不动),返回替换处数。
// 正则查询里 repl 可用 $1 / ${name} 引用捕获组;PreserveCase 时按原文大小写风格调整。
func Replace(fs workspace.FS, q Query, repl string, picks []Match) (int, error) {
	re, err := Compile(q)
	if err != nil {
		return 0, err
	}
	byFile := map[string][]Match{}
	for _, m := range picks {
		byFile[m.Path] = append(byFile[m.Path], m)
	}
	total := 0
	for p, ms := range byFile {
		src, ok := readable(fs, p)
		if !ok {
			continue
		}
		lines := strings.Split(src, "\n")
		sort.Slice(ms, func(i, j int) bool { // 同一行从右往左替换,前面的列不受影响
			if ms[i].Line != ms[j].Line {
				return ms[i].Line < ms[j].Line
			}
			return ms[i].Col > ms[j].Col
		})
		for _, m := range ms {
			if m.Line >= len(lines) {
				continue
			}
			line := lines[m.Line]
			cr := ""
			if strings.HasSuffix(line, "\r") {
				line, cr = line[:len(line)-1], "\r"
			}
			for _, sm := range re.FindAllStringSubmatchIndex(line, -1) {
				if sm[1] == sm[0] || utf8.RuneCountInString(line[:sm[0]]) != m.Col {
					continue
				}
				out := repl
				if q.Regex {
					out = string(re.ExpandString(nil, repl, line, sm))
				}
				if q.PreserveCase {
					out = matchCase(line[sm[0]:sm[1]], out)
				}
				line = line[:sm[0]] + out + line[sm[1]:]
				total++
				break
			}
			lines[m.Line] = line + cr
		}
		if err := fs.Write(p, strings.Join(lines, "\n")); err != nil {
			return total, err
		}
	}
	return total, nil
}

// matchCase 让 repl 沿用 orig 的大小写风格:全大写 / 全小写 / 首字母大写。
func matchCase(orig, repl string) string {
	if orig == "" || repl == "" {
		return repl
	}
	up, low := true, true
	hasLetter := false
	for _, r := range orig {
		if unicode.IsLetter(r) {
			hasLetter = true
			if unicode.IsLower(r) {
				up = false
			} else {
				low = false
			}
		}
	}
	switch {
	case !hasLetter:
		return repl
	case up && utf8.RuneCountInString(orig) > 1:
		return strings.ToUpper(repl)
	case low:
		return strings.ToLower(repl)
	}
	first, _ := utf8.DecodeRuneInString(orig)
	if unicode.IsUpper(first) {
		r, n := utf8.DecodeRuneInString(repl)
		return string(unicode.ToUpper(r)) + repl[n:]
	}
	return repl
}

// ───────── 预览 ─────────

// Preview 把命中行拆成「前文 / 命中 / 后文」:去掉行首缩进,过长时前后各留上下文并加省略号。
func Preview(m Match, maxRunes int) (pre, hit, post string) {
	rs := []rune(m.Text)
	col, end := m.Col, m.End
	if end > len(rs) {
		end = len(rs)
	}
	if col > end {
		col = end
	}
	lead := 0
	for lead < col && (rs[lead] == ' ' || rs[lead] == '\t') {
		lead++
	}
	rs, col, end = rs[lead:], col-lead, end-lead
	hit = string(rs[col:end])
	before, after := rs[:col], rs[end:]
	room := maxRunes - len([]rune(hit))
	if room < 8 {
		room = 8
	}
	if len(before) > room/3 {
		before = append([]rune("…"), before[len(before)-room/3:]...)
	}
	room -= len(before)
	if len(after) > room {
		after = append(append([]rune{}, after[:max(room, 0)]...), '…')
	}
	return string(before), hit, string(after)
}

// ───────── glob ─────────

func parseGlobs(s string) []*regexp.Regexp {
	var out []*regexp.Regexp
	depth, start := 0, 0
	flush := func(end int) {
		if g := strings.TrimSpace(s[start:end]); g != "" {
			if re, err := regexp.Compile(globToRegexp(g)); err == nil {
				out = append(out, re)
			}
		}
		start = end + 1
	}
	for i, r := range s { // 花括号里的逗号不拆
		switch r {
		case '{':
			depth++
		case '}':
			depth--
		case ',':
			if depth == 0 {
				flush(i)
			}
		}
	}
	flush(len(s))
	return out
}

func anyMatch(res []*regexp.Regexp, p string) bool {
	for _, re := range res {
		if re.MatchString(p) {
			return true
		}
	}
	return false
}

// globToRegexp:** 跨目录,* 不跨目录,? 单字符,{a,b} 择一;不含 / 的模式匹配任意层的文件名或目录名。
func globToRegexp(g string) string {
	var b strings.Builder
	for i := 0; i < len(g); i++ {
		switch c := g[i]; c {
		case '*':
			if i+1 < len(g) && g[i+1] == '*' {
				b.WriteString(".*")
				i++
				if i+1 < len(g) && g[i+1] == '/' {
					i++
					b.WriteString("(?:.*/)?") // **/ 也匹配零层
				}
			} else {
				b.WriteString("[^/]*")
			}
		case '?':
			b.WriteString("[^/]")
		case '{':
			b.WriteString("(?:")
		case '}':
			b.WriteString(")")
		case ',':
			b.WriteString("|")
		case '.', '+', '(', ')', '|', '^', '$', '[', ']', '\\':
			b.WriteString(`\` + string(c))
		default:
			b.WriteByte(c)
		}
	}
	body := b.String()
	if !strings.Contains(g, "/") {
		return `(?:^|/)` + body + `(?:$|/)`
	}
	return `^` + strings.TrimPrefix(body, "/") + `(?:$|/)`
}
