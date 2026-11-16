package app

import (
	"strings"
	"testing"
	"time"
)

// 底部面板拖动:尺寸必须跟手(不能有缓动)——以前 a.mt() 返回零值过渡,框架把它当成「全部属性做 200 ms 缓动」,
// 面板上缘落后指针几十像素,编辑器文字还会透到面板标签条上。
func TestPanelDragFollowsPointer(t *testing.T) {
	a, tt := demoIDE(t)
	tt.Advance(time.Second)
	tt.Frame()
	r := a.ide.panelTop
	if r <= 0 {
		t.Skip("面板上缘位置未量到")
	}
	x, y0 := float32(400), r+1
	tt.Move(x, y0)
	tt.Press(x, y0)
	tt.Frame()
	var y float32
	for i := 1; i <= 6; i++ {
		y = y0 - 20*float32(i)
		tt.Move(x, y)
		tt.Advance(16 * time.Millisecond)
		tt.Frame()
	}
	tt.Frame() // 量到的是上一帧的矩形
	want := r - 120
	if got := a.ide.panelTop; got < want-2 || got > want+2 {
		t.Fatalf("拖动后面板上缘应跟手(期望约 %v),实际 %v(落后 = 还在做缓动)", want, got)
	}
	tt.Release(x, y)
	tt.Frame()
}

// 迷你地图:滚动后显示行范围标签,停 3 秒后收起。
func TestMinimapNeverShowsRange(t *testing.T) {
	a, tt := demoIDE(t)
	a.SetClock(tt.Now) // 迷你地图按 a.now() 判断「停了 3 秒」,跟着测试器的虚拟时间走
	tb := a.ide.cur()
	if tb == nil || tb.st == nil {
		t.Skip("没有当前文件标签")
	}
	tt.Advance(time.Second)
	tt.Frame()
	tt.Frame()
	has := func() bool {
		for _, s := range tt.Texts() {
			if strings.Contains(s, " 行 / 共 ") {
				return true
			}
		}
		return false
	}
	if has() {
		t.Fatal("刚打开、没有滚动,不应显示行范围")
	}
	tb.st.SetScrollY(tb.st.ScrollY() + 60)
	tt.Frame()
	tt.Frame()
	if tb.mmAt.IsZero() {
		t.Fatal("滚动后应记录滚动时刻")
	}
	// 「第 x–y 行 / 共 n」气泡已取消(用户反馈没人看):滚动中也不显示
	if has() {
		t.Fatalf("滚动中也不应显示行范围:%v", tt.Texts())
	}
	tt.Advance(3500 * time.Millisecond)
	tt.Frame()
	tt.Frame()
	if has() {
		t.Fatal("不应显示行范围")
	}
}

// 标签条:标签放不下时出现「全部标签」入口,菜单列出全部标签。
func TestTabStripOverflowList(t *testing.T) {
	a, tt := demoIDE(t)
	for _, p := range []string{"src/cache.rs", "src/config_copy.rs", "src/config_new.rs", "src/legacy.rs", "src/main.rs", "src/server.rs", "src/util.rs", "tests/case_test.rs", "tests/router_test.rs"} {
		a.ide.openFile(p, true)
	}
	for _, p := range []string{"docs/guide.md", "docs/api.md", "Cargo.toml"} {
		a.ide.openFile(p, true)
	}
	tt.SetSize(900, 700)
	tt.Advance(time.Second)
	tt.Frame()
	tt.Frame()
	n := len(a.ide.tabs)
	if n < 8 {
		t.Fatalf("应打开足够多的标签,实际 %d", n)
	}
	tip := "全部标签(" + itoa(n) + ")"
	if _, ok := tt.Access(tip); !ok {
		t.Fatalf("放不下时应有「%s」入口", tip)
	}
}
