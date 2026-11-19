package app

import (
	"strings"
	"testing"
)

// 预览只取「最新的一句」:按换行 / 句末标点切,刚写完一句时取的还是这一句。
func TestThinkLastLine(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"先看路由表", "先看路由表"},
		{"先看路由表。再看匹配逻辑", "再看匹配逻辑"},
		{"先看路由表。再看匹配逻辑。", "再看匹配逻辑"},                           // 刚写完一句:仍是这一句
		{"第一行\n第二行\n", "第二行"},                                 // 换行切句
		{"Look at router.rs first. Then match", "Then match"}, // 文件名里的点不切
		{"Done. ", "Done"},
		{"## 分析\n- 键没有归一化", "键没有归一化"}, // Markdown 记号去掉
		{"**注意**", "注意**"},            // 只去开头的记号
	}
	for _, c := range cases {
		if got := thinkLastLine(c.in); got != c.want {
			t.Errorf("thinkLastLine(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	if n := len([]rune(thinkLastLine(strings.Repeat("长", 500)))); n != thinkLineMaxRunes {
		t.Fatalf("长句应截到 %d 字,实际 %d", thinkLineMaxRunes, n)
	}
}
