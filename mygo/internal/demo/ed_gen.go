package demo

import (
	"fmt"
	"strings"
)

// genPrefixedLines 生成 n 行重复文本,用于大文件与日志。
func genPrefixedLines(prefix string, n int) string {
	var b strings.Builder
	b.Grow(n * (len(prefix) + 16))
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, "%s %d normalize\n", prefix, i)
	}
	return b.String()
}

// genSearchHits 单文件 6000 处 normalize(搜索「大量结果 / 上限」演示)。
func genSearchHits() string {
	return genPrefixedLines("normalize", 6000)
}

// genBulkFiles 在 bulk/many/ 下生成 count 个小文件。
func genBulkFiles() map[string]string {
	out := make(map[string]string, 1500)
	for i := 1; i <= 1500; i++ {
		p := fmt.Sprintf("bulk/many/file_%04d.txt", i)
		out[p] = fmt.Sprintf("# 批量文件 %d\nDEMO_SEARCH normalize\n", i)
	}
	return out
}

// genBigJSON 约 200KB 的 JSON 数组。
func genBigJSON() string {
	var b strings.Builder
	b.WriteString("[\n")
	n := 0
	for b.Len() < 200<<10 {
		if n > 0 {
			b.WriteString(",\n")
		}
		fmt.Fprintf(&b, `  {"id":%d,"tag":"normalize","payload":"demo-%d"}`, n, n)
		n++
	}
	b.WriteString("\n]\n")
	return b.String()
}

// genLongLine 生成一行 len 字符的「压缩」JS。
func genLongLine(n int) string {
	var b strings.Builder
	b.Grow(n + 64)
	b.WriteString("// DEMO minified one-liner\n")
	b.WriteString("function normalize(a){return a}")
	for b.Len() < n+40 {
		b.WriteString(";normalize(1)")
	}
	s := b.String()
	if len(s) > n {
		return s[:n]
	}
	return s
}

// deepPath 10 层目录下的叶子文件。
func deepPathLeaf() string {
	return "deep/l1/l2/l3/l4/l5/l6/l7/l8/l9/l10/leaf.txt"
}
