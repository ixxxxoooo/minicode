#ifndef FONT_COV_DARWIN_H
#define FONT_COV_DARWIN_H

#include <stdint.h>

// mc_font_coverage 查某个字体族对一组码点有没有字形。
// 返回 -1 = 系统里没有这个字体(CoreText 会悄悄换成默认字体,这里要识破);0 = 查询完成,out[i] = 1 表示有字形。
int mc_font_coverage(const char *family, const uint32_t *cps, int n, unsigned char *out);

#endif
