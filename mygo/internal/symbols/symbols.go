// Package symbols 是符号索引:没有接语言服务器时,用「按行的正则 + 括号 / 缩进配对」从源码里取出
// 函数、类型、实现块等声明。它支撑大纲、面包屑、粘性滚动、转到定义、引用、悬停、补全、重命名;
// 精度比 LSP 低(不懂作用域与类型),界面上不假装是语言服务器。纯函数,不依赖界面。
package symbols

import (
	"path"
	"regexp"
	"strings"
)

// Symbol 是一处声明。
type Symbol struct {
	Name   string
	Kind   string // fn / struct / enum / trait / impl / mod / type / const / class / h(标题)
	Line   int    // 声明所在行(0 起)
	Col    int    // 名字起始列(按字节)
	End    int    // 声明块的最后一行(含)
	Parent int    // 外层符号在切片里的下标,-1 = 顶层
	Detail string // 声明那一行去掉缩进后的文本(悬停的签名)
}

var (
	reRust    = regexp.MustCompile(`^\s*(?:pub(?:\([^)]*\))?\s+)?(?:async\s+)?(?:unsafe\s+)?(?:const\s+)?(fn|struct|enum|trait|mod|type|static)\s+(\w+)`)
	reRustImp = regexp.MustCompile(`^\s*(?:unsafe\s+)?impl(?:<[^>]*>)?\s+(?:[\w:]+(?:<[^>]*>)?\s+for\s+)?(\w+)`)
	reRustCon = regexp.MustCompile(`^(?:pub(?:\([^)]*\))?\s+)?const\s+(\w+)\s*:`)
	reGoFn    = regexp.MustCompile(`^func\s+(?:\([^)]*\)\s*)?(\w+)`)
	reGoType  = regexp.MustCompile(`^type\s+(\w+)\s+(struct|interface)?`)
	reGoVar   = regexp.MustCompile(`^(?:const|var)\s+(\w+)\s`)
	reTS      = regexp.MustCompile(`^\s*(?:export\s+)?(?:default\s+)?(?:async\s+)?(function|class|interface|enum|type)\s+(\w+)`)
	reTSConst = regexp.MustCompile(`^\s*(?:export\s+)?const\s+(\w+)\s*=\s*(?:async\s*)?(?:\([^)]*\)|\w+)\s*=>`)
	rePy      = regexp.MustCompile(`^(\s*)(def|class)\s+(\w+)`)
	// MarkdownHeading 匹配 Markdown 标题行(# 一级到 ###### 六级);大纲与 Markdown 导航共用。
	MarkdownHeading = regexp.MustCompile(`^(#{1,6})\s+(.+?)\s*#*$`)
	reToml          = regexp.MustCompile(`^\[+([^\]]+)\]+`)
)

// Lang 按扩展名决定用哪套规则("" = 不支持)。
func Lang(p string) string {
	switch strings.ToLower(path.Ext(p)) {
	case ".rs":
		return "rust"
	case ".go":
		return "go"
	case ".ts", ".tsx", ".js", ".jsx", ".mjs":
		return "ts"
	case ".py":
		return "py"
	case ".md", ".mdc", ".markdown":
		return "md"
	case ".toml":
		return "toml"
	}
	return ""
}

// Parse 取出一份源码里的全部声明(按出现顺序),并算出每个声明块的结束行与外层关系。
func Parse(p, src string) []Symbol {
	lang := Lang(p)
	if lang == "" {
		return nil
	}
	lines := strings.Split(src, "\n")
	var out []Symbol
	add := func(name, kind string, line, col int) {
		out = append(out, Symbol{Name: name, Kind: kind, Line: line, Col: col, End: line, Parent: -1,
			Detail: strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(lines[line]), "{"))})
	}
	for i, ln := range lines {
		switch lang {
		case "rust":
			if m := reRustImp.FindStringSubmatchIndex(ln); m != nil {
				add("impl "+ln[m[2]:m[3]], "impl", i, m[2])
			} else if m := reRust.FindStringSubmatchIndex(ln); m != nil {
				add(ln[m[4]:m[5]], ln[m[2]:m[3]], i, m[4])
			} else if m := reRustCon.FindStringSubmatchIndex(ln); m != nil {
				add(ln[m[2]:m[3]], "const", i, m[2])
			}
		case "go":
			if m := reGoFn.FindStringSubmatchIndex(ln); m != nil {
				add(ln[m[2]:m[3]], "fn", i, m[2])
			} else if m := reGoType.FindStringSubmatchIndex(ln); m != nil {
				kind := "type"
				if m[4] >= 0 {
					kind = ln[m[4]:m[5]]
				}
				add(ln[m[2]:m[3]], kind, i, m[2])
			} else if m := reGoVar.FindStringSubmatchIndex(ln); m != nil {
				add(ln[m[2]:m[3]], "const", i, m[2])
			}
		case "ts":
			if m := reTS.FindStringSubmatchIndex(ln); m != nil {
				add(ln[m[4]:m[5]], ln[m[2]:m[3]], i, m[4])
			} else if m := reTSConst.FindStringSubmatchIndex(ln); m != nil {
				add(ln[m[2]:m[3]], "fn", i, m[2])
			}
		case "py":
			if m := rePy.FindStringSubmatchIndex(ln); m != nil {
				k := "fn"
				if ln[m[4]:m[5]] == "class" {
					k = "class"
				}
				add(ln[m[6]:m[7]], k, i, m[6])
			}
		case "md":
			if m := MarkdownHeading.FindStringSubmatchIndex(ln); m != nil {
				add(ln[m[4]:m[5]], "h", i, m[4])
				out[len(out)-1].Detail = ln[m[2]:m[3]] // 标题级别(#、## …)
			}
		case "toml":
			if m := reToml.FindStringSubmatchIndex(ln); m != nil {
				add(ln[m[2]:m[3]], "mod", i, m[2])
			}
		}
	}
	// 结束行
	switch lang {
	case "py":
		for k := range out {
			out[k].End = pyEnd(lines, out[k].Line)
		}
	case "md":
		for k := range out {
			lv := len(out[k].Detail)
			out[k].End = len(lines) - 1
			for j := k + 1; j < len(out); j++ {
				if len(out[j].Detail) <= lv {
					out[k].End = out[j].Line - 1
					break
				}
			}
		}
	case "toml":
		for k := range out {
			out[k].End = len(lines) - 1
			if k+1 < len(out) {
				out[k].End = out[k+1].Line - 1
			}
		}
	default:
		for k := range out {
			out[k].End = braceEnd(lines, out[k].Line)
		}
	}
	// 外层关系:行范围包含它、且开始更早的最近一个符号
	for k := range out {
		for j := k - 1; j >= 0; j-- {
			if out[j].Line < out[k].Line && out[j].End >= out[k].End && out[j].Kind != "const" {
				out[k].Parent = j
				break
			}
		}
	}
	return out
}

