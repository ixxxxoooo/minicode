package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/vcs"
	"github.com/ixxxxoooo/minicode/internal/workspace"
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

// 编辑经过权限卡后,列表里的 ToolCall 被 PermissionRequest 替换(工具挂在 Tool 字段),
// 改动统计与回合折叠摘要仍要算上它;被拒绝的、还在等待的不算。(实机发现:允许一次后改动卡 / 本轮 / 审查全空)
func TestChangedFilesThroughPermission(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	a.proj = "/tmp/mc-e2e"
	a.st.Apply(core.EvAppend{Item: &core.User{ID: "u1", Text: "加 sub"}})
	mk := func(id, opt string) {
		tc := core.NewToolCall(id, "Edit", "编辑 main.go").
			WithContent(&core.DiffBlock{Path: "/private/tmp/mc-e2e/" + id + ".go", Old: "a", New: "b", Line: 1})
		a.st.Apply(core.EvAppend{Item: tc})
		req := &core.PermissionRequest{ID: "p-" + id, Tool: tc, Options: []core.PermissionOption{
			{ID: "allow", Label: "允许一次", Kind: core.AllowOnce}, {ID: "reject", Label: "拒绝", Kind: core.RejectOnce}}}
		a.st.Apply(core.EvPromoteToPermission{Request: req})
		if opt != "" {
			a.st.Apply(core.EvPermissionResolved{ID: req.ID, Option: opt})
		}
	}
	mk("ok", "allow")
	mk("no", "reject")
	mk("wait", "")
	fs := a.changedFiles()
	if len(fs) != 1 || !strings.HasSuffix(fs[0].path, "ok.go") {
		t.Fatalf("只有已允许的编辑算改动,实际 %+v", fs)
	}
	if _, n := turnFoldSummary(a.st.Items[1:]); n != 2 { // 已允许 + 已拒绝(工具本身带 diff),等待中的不算
		t.Fatalf("折叠摘要文件数=%d,期望 2", n)
	}
}

// 被拒绝的命令没有执行过,命令卡不能显示红色「退出码 1」(后端的拒绝信号);实机发现。
func TestDeniedCommandCardNoExitCode(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	a.st.Apply(core.EvAppend{Item: &core.User{ID: "u1", Text: "跑一下"}})
	one := 1
	tc := core.NewToolCall("t1", "Bash", "运行 rm -rf x").
		WithContent(&core.TerminalBlock{Command: "rm -rf x", Output: "用户拒绝了此次操作", ExitCode: &one})
	tc.Status = core.ToolDenied
	a.st.Apply(core.EvAppend{Item: tc})
	a.st.Apply(core.EvAppend{Item: &core.Assistant{ID: "a1", Text: "好的"}})
	a.st.Apply(core.EvTurnEnded{})
	a.folds["turnfold:u1"] = true // 过程区展开
	tt := ui.NewTester(a.View, 1280, 800)
	tt.Frame()
	if tt.HasText("退出码 1") {
		t.Fatal("被拒绝的命令不该显示退出码")
	}
	if !tt.HasText("未执行") {
		t.Fatal("被拒绝的命令卡应显示「未执行」")
	}
}

// 改动里出现新文件(Claude 新建)时,文件树目录缓存要作废;实机发现 main_test.go 只出现在 SCM、树里要手点刷新。
func TestSyncTreeGitInvalidatesTreeCache(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	a.ide.listCache = map[string][]workspace.Entry{"": nil}
	a.ide.filesCache = []string{"a.go"}
	a.sc.changes = []vcs.Change{{Path: "new_test.go", Status: 'U'}}
	a.syncTreeGit()
	if a.ide.listCache != nil || a.ide.filesCache != nil {
		t.Fatal("出现新的改动文件后,树缓存应作废")
	}
	a.ide.listCache = map[string][]workspace.Entry{"": nil}
	a.syncTreeGit() // 路径集合没变:缓存保留,避免每次轮询都重读磁盘
	if a.ide.listCache == nil {
		t.Fatal("路径集合不变时不该清缓存")
	}
}
