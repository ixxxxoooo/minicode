package app

import (
	"strings"
	"testing"
	"time"
)

// 回归:预览里富文本编辑器是自绘的,库的 Clip 只裁子元素、不裁自己的 Draw,
// 滚动后部分可见的段落会整段画到编辑区外面,盖住上面的提示条与下面的底部面板。
func TestMdPreviewDoesNotPaintOutsideEditor(t *testing.T) {
	a, tt := demoIDE(t)
	tt.Advance(time.Second)
	d := a.ide
	d.openFile("docs/guide.md", true)
	tt.Frame()
	tb := d.cur()
	var sb strings.Builder
	sb.WriteString("# 长文档\n\n")
	for i := 0; i < 40; i++ {
		sb.WriteString("## 小节 标题很长很长很长很长很长很长很长很长很长很长很长很长很长很长\n\n")
		sb.WriteString("1. 第一项:这是一段足够长的说明文字,用来让列表项换行成好几行,这样滚动停在中间时一定会有被切到的段落。\n")
		sb.WriteString("   - 嵌套项:这是一段足够长的说明文字,用来让列表项换行成好几行,这样滚动停在中间时一定会有被切到的段落。\n")
		sb.WriteString("   - 嵌套项二:同样很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长。\n\n")
	}
	tb.buf.Set(sb.String())
	tt.Frame()
	tt.Advance(time.Second)
	tt.Frame()
	info, ok := tt.Access("Markdown 文档")
	if !ok || info.Bounds.H < 100 {
		t.Fatalf("找不到预览编辑器:%+v ok=%v", info, ok)
	}
	b := info.Bounds
	for step := 0; step < 12; step++ {
		tt.Wheel(b.X+b.W/2, b.Y+b.H/2, 0, 37)
		tt.Advance(120 * time.Millisecond)
		tt.Frame()
		img := tt.Image()
		// 编辑器上沿之上 / 下沿之下紧挨的几行像素(都在外层内边距里)应是纯背景,不能有文字
		strip := func(y0, y1 int) bool {
			var first uint32
			for y := y0; y < y1; y++ {
				for x := int(b.X) + 20; x < int(b.X+b.W)-40; x++ {
					r, g, bl, _ := img.At(x, y).RGBA()
					c := r<<16 ^ g<<8 ^ bl
					if y == y0 && x == int(b.X)+20 {
						first = c
					}
					if c != first {
						return false
					}
				}
			}
			return true
		}
		top, bot := int(b.Y), int(b.Y+b.H)
		if !strip(top-8, top-1) {
			t.Fatalf("第 %d 次滚动后,文字画到了编辑器上沿之外", step)
		}
		if !strip(bot+1, bot+5) {
			t.Fatalf("第 %d 次滚动后,文字画到了编辑器下沿之外", step)
		}
	}
}
