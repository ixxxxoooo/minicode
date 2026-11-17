//go:build darwin && cgo

package app

/*
#cgo darwin LDFLAGS: -framework Foundation -framework CoreText
#include <stdlib.h>
#include "font_cov_darwin.h"
*/
import "C"
import "unsafe"

// @author ixxxxoooo

// platformFontCoverage 用 CoreText 查字体对一组码点有没有字形。
// 参数:family 字体族名;cps 码点。
// 返回:has[i] 为第 i 个码点是否有字形;installed 为 false 表示系统里没有这个字体。
func platformFontCoverage(family string, cps []rune) (has []bool, installed bool) {
	if len(cps) == 0 {
		return nil, true
	}
	cf := C.CString(family)
	defer C.free(unsafe.Pointer(cf))
	in := make([]C.uint32_t, len(cps))
	for i, r := range cps {
		in[i] = C.uint32_t(r)
	}
	out := make([]C.uchar, len(cps))
	if C.mc_font_coverage(cf, &in[0], C.int(len(cps)), &out[0]) < 0 {
		return nil, false
	}
	has = make([]bool, len(cps))
	for i := range out {
		has[i] = out[i] != 0
	}
	return has, true
}
