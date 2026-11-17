package app

import (
	"strings"
	"testing"
)

const longBlameMsg = "feat: AI 历史会话改为按钮下弹出面板、输入框快捷键可用；恢复顶栏胶囊与图标；侧栏历史语法高亮且筛选默认隐藏；网格列宽按内容；日志悬停复制；分割线去空隙；行预览只留一个"

// 截断按剩余格数自适应:剩得多放得多、剩得少放得少,永远不超过 room、单行、放不下补省略号。
func TestBlameInlayFitAdaptsToRoom(t *testing.T) {
	prev := 0
	for _, room := range []int{30, 50, 80, 120, 200} {
		got := blameInlayFit("demo", "4 天前", longBlameMsg, room)
		if textCells(got) > float32(room) {
			t.Fatalf("room=%d 时不应超出:%q(%.1f 格)", room, got, textCells(got))
		}
		if strings.ContainsAny(got, "\n\t") {
			t.Fatalf("应是单行:%q", got)
		}
		if int(textCells(got)) < prev {
			t.Fatalf("room 越大放得越多:room=%d 得 %.1f 格,上一档 %d 格", room, textCells(got), prev)
		}
		if room < int(textCells(longBlameMsg))+30 && strings.Contains(got, "feat") && !strings.HasSuffix(got, "…") {
			t.Fatalf("放不下整句时应以省略号结尾:%q", got)
		}
		prev = int(textCells(got))
	}
	// 放得下就整句放
	if got := blameInlayFit("a", "1 天前", "fix: 小修", 100); !strings.HasSuffix(got, "fix: 小修") || strings.Contains(got, "…") {
		t.Fatalf("放得下应完整显示:%q", got)
	}
	// 窄到放不下:不显示 / 只显示作者与时间
	if got := blameInlayFit("demo", "4 天前", longBlameMsg, 8); got != "" {
		t.Fatalf("room 太小应不显示:%q", got)
	}
	if got := blameInlayFit("demo", "4 天前", longBlameMsg, 24); strings.Contains(got, "feat") || textCells(got) > 24 {
		t.Fatalf("只够放作者与时间:%q", got)
	}
	// 量不出宽度(首帧):退回固定上限
	if got := blameInlayFit("demo", "4 天前", longBlameMsg, 0); textCells(got) > float32(4+blameAuthorCells+3+8+3+blameMsgFallback) {
		t.Fatalf("量不出宽度时应有上限:%q", got)
	}
}

// 最后一行占了多少格:不折行就是整行宽;折行按词 / 宽字符 / 超长词估算。
func TestLastRowCells(t *testing.T) {
	if got := lastRowCells("hello world", 80, 4, false); got != 11 {
		t.Fatalf("不折行应为整行宽 11,得 %d", got)
	}
	if got := lastRowCells("\tx", 80, 4, false); got != 5 {
		t.Fatalf("Tab 应对齐到 4 格,得 %d", got)
	}
	if got := lastRowCells("中文ab", 80, 4, false); got != 6 {
		t.Fatalf("宽字符占 2 格,得 %d", got)
	}
	// 折行:cols=10,「aaaa bbbbbbb」整词 bbbbbbb(7)放不下第一行剩余 → 移到下一行,最后一行 7 格
	if got := lastRowCells("aaaa bbbbbbb", 10, 4, true); got != 7 {
		t.Fatalf("整词移到下一行,最后一行应为 7,得 %d", got)
	}
	// 超长词(比一行还长):按字符断,25 个字符在 10 格宽里最后一行 5 格
	if got := lastRowCells(strings.Repeat("x", 25), 10, 4, true); got != 5 {
		t.Fatalf("超长词按字符断,最后一行应为 5,得 %d", got)
	}
	// 没折行时与不折行一致
	if got := lastRowCells("short", 80, 4, true); got != 5 {
		t.Fatalf("短行应为 5,得 %d", got)
	}
}
