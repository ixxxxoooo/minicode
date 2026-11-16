package app

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// tabCenter 取某标签(标签条一行里、名字含 name 的可访问项)的中心点。
func tabCenter(t *testing.T, tt *ui.Tester, name string) (float32, float32) {
	t.Helper()
	for _, i := range tt.AccessAll() {
		if strings.Contains(i.Name, name) && !i.Offscreen && i.Bounds.Y >= 30 && i.Bounds.Y < 50 && i.Bounds.H > 20 {
			return i.Bounds.X + i.Bounds.W/2, i.Bounds.Y + i.Bounds.H/2
		}
	}
	t.Fatalf("找不到标签 %s", name)
	return 0, 0
}

// 鼠标中键:在标签上按下再抬起关闭该标签;按下与抬起不在同一标签不关。
func TestMiddleClickClosesTab(t *testing.T) {
	a, tt := demoIDE(t)
	d := a.ide
	d.tabs, d.active = nil, ""
	for _, p := range []string{"README.md", "docs/guide.md", "src/router.rs"} {
		d.openFile(p, true)
	}
	tt.Frame()
	tt.Advance(time.Second) // 等形态切换的宽度过渡走完
	tt.Frame()
	// 按下与抬起不在同一标签:不关
	rx, ry := tabCenter(t, tt, "router.rs")
	gx, gy := tabCenter(t, tt, "guide.md")
	tt.PressButton(2, rx, ry)
	tt.ReleaseButton(2, gx, gy)
	tt.Frame()
	if len(d.tabs) != 3 {
		t.Fatalf("按下与抬起不在同一标签不应关闭:%d 个标签", len(d.tabs))
	}
	tt.PressButton(2, gx, gy)
	tt.ReleaseButton(2, gx, gy)
	tt.Frame()
	if d.tab("docs/guide.md") != nil || len(d.tabs) != 2 {
		t.Fatalf("中键应关闭 guide.md:%d 个标签", len(d.tabs))
	}
}

// 「全部标签」弹层:列出所有标签,点一行切换并关闭弹层。
func TestTabListPopup(t *testing.T) {
	a, tt := demoIDE(t)
	d := a.ide
	d.tabs, d.active = nil, ""
	for _, p := range []string{"README.md", "docs/guide.md", "src/router.rs"} {
		d.openFile(p, true)
	}
	a.popup = etabListKey
	tt.Frame()
	tt.Frame()
	if d.active != "src/router.rs" {
		t.Fatalf("前提:当前标签应是最后打开的:%s", d.active)
	}
	if got := tabListDir("docs/guide.md"); got != "docs" {
		t.Fatalf("目录:%q", got)
	}
	if tabListDir("README.md") != "" {
		t.Fatal("根目录下的文件不显示目录")
	}
	if got := tabListDir("/Users/x/.claude/memory/a.md"); !strings.HasSuffix(got, ".claude/memory") {
		t.Fatalf("绝对路径目录:%q", got)
	}
}
