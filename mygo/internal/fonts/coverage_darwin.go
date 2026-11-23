//go:build darwin && cgo

package fonts

// Objective-C 写在 cgo 序言里(和 internal/macos 一致),不单独放 .m 文件:
// Go 工具链会给每个含 .m 文件的包追加 -lobjc,两个包都有 .m 时 macOS 链接器会报
// 「ignoring duplicate libraries: '-lobjc'」。这里沿用原 .m 的编译方式:不开 ARC(-fobjc-arc),
// __bridge 在非 ARC 下是空操作,@autoreleasepool 照常回收临时对象。

/*
#cgo darwin CFLAGS: -x objective-c
#cgo darwin LDFLAGS: -framework Foundation -framework CoreText
#import <Foundation/Foundation.h>
#import <CoreText/CoreText.h>
#include <stdint.h>
#include <stdlib.h>

// 规范化字体名:小写并去掉空格 / 连字符 / 下划线,和 Go 侧 normFont 同口径。
static NSString *mc_norm(NSString *s) {
  if (!s) return @"";
  NSString *l = [s lowercaseString];
  for (NSString *ch in @[ @" ", @"-", @"_" ]) {
    l = [l stringByReplacingOccurrencesOfString:ch withString:@""];
  }
  return l;
}

// mc_font_coverage 查某个字体族对一组码点有没有字形。
// 返回 -1 = 系统里没有这个字体(CoreText 会悄悄换成默认字体,这里要识破);0 = 查询完成,out[i] = 1 表示有字形。
static int mc_font_coverage(const char *family, const uint32_t *cps, int n, unsigned char *out) {
  @autoreleasepool {
    NSString *want = [NSString stringWithUTF8String:family];
    CTFontRef f = CTFontCreateWithName((__bridge CFStringRef)want, 12, NULL);
    if (!f) return -1;
    // CoreText 找不到字体时不报错而是换成默认字体:比对族名 / 全名 / PostScript 名,都对不上就当没装
    NSString *w = mc_norm(want);
    BOOL match = NO;
    CFStringRef names[3] = {CTFontCopyFamilyName(f), CTFontCopyFullName(f), CTFontCopyPostScriptName(f)};
    for (int i = 0; i < 3; i++) {
      if (names[i]) {
        if ([mc_norm((__bridge NSString *)names[i]) isEqualToString:w]) match = YES;
        CFRelease(names[i]);
      }
    }
    if (!match) {
      CFRelease(f);
      return -1;
    }
    for (int i = 0; i < n; i++) {
      uint32_t cp = cps[i];
      UniChar u[2];
      CGGlyph g[2] = {0, 0};
      CFIndex len = 1;
      if (cp > 0xFFFF) { // 补充平面:代理对,第一个字形非零即有
        uint32_t v = cp - 0x10000;
        u[0] = (UniChar)(0xD800 + (v >> 10));
        u[1] = (UniChar)(0xDC00 + (v & 0x3FF));
        len = 2;
      } else {
        u[0] = (UniChar)cp;
      }
      CTFontGetGlyphsForCharacters(f, u, g, len);
      out[i] = g[0] != 0 ? 1 : 0;
    }
    CFRelease(f);
    return 0;
  }
}
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
