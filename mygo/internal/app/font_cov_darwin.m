#import <Foundation/Foundation.h>
#import <CoreText/CoreText.h>
#include "font_cov_darwin.h"

// 规范化字体名:小写并去掉空格 / 连字符 / 下划线,和 Go 侧 normFont 同口径。
static NSString *mc_norm(NSString *s) {
  if (!s) return @"";
  NSString *l = [s lowercaseString];
  for (NSString *ch in @[ @" ", @"-", @"_" ]) {
    l = [l stringByReplacingOccurrencesOfString:ch withString:@""];
  }
  return l;
}

int mc_font_coverage(const char *family, const uint32_t *cps, int n, unsigned char *out) {
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
