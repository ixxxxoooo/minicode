package app

import (
	"path"
	"regexp"
	"strings"
	"unicode/utf8"
)

// refRowKind 是引用面板拍平列表中的一行种类。
type refRowKind int

const (
	refRowFile refRowKind = iota
	refRowHit
)

// refFlatRow 是引用面板 List 里的一行(文件组头或一条引用)。
type refFlatRow struct {
	kind refRowKind
	path string // 文件行:路径;命中行:所属路径
	ix   int    // 命中行:items 下标;-1 = 组头
}

// refFileCount 统计引用里有多少个不同文件。
func refFileCount(items []refItem) int {
	seen := map[string]bool{}
	for _, it := range items {
		seen[it.path] = true
	}
	return len(seen)
}

// refFlatten 把 items 按 path 分组后拍平成行;folded 里为 true 的路径只保留组头。
func refFlatten(items []refItem, folded map[string]bool) []refFlatRow {
	if len(items) == 0 {
		return nil
	}
	var rows []refFlatRow
	last := ""
	for i, it := range items {
		if it.path != last {
			last = it.path
			rows = append(rows, refFlatRow{kind: refRowFile, path: it.path, ix: -1})
			if folded[it.path] {
				continue
			}
		}
		if !folded[it.path] {
			rows = append(rows, refFlatRow{kind: refRowHit, path: it.path, ix: i})
		}
	}
	return rows
}

// refItemKey 是引用行的稳定键(选中、同步光标用)。
func refItemKey(it refItem, word string) string {
	return it.path + ":" + itoa(it.line) + ":" + itoa(it.col) + ":" + word
}

// refFindSel 在 items 里找与给定文件行列最接近的一条(含命中词),用于与编辑器同步。
func refFindSel(items []refItem, word, file string, line, col int) int {
	best, dist := -1, 1<<30
	for i, it := range items {
		if it.path != file {
			continue
		}
		d := refAbs(it.line-line)*1000 + refAbs(it.col-col)
		if d < dist {
			dist, best = d, i
		}
	}
	if best < 0 {
		return -1
	}
	// 光标仍须在命中词附近才算同一条引用
	it := items[best]
	wl := len(word)
	if col < it.col-wl || col > it.col+wl {
		return -1
	}
	return best
}

func refAbs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

var (
	refDefRe  = regexp.MustCompile(`(?i)\b(?:fn|func|def|class|type|struct|interface|enum|trait|impl)\s+`)
	refTestRe = regexp.MustCompile(`(?i)_test\.(go|rs)|\.test\.(ts|tsx|js|jsx)$|_spec\.(rb|js|ts)$`)
)

// refIsTestFile 判断路径是否像测试文件。
func refIsTestFile(p string) bool {
	base := strings.ToLower(path.Base(p))
	if strings.HasSuffix(base, "_test.go") {
		return true
	}
	return refTestRe.MatchString(base)
}

// refLooksLikeDef 用启发式判断该行是否像符号定义(语言服务器未区分时的诚实标注)。
func refLooksLikeDef(line, word string) bool {
	t := strings.TrimSpace(line)
	if t == "" || word == "" {
		return false
	}
	if !strings.Contains(t, word) {
		return false
	}
	rest := t
	if i := strings.Index(rest, word); i >= 0 {
		rest = strings.TrimSpace(rest[:i])
	}
	return refDefRe.MatchString(rest + " ")
}

// refPreviewBytes 返回截断后的可见文本与命中在其中的字节区间(对标 search.Preview,以命中为中心)。
func refPreviewBytes(raw string, hitLo, hitHi, maxRunes int) (text string, lo, hi int) {
	if raw == "" {
		return "", 0, 0
	}
	rs := []rune(raw)
	if hitLo > len(rs) {
		hitLo = len(rs)
	}
	if hitHi > len(rs) {
		hitHi = len(rs)
	}
	if hitLo > hitHi {
		hitLo = hitHi
	}
	lead := 0
	for lead < hitLo && (rs[lead] == ' ' || rs[lead] == '\t') {
		lead++
	}
	rs, hitLo, hitHi = rs[lead:], hitLo-lead, hitHi-lead
	hit := string(rs[hitLo:hitHi])
	before, after := rs[:hitLo], rs[hitHi:]
	room := maxRunes - utf8.RuneCountInString(hit)
	if room < 8 {
		room = 8
	}
	if len(before) > room/2 {
		skip := len(before) - room/2
		before = append([]rune("…"), before[skip:]...)
	}
	room -= len(before)
	if len(after) > room {
		after = append(append([]rune{}, after[:max(room, 0)]...), '…')
	}
	visRunes := append(append([]rune{}, before...), append([]rune(hit), after...)...)
	vis := string(visRunes)
	visHitLo := len(before)
	lo = runeIndexToByte(vis, visHitLo)
	hi = runeIndexToByte(vis, visHitLo+utf8.RuneCountInString(hit))
	return vis, lo, hi
}

func runeIndexToByte(s string, ri int) int {
	if ri <= 0 {
		return 0
	}
	n := 0
	for i := range s {
		if n == ri {
			return i
		}
		n++
	}
	return len(s)
}
