package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/richtext"
	"github.com/ixxxxoooo/minicode/internal/demo"
)

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
	if a.draftBody != "看看这个" || len(a.draftImgs) != 2 || textToDraft(a.compDoc.Text()) != "看看这个" {
		t.Fatalf("输入框正文应只有文字、附件 2 张:%q %v", a.draftBody, a.draftImgs)
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
	a.compDoc.SetText(richtext.NewText("改过了"))
	tt.Frame()
	if a.draft != "改过了\n![](/tmp/b.png)" {
		t.Fatalf("正文改动应合回草稿:%q", a.draft)
	}
}

// 带胶囊的草稿在输入框里按 Enter 发送:胶囊展开成 @路径(不带私有标记),发完输入框清空。
func TestComposerEnterSendsPillsAsAtPaths(t *testing.T) {
	a, tt, fb := pendingApp(t, demo.FixBug(false))
	a.draft = "看 " + atomString(atomFile, "src/a.rs") + " 和 " + atomString(atomDir, "docs") + " 吧"
	a.ux.focusDraft = true
	tt.Frame()
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if got := fb.last(); !isPrompt(got, "看 @src/a.rs 和 @docs/ 吧", false) {
		t.Fatalf("发送应把胶囊展开成 @路径:%v", got)
	}
	tt.Frame()
	if a.draft != "" || textToDraft(a.compDoc.Text()) != "" {
		t.Fatalf("发送后输入框应清空:%q", a.draft)
	}
}

// 带括号 / 百分号 / 中文的文件名:图片附件语法不能被截断,缩略图卡与发送拿到的都是完整原路径。
func TestDraftImagePathWithParens(t *testing.T) {
	for _, p := range []string{"/Users/x/Downloads/舆图志-陕西省 (3).png", "/a/b%c (1)).png", "/plain/渭南市.png"} {
		a, tt := sessionApp(t, "flow-gaps-b4")
		a.draft = ""
		tt.Frame()
		if !a.addDraftImage(p) {
			t.Fatalf("应能加入:%q", p)
		}
		tt.Frame()
		// 图片会在光标处插入一枚位置胶囊(正文里是原子标记串,见 addDraftImage),这是正常的;
		// 要保护的是:路径不能以明文漏进正文,胶囊必须恰好一枚且指向原路径。
		if rest := atomRE.ReplaceAllString(a.draftBody, ""); rest != "" {
			t.Fatalf("路径不应以明文漏进输入框正文:%q(%q)", a.draftBody, p)
		}
		if pills := imgAtomPaths(a.draftBody); len(pills) != 1 || !pills[p] {
			t.Fatalf("正文里应恰有一枚指向原路径的图片胶囊:%v(%q)", pills, p)
		}
		atts := parseDraftAttaches(a.draftNow())
		if len(atts) != 1 || atts[0].Path != p {
			t.Fatalf("解析出的路径应与原路径一致:%v(%q)", atts, p)
		}
		if _, imgs := splitDraftImages(a.draftNow()); len(imgs) != 1 || imgs[0] != p {
			t.Fatalf("发送拆出的图片路径不对:%v(%q)", imgs, p)
		}
		a.removeDraftAttachPath(p)
		tt.Frame()
		if len(a.draftImgs) != 0 {
			t.Fatalf("移除后应没有附件:%v", a.draftImgs)
		}
	}
}
