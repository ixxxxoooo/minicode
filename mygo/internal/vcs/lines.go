package vcs

import "strings"

// MergeLineRange 把 src 里 [start,end] 行(含,0 起)替换进 base,用于行级暂存/还原。
func MergeLineRange(base, src string, start, end int) string {
	bl := strings.Split(base, "\n")
	sl := strings.Split(src, "\n")
	if end < start {
		return base
	}
	if start >= len(sl) {
		start = len(sl) - 1
	}
	if start < 0 {
		start = 0
	}
	if end >= len(sl) {
		end = len(sl) - 1
	}
	chunk := sl[start : end+1]
	if start >= len(bl) {
		return strings.Join(append(bl, chunk...), "\n")
	}
	out := append(append([]string{}, bl[:start]...), chunk...)
	if end+1 < len(bl) {
		out = append(out, bl[end+1:]...)
	}
	return strings.Join(out, "\n")
}