// braceEnd 从 start 行起配对花括号,返回块的最后一行;一直没遇到 { 的单行声明(如 `struct A;`)就是自己。
func braceEnd(lines []string, start int) int {
	depth, seen := 0, false
	for i := start; i < len(lines) && i < start+4000; i++ {
		ln := lines[i]
		inStr := byte(0)
		for j := 0; j < len(ln); j++ {
			ch := ln[j]
			switch {
			case inStr != 0:
				if ch == '\\' {
					j++
				} else if ch == inStr {
					inStr = 0
				}
			case ch == '"' || ch == '`':
				inStr = ch
			case ch == '/' && j+1 < len(ln) && ln[j+1] == '/':
				j = len(ln)
			case ch == '{':
				depth++
				seen = true
			case ch == '}':
				depth--
			case ch == ';' && !seen && depth == 0:
				return i // 没有块体的声明
			}
		}
		if seen && depth <= 0 {
			return i
		}
		if !seen && i >= start+3 {
			return start
		}
	}
	return start
}

// pyEnd 按缩进找 Python 块的结束行。
func pyEnd(lines []string, start int) int {
	ind := IndentOf(lines[start])
	end := start
	for i := start + 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "" {
			continue
		}
		if IndentOf(lines[i]) <= ind {
			break
		}
		end = i
	}
	return end
}

// IndentOf 返回行首缩进宽度:空格算 1,制表符算 4,遇到第一个非空白字符停。
func IndentOf(s string) int {
	n := 0
	for _, r := range s {
		switch r {
		case ' ':
			n++
		case '\t':
			n += 4
		default:
			return n
		}
	}
	return n
}

// Enclosing 返回包住第 line 行的符号链(从外到内的下标)。
func Enclosing(syms []Symbol, line int) []int {
	best := -1
	for i, s := range syms {
		if s.Line <= line && line <= s.End && s.Kind != "const" {
			if best < 0 || s.Line >= syms[best].Line {
				best = i
			}
		}
	}
	var chain []int
	for k := best; k >= 0; k = syms[k].Parent {
		chain = append([]int{k}, chain...)
	}
	return chain
}

// ───────── 单词 ─────────

func IsWordByte(b byte) bool {
	return b == '_' || b >= '0' && b <= '9' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= 0x80
}

// WordAt 返回 s 里第 col 列(字节)所在的单词及其起止;不在单词上返回空。
func WordAt(s string, col int) (string, int, int) {
	if col > len(s) {
		col = len(s)
	}
	if col < 0 {
		return "", 0, 0
	}
	a, b := col, col
	for a > 0 && IsWordByte(s[a-1]) {
		a--
	}
	for b < len(s) && IsWordByte(s[b]) {
		b++
	}
	if a == b {
		return "", 0, 0
	}
	return s[a:b], a, b
}

// WordHits 返回 word 作为整词在 s 里的全部起始列。
func WordHits(s, word string) []int {
	if word == "" {
		return nil
	}
	var out []int
	for from := 0; ; {
		i := strings.Index(s[from:], word)
		if i < 0 {
			return out
		}
		i += from
		end := i + len(word)
		if (i == 0 || !IsWordByte(s[i-1])) && (end >= len(s) || !IsWordByte(s[end])) {
			out = append(out, i)
		}
		from = end
	}
}

// IsKeyword 判断是不是各语言的关键字(悬停 / 引用 / 重命名不处理它们)。
func IsKeyword(w string) bool { return keywordSet[w] }

var keywordSet = func() map[string]bool {
	m := map[string]bool{}
	for _, k := range strings.Fields(`fn let mut pub use mod struct enum impl trait for in if else match while loop return self Self super crate as
		const static type where async await move ref unsafe dyn true false break continue
		func var package import interface map chan go defer range select switch case default nil
		function class export from new this typeof extends def lambda None True False pass with try except finally raise`) {
		m[k] = true
	}
	return m
}()
