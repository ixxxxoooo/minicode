package app

import "testing"

// 正文与图片附件拆分 / 合成互为逆运算,输入框里不出现图片语法。
// @author ixxxxoooo
func TestSplitComposeDraft(t *testing.T) {
	for _, body := range []string{"", "x", "第一行\n第二行", "末尾有换行\n"} {
		for _, toks := range [][]string{nil, {"![](/a.png)"}, {"![](/a.png)", "![](b/c.jpg)"}} {
			full := composeDraft(body, toks)
			gb, gt := splitDraftBody(full)
			if gb != body || len(gt) != len(toks) {
				t.Fatalf("body=%q toks=%v → full=%q → 拆出 %q %v", body, toks, full, gb, gt)
			}
			for i := range toks {
				if gt[i] != toks[i] {
					t.Fatalf("附件顺序 / 内容不符:%v vs %v", gt, toks)
				}
			}
		}
	}
}

// 输入框正文不含图片语法;草稿(发送用)含;移除一张只去掉它。
func TestComposerBodyHidesImageTokens(t *testing.T) {
	a, tt := sessionApp(t, "flow-gaps-b4")
	a.draft = "看看这个\n![](/tmp/a.png)\n![](/tmp/b.png)"
	tt.Frame()
	if a.draftBody != "看看这个" || len(a.draftImgs) != 2 {
		t.Fatalf("正文应只有文字、附件 2 张:%q %v", a.draftBody, a.draftImgs)
	}
	if tt.HasText("![](") {
		t.Fatal("输入框里不应露出图片语法")
	}
	a.removeDraftAttachPath("/tmp/a.png")
	tt.Frame()
	if a.draftBody != "看看这个" || len(a.draftImgs) != 1 || a.draftImgs[0] != "![](/tmp/b.png)" {
		t.Fatalf("只应移除 a.png:%q %v", a.draftBody, a.draftImgs)
	}
	if got := a.draftNow(); got != "看看这个\n![](/tmp/b.png)" {
		t.Fatalf("发送用的草稿不对:%q", got)
	}
	// 用户在输入框里改了正文 → 合回 a.draft,附件不丢
	a.draftBody = "改过了"
	tt.Frame()
	if a.draft != "改过了\n![](/tmp/b.png)" {
		t.Fatalf("正文改动应合回草稿:%q", a.draft)
	}
}
