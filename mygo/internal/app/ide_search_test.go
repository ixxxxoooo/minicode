package app

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestSearchPanel(t *testing.T) {
	a, tt := newTestApp(t)
	a.OpenSearch()
	tt.Frame()
	if a.mode != ModeIDE || a.NavID() != "search" {
		t.Fatalf("⌘⇧F 应进 IDE 并打开搜索:mode=%v nav=%s", a.mode, a.NavID())
	}
	a.SetSearch("match_path", "", false)
	tt.Frame()
	if a.sr.res.Total == 0 || len(a.sr.rows) == 0 {
		t.Fatalf("应搜到结果:%+v", a.sr.res.Total)
	}
	total := a.sr.res.Total
	// 折叠第一个文件
	a.sr.folded[a.sr.res.Files[0].Path] = true
	a.sr.rows = a.sr.flatten()
	if len(a.sr.rows) != len(a.sr.res.Files)+total-len(a.sr.res.Files[0].Matches) {
		t.Fatalf("折叠后行数不对:%d", len(a.sr.rows))
	}
	a.sr.folded = map[string]bool{}
	a.sr.rows = a.sr.flatten()
	// 点击匹配:在预览标签里打开
	var first = a.sr.res.Files[0].Matches[0]
	a.ide.openAt(first.Path, first.Line, first.Col, first.End, false)
	tt.Frame()
	if a.ide.active != first.Path {
		t.Fatalf("应打开 %s,实际 %s", first.Path, a.ide.active)
	}
	if tab := a.ide.cur(); tab.reveal != nil {
		t.Fatal("渲染后应已应用选区")
	} else if sel := tab.st.SelectedText(); !strings.EqualFold(sel, "match_path") {
		t.Fatalf("应选中命中文本,实际 %q", sel)
	}
}

func TestSearchReplaceAll(t *testing.T) {
	a, tt := newTestApp(t)
	a.OpenSearch()
	a.SetSearch("match_path", "find_route", true)
	tt.Frame()
	// 忽略第一处
	skip := a.sr.res.Files[0].Matches[0]
	a.sr.ignored[mkey(skip)] = true
	live := len(a.sr.live())
	a.replaceAll()
	tt.Frame()
	src, _ := a.ide.fs.Read(skip.Path)
	lines := strings.Split(src, "\n")
	if !strings.Contains(lines[skip.Line], "match_path") {
		t.Fatalf("被忽略的匹配不应替换:%q", lines[skip.Line])
	}
	// 重搜后只剩被忽略的那一处
	if a.sr.res.Total != 1 || live < 2 {
		t.Fatalf("替换后应只剩 1 处,实际 %d(替换前列出 %d)", a.sr.res.Total, live)
	}
	_ = ui.Cmd
}
