package app

import (
	"strings"
	"testing"
)

func cellsOf(s string) int {
	n := 0
	for _, r := range s {
		n += cellWidth(r)
	}
	return n
}

// 行末追溯提示:长说明截到约 50 格、单行、不含换行;中文按 2 格算;短的原样。
func TestBlameInlayClip(t *testing.T) {
	if got := clipCells("fix: 修一下", 50); got != "fix: 修一下" {
		t.Fatalf("短说明应原样:%q", got)
	}
	long := "feat: AI 历史会话改为按钮下弹出面板、输入框快捷键可用;恢复顶栏胶囊与图标;侧栏历史语法高亮且筛选默认隐藏;网格列宽按内容;日志悬停复制;分割线去空隙;行预览只留一个"
	got := clipCells(long, blameMsgCells)
	if cellsOf(got) > blameMsgCells || !strings.HasSuffix(got, "…") {
		t.Fatalf("应截到 %d 格内并以省略号结尾,得 %q(%d 格)", blameMsgCells, got, cellsOf(got))
	}
	if got := clipCells("第一行\n\n第二行\t后面", 50); got != "第一行 第二行 后面" {
		t.Fatalf("换行 / 制表应压成空格:%q", got)
	}
	txt := blameInlayText("demo", "4 天前", long)
	if strings.ContainsAny(txt, "\n\t") || cellsOf(txt) > 4+blameAuthorCells+3+len("4 天前")+3+blameMsgCells+2 {
		t.Fatalf("整段提示应是单行且有长度上限:%q(%d 格)", txt, cellsOf(txt))
	}
	if got := clipCells("abcdefghij", 10); got != "abcdefghij" {
		t.Fatalf("刚好 10 格不应截:%q", got)
	}
	if got := clipCells("abcdefghijk", 10); cellsOf(got) > 10 || !strings.HasSuffix(got, "…") {
		t.Fatalf("11 格应截成 ≤10 格:%q", got)
	}
}
