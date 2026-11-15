package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
)

// 项目根经过软链(macOS 的 /tmp、/var)时,claude 汇报的是真实路径,也要能换成相对路径。
func TestRelToProjResolvesSymlinkedRoot(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	link := filepath.Join(t.TempDir(), "link")
	real := t.TempDir()
	if err := os.Symlink(real, link); err != nil {
		t.Skip("无法创建软链:", err)
	}
	a.proj = link
	realRoot, _ := filepath.EvalSymlinks(link)
	for _, c := range []struct{ in, want string }{
		{link + "/src/a.go", "src/a.go"},
		{realRoot + "/src/a.go", "src/a.go"},
		{realRoot, ""},
		{"/elsewhere/x.go", "elsewhere/x.go"},
	} {
		if got := a.relToProj(c.in); got != c.want {
			t.Fatalf("relToProj(%q)=%q,期望 %q", c.in, got, c.want)
		}
	}
}

// 回合结束后,会话里改过文件(路径是真实路径)就要出现「N 个文件已改动」卡,工作区起始页也要有「审查本轮改动」。
func TestChangesCardAfterRealTurn(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	a.proj = "/tmp/mc-e2e"
	a.st.Apply(core.EvAppend{Item: &core.User{ID: "u1", Text: "修一下 add"}})
	tc := core.NewToolCall("t1", "Edit", "编辑 main.go").
		WithContent(&core.DiffBlock{Path: "/private/tmp/mc-e2e/main.go", Old: "a - b", New: "a + b", Line: 3})
	a.st.Apply(core.EvAppend{Item: tc})
	a.st.Apply(core.EvAppend{Item: &core.Assistant{ID: "a1", Text: "已修复"}})
	a.st.Apply(core.EvTurnEnded{})
	tt := ui.NewTester(a.View, 1280, 800)
	tt.Frame()
	if n := len(a.changedFiles()); n != 1 {
		t.Fatalf("changedFiles=%d,期望 1", n)
	}
	if !tt.HasText("1 个文件已改动") {
		t.Fatal("回合结束后应出现改动卡")
	}
}

// 多轮会话里改动卡只出现一次(最后一轮),并且目录是相对项目根的。
func TestChangesCardOnlyOnLastRound(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	a.proj = "/tmp/mc-e2e"
	for _, id := range []string{"1", "2"} {
		a.st.Apply(core.EvAppend{Item: &core.User{ID: "u" + id, Text: "改 " + id}})
		a.st.Apply(core.EvAppend{Item: core.NewToolCall("t"+id, "Edit", "编辑").
			WithContent(&core.DiffBlock{Path: "/private/tmp/mc-e2e/sub/x.go", Old: "a", New: "b", Line: 1})})
		a.st.Apply(core.EvAppend{Item: &core.Assistant{ID: "a" + id, Text: "好了"}})
		a.st.Apply(core.EvTurnEnded{})
	}
	tt := ui.NewTester(a.View, 1280, 800)
	tt.Frame()
	n := 0
	for _, x := range tt.Texts() {
		if strings.Contains(x, "个文件已改动") {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("改动卡出现 %d 次,期望 1", n)
	}
}

// 失败 / 被拒绝的编辑没有落盘,不计入改动统计。
func TestChangedFilesSkipsFailedEdits(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	a.proj = "/tmp/mc-e2e"
	a.st.Apply(core.EvAppend{Item: &core.User{ID: "u1", Text: "改"}})
	bad := core.NewToolCall("t1", "Edit", "编辑").WithContent(&core.DiffBlock{Path: "/tmp/mc-e2e/a.go", Old: "x", New: "y", Line: 1})
	bad.Status = core.ToolFailed
	ok := core.NewToolCall("t2", "Edit", "编辑").WithContent(&core.DiffBlock{Path: "/tmp/mc-e2e/b.go", Old: "x", New: "y", Line: 1})
	ok.Status = core.ToolCompleted
	a.st.Apply(core.EvAppend{Item: bad})
	a.st.Apply(core.EvAppend{Item: ok})
	fs := a.changedFiles()
	if len(fs) != 1 || fs[0].path != "/tmp/mc-e2e/b.go" {
		t.Fatalf("changedFiles=%v,期望只有 b.go", fs)
	}
}

// 编辑权限卡要预览改动,路径写相对项目根。
func TestPermCardShowsDiffPreview(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	a.proj = "/tmp/mc-e2e"
	a.st.Apply(core.EvAppend{Item: &core.User{ID: "u1", Text: "改"}})
	tool := core.NewToolCall("p1", "Edit", "编辑 main.go").
		WithInput(map[string]any{"file_path": "/private/tmp/mc-e2e/sub/main.go"}).
		WithContent(&core.DiffBlock{Path: "/private/tmp/mc-e2e/sub/main.go", Old: "return a - b", New: "return a + b", Line: 3})
	a.st.Apply(core.EvAppend{Item: &core.PermissionRequest{ID: "p1", Tool: tool,
		Options: []core.PermissionOption{{ID: "allow", Label: "允许一次", Kind: core.AllowOnce}, {ID: "no", Label: "拒绝", Kind: core.RejectOnce}}}})
	tt := ui.NewTester(a.View, 1280, 800)
	tt.Frame()
	if !tt.HasText("return a + b") {
		t.Fatalf("权限卡应预览改动后的代码,实际文本:%v", tt.Texts())
	}
	if !tt.HasText("sub/main.go") {
		t.Fatal("权限卡的路径应是相对项目根的 sub/main.go")
	}
}
