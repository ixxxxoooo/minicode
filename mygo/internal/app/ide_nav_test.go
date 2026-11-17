package app

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

func demoIDE(t *testing.T) (*App, *ui.Tester) {
	t.Helper()
	a, tt := quickApp(t)
	tt.Frame()
	a.ide.seedDemo()
	a.SetMode(ModeIDE)
	tt.Frame()
	return a, tt
}

func TestGoToDefinitionAndHistory(t *testing.T) {
	a, tt := demoIDE(t)
	d := a.ide
	tb := d.tab("src/router.rs")
	// 光标放在 match_path 里的 normalize(path) 调用上
	line := -1
	for i := 0; i < tb.buf.LineCount(); i++ {
		if strings.Contains(tb.buf.Line(i), "let key = normalize(path);") {
			line = i
		}
	}
	if line < 0 {
		t.Fatal("演示文件里找不到调用点")
	}
	col := strings.Index(tb.buf.Line(line), "normalize") + 2
	d.active = "src/router.rs"
	tb.st.GoTo(editor.Pos{Line: line, Col: col})
	a.goDef()
	tt.Frame()
	if p, _ := d.curPos(); p.id != "src/router.rs" || !strings.Contains(tb.buf.Line(p.line), "fn normalize") {
		t.Fatalf("转到定义没落在 fn normalize 上:%+v", p)
	}
	if !d.canBack() {
		t.Fatal("跳转后应能后退")
	}
	a.navGo(-1)
	tt.Frame()
	if p, _ := d.curPos(); p.line != line {
		t.Fatalf("后退应回到调用点(行 %d),得到 %d", line, p.line)
	}
	if !d.canForward() {
		t.Fatal("后退后应能前进")
	}
	a.navGo(1)
	tt.Frame()
	if p, _ := d.curPos(); !strings.Contains(tb.buf.Line(p.line), "fn normalize") {
		t.Fatalf("前进应回到定义,得到 %+v", p)
	}
}

func TestRefsAndRename(t *testing.T) {
	a, _ := demoIDE(t)
	d := a.ide
	refs := d.findRefs("normalize")
	files := map[string]bool{}
	for _, r := range refs {
		files[r.path] = true
	}
	if len(refs) < 3 {
		t.Fatalf("normalize 至少有定义 + 2 处调用,得到 %d 处", len(refs))
	}
	a.showRefs("normalize")
	// 演示模式下 normalize 的引用面板用的是固定的演示数据(含项目外 / 测试文件 / 超长行),条数与本地扫描不必相等
	if d.bottomTab != "refs" || d.refs == nil || len(d.refs.items) < 3 {
		t.Fatalf("引用面板没有打开 / 数量不对:%q %+v", d.bottomTab, d.refs)
	}
	a.applyRename("normalize", "canonical")
	if n := len(d.findRefs("normalize")); n != 0 {
		t.Fatalf("重命名后旧名还剩 %d 处", n)
	}
	if n := len(d.findRefs("canonical")); n != len(refs) {
		t.Fatalf("新名应有 %d 处,得到 %d", len(refs), n)
	}
	if defs := d.findDefs("canonical", ""); len(defs) == 0 {
		t.Fatal("重命名后找不到新名的定义")
	}
}

func TestMinimapOverlayInfo(t *testing.T) {
	a, _ := demoIDE(t)
	tb := a.ide.tab("src/router.rs")
	tb.st.GoTo(editor.Pos{Line: 17, Col: 20}) // 在 normalize 之类的单词上
	info := a.overlayInfo(tb)
	if len(info.git) == 0 {
		t.Error("router.rs 相对 HEAD 有改动,迷你地图左缘应有 Git 刻度")
	}
	if got := gitMarks("a\nb\n", "a\nB\nc\n"); got[1] != '~' || got[2] != '+' {
		t.Errorf("gitMarks 不对:%v", got)
	}
}
