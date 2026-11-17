//go:build !darwin || !cgo

package app

// platformFontCoverage 非 macOS 或禁用 cgo 时没有字形查询:按「已安装、覆盖未知」处理。
func platformFontCoverage(family string, cps []rune) (has []bool, installed bool) {
	return nil, true
}
