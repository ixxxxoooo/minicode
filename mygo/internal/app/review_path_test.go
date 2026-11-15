package app

import (
	"os"
	"path/filepath"
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
