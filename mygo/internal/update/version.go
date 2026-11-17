package update

import (
	"strconv"
	"strings"
)

// parseVersion 取版本号开头的数字段(去掉前缀 v 与 -mygo / -beta.1 / +build 之类后缀)。
// 参数:v 版本字符串。返回值:数字段,如 "0.3.0-mygo" → [0 3 0];解析不出任何数字返回 nil。
func parseVersion(v string) []int {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexAny(v, "-+ "); i >= 0 {
		v = v[:i]
	}
	var out []int
	for _, p := range strings.Split(v, ".") {
		n, err := strconv.Atoi(p)
		if err != nil {
			break
		}
		out = append(out, n)
	}
	return out
}

// IsNewer 判断远端版本是否比当前版本新(逐段比较数字,缺的段按 0;后缀不参与比较)。
// 用途:只在「真的更新」时提示,发布源回退到旧版本或与当前相同都不算有更新。
// 参数:remote 发布源里的版本,cur 当前版本。返回值:remote 严格新于 cur 为真;任一方解析失败时退回「不同即更新」。
func IsNewer(remote, cur string) bool {
	a, b := parseVersion(remote), parseVersion(cur)
	if len(a) == 0 || len(b) == 0 {
		return remote != cur
	}
	for i := 0; i < len(a) || i < len(b); i++ {
		var x, y int
		if i < len(a) {
			x = a[i]
		}
		if i < len(b) {
			y = b[i]
		}
		if x != y {
			return x > y
		}
	}
	return false
}
