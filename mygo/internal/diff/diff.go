// Package diff 是按行的文本差异:最长公共子序列,输出带新旧行号的行序列,供工具卡 / 差异标签使用。
// 与界面无关,有单元测试。
package diff

import "strings"

// Kind 是一行的类别。
type Kind int

const (
	Same Kind = iota
	Add
	Del
)

// Line 是差异里的一行。OldNo / NewNo 为 0 表示该侧没有这一行。
type Line struct {
	Kind         Kind
	OldNo, NewNo int
	Text         string
}

// MaxLines 超过它就不再做 LCS(O(n·m)),退化成整段删除 + 整段新增,避免超大文本卡住界面。
const MaxLines = 3000

// Split 按行拆分,不保留末尾换行造成的空行。
func Split(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}

// Lines 计算 old → new 的行差异。startOld / startNew 是两侧第一行的行号(从 1 起,0 当 1)。
func Lines(old, new string, startOld, startNew int) []Line {
	a, b := Split(old), Split(new)
	if startOld < 1 {
		startOld = 1
	}
	if startNew < 1 {
		startNew = startOld
	}
	var out []Line
	if len(a) > MaxLines || len(b) > MaxLines {
		for i, t := range a {
			out = append(out, Line{Del, startOld + i, 0, t})
		}
		for i, t := range b {
			out = append(out, Line{Add, 0, startNew + i, t})
		}
		return out
	}
	// 先剪掉共同的头尾,缩小 DP 规模
	p := 0
	for p < len(a) && p < len(b) && a[p] == b[p] {
		p++
	}
	s := 0
	for s < len(a)-p && s < len(b)-p && a[len(a)-1-s] == b[len(b)-1-s] {
		s++
	}
	for i := 0; i < p; i++ {
		out = append(out, Line{Same, startOld + i, startNew + i, a[i]})
	}
	am, bm := a[p:len(a)-s], b[p:len(b)-s]
	// dp[i][j] = am[i:] 与 bm[j:] 的 LCS 长度
	dp := make([][]int32, len(am)+1)
	for i := range dp {
		dp[i] = make([]int32, len(bm)+1)
	}
	for i := len(am) - 1; i >= 0; i-- {
		for j := len(bm) - 1; j >= 0; j-- {
			if am[i] == bm[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else {
				dp[i][j] = max(dp[i+1][j], dp[i][j+1])
			}
		}
	}
	i, j := 0, 0
	on, nn := startOld+p, startNew+p
	for i < len(am) || j < len(bm) {
		switch {
		case i < len(am) && j < len(bm) && am[i] == bm[j]:
			out = append(out, Line{Same, on, nn, am[i]})
			i, j, on, nn = i+1, j+1, on+1, nn+1
		case i < len(am) && (j == len(bm) || dp[i+1][j] >= dp[i][j+1]):
			out = append(out, Line{Del, on, 0, am[i]})
			i, on = i+1, on+1
		default:
			out = append(out, Line{Add, 0, nn, bm[j]})
			j, nn = j+1, nn+1
		}
	}
	// 删除块排在新增块前面(与设计稿、VS Code 一致):同一处改动里先删后增
	for k := 0; k < len(out); k++ {
		if out[k].Kind != Add {
			continue
		}
		e := k
		for e < len(out) && out[e].Kind == Add {
			e++
		}
		d := e
		for d < len(out) && out[d].Kind == Del {
			d++
		}
		if d > e { // 增块后面紧跟删块:对调
			seg := append(append([]Line{}, out[e:d]...), out[k:e]...)
			copy(out[k:d], seg)
		}
		k = d - 1
	}
	for k := 0; k < s; k++ {
		out = append(out, Line{Same, on + k, nn + k, a[len(a)-s+k]})
	}
	return out
}

// Stat 统计新增 / 删除行数。
func Stat(ls []Line) (add, del int) {
	for _, l := range ls {
		switch l.Kind {
		case Add:
			add++
		case Del:
			del++
		}
	}
	return
}
