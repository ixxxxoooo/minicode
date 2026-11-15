package claude

import (
	"encoding/json"
	"strconv"
	"strings"
)

// 对 map[string]any 的宽松取值:字段缺失或类型不对一律返回零值,协议里的可选字段多,不必处处判错。

func str(m obj, k string) string {
	s, _ := m[k].(string)
	return s
}

func boolOf(m obj, k string, def bool) bool {
	if b, ok := m[k].(bool); ok {
		return b
	}
	return def
}

func num(m obj, k string) (uint64, bool) {
	switch x := m[k].(type) {
	case float64:
		if x < 0 {
			return 0, true
		}
		return uint64(x), true
	case json.Number:
		n, err := x.Int64()
		return uint64(max(n, 0)), err == nil
	}
	return 0, false
}

func u64(m obj, k string) uint64 { n, _ := num(m, k); return n }

func flt(m obj, k string) (float64, bool) {
	f, ok := m[k].(float64)
	return f, ok
}

func sub(m obj, k string) obj {
	o, _ := m[k].(obj)
	return o
}

func arr(m obj, k string) []any {
	a, _ := m[k].([]any)
	return a
}

func asObj(v any) obj {
	o, _ := v.(obj)
	return o
}

// parseLine 解析一行 JSON;失败返回 nil。
func parseLine(line string) obj {
	var m obj
	if err := json.Unmarshal([]byte(line), &m); err != nil {
		return nil
	}
	return m
}

// truncate 按字符数截断。
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "\n…(已截断)"
}

// firstLine 取第一行并限制 80 字符。
func firstLine(s string) string {
	line := strings.TrimSpace(strings.SplitN(s, "\n", 2)[0])
	if r := []rune(line); len(r) > 80 {
		return string(r[:80]) + "…"
	}
	return line
}

func itoa(n int) string { return strconv.Itoa(n) }

// tagText 取 <tag>…</tag> 之间的文本。
func tagText(text, tag string) string {
	open, closeT := "<"+tag+">", "</"+tag+">"
	i := strings.Index(text, open)
	if i < 0 {
		return ""
	}
	rest := text[i+len(open):]
	if j := strings.Index(rest, closeT); j >= 0 {
		rest = rest[:j]
	}
	return strings.TrimSpace(rest)
}

// partialJSONString 从不完整的 JSON 片段里取出字符串字段 key 已到达的部分(处理转义,半截转义留到下次);
// 字段还没出现返回 false。
func partialJSONString(partial, key string) (string, bool) {
	needle := `"` + key + `"`
	from := 0
	start := -1
	for from < len(partial) {
		i := strings.Index(partial[from:], needle)
		if i < 0 {
			return "", false
		}
		at := from + i + len(needle)
		rest := strings.TrimLeft(partial[at:], " \t\r\n")
		if strings.HasPrefix(rest, ":") {
			r := strings.TrimLeft(rest[1:], " \t\r\n")
			if strings.HasPrefix(r, `"`) {
				start = len(partial) - len(r) + 1
				break
			}
		}
		from = at
	}
	if start < 0 {
		return "", false
	}
	var out strings.Builder
	rs := []rune(partial[start:])
	for i := 0; i < len(rs); i++ {
		c := rs[i]
		switch c {
		case '"':
			return out.String(), true
		case '\\':
			i++
			if i >= len(rs) {
				return out.String(), true
			}
			switch e := rs[i]; e {
			case 'n':
				out.WriteRune('\n')
			case 't':
				out.WriteRune('\t')
			case 'r':
				out.WriteRune('\r')
			case 'b':
				out.WriteRune('\b')
			case 'f':
				out.WriteRune('\f')
			case 'u':
				code, n := hex4(rs, i+1)
				if n == 0 {
					return out.String(), true
				}
				i += n
				if code >= 0xD800 && code < 0xDC00 { // 代理对:后面必须跟 \uDC00..DFFF 才能拼成一个字符
					if i+2 < len(rs) && rs[i+1] == '\\' && rs[i+2] == 'u' {
						lo, n2 := hex4(rs, i+3)
						if n2 == 0 || lo < 0xDC00 || lo >= 0xE000 {
							return out.String(), true
						}
						code = 0x10000 + (code-0xD800)<<10 + (lo - 0xDC00)
						i += 2 + n2
					} else {
						return out.String(), true
					}
				}
				out.WriteRune(rune(code))
			default:
				out.WriteRune(e)
			}
		default:
			out.WriteRune(c)
		}
	}
	return out.String(), true
}

// hex4 读 rs[at:] 起的 4 位十六进制;不足 4 位返回 n=0。
func hex4(rs []rune, at int) (rune, int) {
	if at+4 > len(rs) {
		return 0, 0
	}
	v, err := strconv.ParseUint(string(rs[at:at+4]), 16, 32)
	if err != nil {
		return 0, 0
	}
	return rune(v), 4
}
