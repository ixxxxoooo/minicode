package app

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// lastStepRow 取会话流里名字含 name 的、最靠下的可见步骤行。
func lastStepRow(t *testing.T, tt *ui.Tester, name string) ui.AccessInfo {
	t.Helper()
	var best ui.AccessInfo
	found := false
	for _, i := range tt.AccessAll() {
		if strings.Contains(i.Name, name) && !i.Offscreen && i.Bounds.H >= 20 && i.Bounds.H <= 30 {
			if !found || i.Bounds.Y > best.Bounds.Y {
				best, found = i, true
			}
		}
	}
	if !found {
		t.Fatalf("找不到步骤行 %q", name)
	}
	return best
}

// 在会话底部点开 / 收起一个工具步骤:被点的那一行必须留在原地、内容向下长,
// 鼠标不动再点一次就原地收起(会话流默认贴底,不拦的话展开会把整段内容向上顶)。
func TestFoldClickKeepsRowInPlace(t *testing.T) {
	a, tt := sessionApp(t, "ci")
	tt.SetSize(1280, 620)
	tt.Frame()
	a.flowList.ScrollToEnd()
	tt.Frame()
	tt.Advance(time.Second)
	tt.Frame()
	if !a.flowList.AtEnd() {
		t.Fatal("准备工作:会话流应停在底部")
	}
	const name = "运行 cargo test --test integration"
	row := lastStepRow(t, tt, name)
	tt.ClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2) // 演示场景默认是展开的:先收起,下面从「已收起」开始测
	tt.Frame()
	tt.Advance(time.Second)
	tt.Frame()
	a.flowList.ScrollToEnd()
	tt.Frame()
	row = lastStepRow(t, tt, name)
	cx, cy := row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2
	y0 := row.Bounds.Y

	for i, want := range []string{"点击后应展开(行位置不变)", "再点一次应收起(行位置仍不变)"} {
		tt.ClickAt(cx, cy)
		tt.Frame()
		tt.Advance(time.Second)
		tt.Frame()
		now := lastStepRow(t, tt, name)
		if d := now.Bounds.Y - y0; d > 1 || d < -1 {
			t.Fatalf("第 %d 次点击:%s,实际行 Y 从 %.1f 变成 %.1f", i+1, want, y0, now.Bounds.Y)
		}
	}
}
