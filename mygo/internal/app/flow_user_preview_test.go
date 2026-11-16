package app

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// 折叠态只送前缀进排版:短文不动,长文按「6 个物理行 / 1000 字符」较小者截断,且不切坏多字节字符。
func TestUserFoldPreview(t *testing.T) {
	if got := userFoldPreview("你好\n世界"); got != "你好\n世界" {
		t.Fatalf("短文不应改动:%q", got)
	}
	big := strings.Repeat("日志行 abc 一二三四五六七八九十\n", 5000)
	got := userFoldPreview(big)
	if n := strings.Count(got, "\n"); n != userFoldMax-1 {
		t.Fatalf("应只留 %d 个物理行,得到换行数 %d", userFoldMax, n)
	}
	if !strings.HasPrefix(big, got) {
		t.Fatal("预览必须是原文的前缀")
	}
	one := strings.Repeat("字", 34101) // 单个超长行
	got = userFoldPreview(one)
	if utf8.RuneCountInString(got) != userPreviewRunes || !utf8.ValidString(got) {
		t.Fatalf("单行应截到 %d 字符且保持合法 UTF-8,得到 %d", userPreviewRunes, utf8.RuneCountInString(got))
	}
}

// estimatedUserLines 数到上限就停,结果与折叠阈值的判断一致;空串 / 空行仍各算一行。
func TestEstimatedUserLinesCap(t *testing.T) {
	if got := estimatedUserLines(strings.Repeat("x\n", 5000), 44); got != userLinesCap {
		t.Fatalf("应封顶在 %d,得到 %d", userLinesCap, got)
	}
	if got := estimatedUserLines("", 44); got != 1 {
		t.Fatalf("空串应为 1 行,得到 %d", got)
	}
	if got := estimatedUserLines("a\n\nb", 44); got != 3 {
		t.Fatalf("含空行应为 3 行,得到 %d", got)
	}
	if got := estimatedUserLines(strings.Repeat("中", 30), 44); got != 2 { // 30 个汉字 = 60 宽 → 2 行
		t.Fatalf("汉字按双倍宽折行,得到 %d", got)
	}
	if estimatedUserLines(strings.Repeat("字", 34101), 44) <= userFoldExp {
		t.Fatal("超长文字必须判为需要折叠 / 滚动")
	}
}
