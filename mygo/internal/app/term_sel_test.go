package app

import (
	"testing"

	"github.com/ixxxxoooo/minicode/internal/term"
)

func TestWordBound(t *testing.T) {
	rs := []rune("foo bar")
	if !wordBound(rs, 0, 3) || !wordBound(rs, 4, 7) {
		t.Fatal("应对齐整词")
	}
	if wordBound(rs, 1, 3) {
		t.Fatal("中间子串不算全词")
	}
}

func fakeScreen(lines ...string) *term.Screen {
	s := &term.Screen{Cols: 80, Rows: len(lines)}
	for _, l := range lines {
		s.Lines = append(s.Lines, term.Line{{Text: l}})
	}
	return s
}

func TestTermSelectionCopyAndWord(t *testing.T) {
	tp := &termPanel{scr: fakeScreen("hello world", "second line  ", "第三行 text")}
	tp.sel = termSel{on: true, a: cellPos{0, 6}, b: cellPos{1, 5}}
	if got := tp.selectedText(); got != "world\nsecond" {
		t.Fatalf("跨行选区文字 %q", got)
	}
	// 反向拖选结果相同
	tp.sel = termSel{on: true, a: cellPos{1, 5}, b: cellPos{0, 6}}
	if got := tp.selectedText(); got != "world\nsecond" {
		t.Fatalf("反向选区 %q", got)
	}
	tp.wordAt(cellPos{0, 2})
	if got := tp.selectedText(); got != "hello" {
		t.Fatalf("双击选词 %q", got)
	}
	tp.selectLine(cellPos{1, 3})
	if got := tp.selectedText(); got != "second line" {
		t.Fatalf("三击选行(去尾部空白) %q", got)
	}
	// 宽字符:每个汉字占两列,选中不丢字
	tp.sel = termSel{on: true, a: cellPos{2, 0}, b: cellPos{2, 5}}
	if got := tp.selectedText(); got != "第三行" {
		t.Fatalf("宽字符选区 %q", got)
	}
	tp.selectAll()
	if got := tp.selectedText(); got != "hello world\nsecond line\n第三行 text" {
		t.Fatalf("全选 %q", got)
	}
	if c0, c1, ok := tp.sel.span(1, 80); !ok || c0 != 0 || c1 != 80 {
		t.Fatalf("中间行整行覆盖 %v %v %v", c0, c1, ok)
	}
}

func TestWordBoundFull(t *testing.T) {
	rs := []rune("foo bar")
	if !wordBound(rs, 0, 3) {
		t.Fatal("foo 应是全词")
	}
	if wordBound(rs, 4, 7) == false {
		// bar
	} else if !wordBound(rs, 4, 7) {
		t.Fatal("bar 应是全词")
	}
	if wordBound(rs, 1, 4) {
		t.Fatal("oob 子串不应算全词")
	}
}

func TestTermFind(t *testing.T) {
	tp := &termPanel{scr: fakeScreen("Error: foo", "ok", "error again, ERROR")}
	tp.find = termFind{open: true, query: "error"}
	tp.refreshFind()
	if len(tp.find.hits) != 3 {
		t.Fatalf("不区分大小写应命中 3 处:%v", tp.find.hits)
	}
	if h := tp.find.hits[0]; h.row != 0 || h.c0 != 0 || h.c1 != 5 {
		t.Fatalf("第一处 %+v", h)
	}
	tp.find.cs, tp.find.key = true, ""
	tp.refreshFind()
	if len(tp.find.hits) != 1 || tp.find.hits[0].row != 2 {
		t.Fatalf("区分大小写只命中小写 error:%v", tp.find.hits)
	}
	tp.find.query, tp.find.key = "", ""
	tp.refreshFind()
	if len(tp.find.hits) != 0 {
		t.Fatal("空查询没有命中")
	}
}

func TestTermLinks(t *testing.T) {
	a, _ := demoIDE(t)
	tp := &termPanel{scr: fakeScreen("see https://docs.rs/axum/routing, ok", "error at src/router.rs:17:20 here", "version 1.2.3 and x.y")}
	if l := a.linkAt(tp, cellPos{0, 10}); l == nil || l.kind != "url" || l.target != "https://docs.rs/axum/routing" {
		t.Fatalf("URL 链接 %+v", l)
	}
	if l := a.linkAt(tp, cellPos{0, 1}); l != nil {
		t.Fatalf("普通文字不是链接 %+v", l)
	}
	l := a.linkAt(tp, cellPos{1, 15})
	if l == nil || l.kind != "file" || l.target != "src/router.rs" || l.line != 17 || l.col != 20 {
		t.Fatalf("文件链接 %+v", l)
	}
	if l := a.linkAt(tp, cellPos{2, 9}); l != nil {
		t.Fatalf("版本号不是链接 %+v", l)
	}
	// 打开文件链接:编辑器跳到那一行
	a.openLink(l)
	if tb := a.ide.cur(); tb == nil || tb.id != "src/router.rs" {
		t.Fatal("应在编辑器里打开 src/router.rs")
	}
}
