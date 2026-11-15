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

// 多根工作区:搜索可限定到某个文件夹;限定后结果只来自那个文件夹,空结果写明范围;退出工作区清除范围。
func TestSearchFolderScope(t *testing.T) {
	a, tt := newTestApp(t)
	a.ActivateWorkspace("ws-fullstack")
	a.OpenSearch()
	a.SetSearch("render", "", false)
	tt.Frame()
	if a.sr.res.Total == 0 {
		t.Fatal("全部文件夹应搜到 render")
	}
	a.setSearchScope("web-dashboard")
	tt.Frame()
	if a.sr.res.Total == 0 {
		t.Fatal("限定 web-dashboard 应仍有结果")
	}
	for _, f := range a.sr.res.Files {
		if !strings.HasPrefix(f.Path, "web-dashboard/") {
			t.Fatalf("结果应只来自 web-dashboard:%s", f.Path)
		}
	}
	if !strings.HasPrefix(a.searchSummary(), "在 web-dashboard 内") {
		t.Fatalf("摘要应写明范围:%q", a.searchSummary())
	}
	a.setSearchScope("router-demo")
	tt.Frame()
	if a.sr.res.Total != 0 || a.searchSummary() != "router-demo 内没有结果" {
		t.Fatalf("router-demo 内没有 render:%d %q", a.sr.res.Total, a.searchSummary())
	}
	if a.scopeIdx() != 0 {
		t.Fatalf("范围序号:%d", a.scopeIdx())
	}
	a.setSearchScope("")
	tt.Frame()
	if a.sr.res.Total == 0 || a.scopeIdx() != -1 {
		t.Fatal("清除范围后应恢复全部结果")
	}
	// 右键「仅在此文件夹搜索」等价于限定范围;退出工作区清除
	a.setSearchScope("docs-site")
	a.leaveWorkspace()
	if a.sr.scope != "" {
		t.Fatal("退出工作区应清除范围")
	}
}
