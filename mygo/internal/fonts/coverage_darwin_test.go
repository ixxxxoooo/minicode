//go:build darwin && cgo

package fonts

import "testing"

// 真的调 CoreText(其它测试都把字形查询换成了假实现,覆盖不到这里):
// Menlo 是每台 Mac 自带的等宽字体,必须查得到且有拉丁字形;不存在的字体必须识破 CoreText 的「悄悄换默认字体」。
func TestPlatformCoverageReal(t *testing.T) {
	ResetCoverage()
	has, installed, known := Has("Menlo", GlyphCats[0])
	if !known || !installed {
		t.Fatalf("Menlo 应被识别为已安装:installed=%v known=%v", installed, known)
	}
	if len(has) == 0 {
		t.Fatal("应返回逐码点结果")
	}
	for i, ok := range has {
		if !ok {
			t.Errorf("Menlo 应有拉丁字符的字形(第 %d 个码点没有)", i)
		}
	}

	if _, installed, known := Has("MiniCode 不存在的字体 9f3a", GlyphCats[0]); !known || installed {
		t.Fatalf("不存在的字体应判为未安装:installed=%v known=%v", installed, known)
	}

	// 补充平面码点(emoji 走代理对分支):Apple Color Emoji 应有 😀
	if h, inst, _ := Has("Apple Color Emoji", GlyphCat{ID: "t-emoji", Sample: "😀"}); !inst || len(h) != 1 || !h[0] {
		t.Errorf("Apple Color Emoji 应有 😀 的字形:installed=%v has=%v", inst, h)
	}

	// 整条链的结论:Menlo 自带拉丁字符
	if r := ChainCoverage([]string{"Menlo"}, GlyphCats[0]); r.Level != CovOwn {
		t.Errorf("Menlo 对拉丁字符应是 CovOwn,得到 %v", r.Level)
	}
}
