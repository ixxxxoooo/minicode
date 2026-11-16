package app

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func hasPath(a *App, p string) bool {
	_, err := a.ide.fs.Read(p)
	return err == nil
}

// 新建文件:右键菜单 → 行内输入 → ⏎;已存在时留着输入行并给原因,改名后成功,新文件在编辑器里打开。
func TestFileTreeNewFile(t *testing.T) {
	a, tt := demoIDE(t)
	a.nav, a.sideOpen = "files", true
	a.ide.open["src"] = true
	tt.Frame()
	if err := tt.RightClick("router.rs"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("新建文件…"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.fo.kind != "new" || a.fo.dir != "src" {
		t.Fatalf("文件上右键新建应建在同级:%+v", a.fo)
	}
	// 与已有文件重名:输入行不关,显示原因
	a.fo.buf = "main.rs"
	a.commitFileOp()
	tt.Frame()
	if !a.fo.active() || !strings.Contains(a.fo.err, "已存在") || !tt.HasText("已存在同名") {
		t.Fatalf("重名应留着输入行并显示原因:%+v", a.fo)
	}
	// 非法名字
	a.fo.buf = ""
	a.commitFileOp()
	if a.fo.err == "" || !a.fo.active() {
		t.Fatal("空名字应报错")
	}
	// 改名成功;带子目录一起建
	a.fo.buf = "util/handlers.rs"
	a.commitFileOp()
	tt.Frame()
	if a.fo.active() || !hasPath(a, "src/util/handlers.rs") {
		t.Fatalf("应创建成功:%+v", a.fo)
	}
	if a.ide.cur() == nil || a.ide.cur().path != "src/util/handlers.rs" || a.ide.sel != "src/util/handlers.rs" {
		t.Fatal("新建的文件应在编辑器里打开并在树里选中")
	}
}

func TestFileTreeNewFolderAndEsc(t *testing.T) {
	a, tt := demoIDE(t)
	a.beginNew("src", true)
	tt.Frame()
	if !a.ide.open["src"] {
		t.Fatal("新建时文件夹应自动展开")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if a.fo.active() {
		t.Fatal("Esc 应取消")
	}
	a.beginNew("src", true)
	a.fo.buf = "handlers"
	a.commitFileOp()
	if !hasPath(a, "src/handlers") && len(a.ide.listing("src/handlers")) == 0 {
		// Mem 里空目录用 List 判断
		if _, err := a.ide.fs.List("src/handlers"); err != nil {
			t.Fatalf("文件夹应创建成功:%v", err)
		}
	}
}

// 重命名:已打开标签跟着改路径并保留未保存的修改;重名报错。
func TestFileTreeRename(t *testing.T) {
	a, tt := demoIDE(t)
	a.ide.openFile("src/router.rs", true)
	tb := a.ide.tab("src/router.rs")
	tb.buf.Set("// 改过\n" + tb.buf.String())
	a.beginRenameFile("src/router.rs")
	tt.Frame()
	if a.fo.buf != "router.rs" || a.fo.ext != ".rs" {
		t.Fatalf("输入框应预填完整文件名:%q ext=%q", a.fo.buf, a.fo.ext)
	}
	a.fo.buf = "main.rs"
	a.commitFileOp()
	if !strings.Contains(a.fo.err, "已存在") || !hasPath(a, "src/router.rs") {
		t.Fatalf("重名不能覆盖:%+v", a.fo)
	}
	a.fo.buf = "routes.rs"
	a.commitFileOp()
	tt.Frame()
	if hasPath(a, "src/router.rs") || !hasPath(a, "src/routes.rs") {
		t.Fatal("文件应改名")
	}
	nt := a.ide.tab("src/routes.rs")
	if nt == nil || nt != tb || a.ide.tab("src/router.rs") != nil || a.ide.active != "src/routes.rs" || !nt.dirty() {
		t.Fatalf("标签应跟着改路径且保留未保存修改:%v dirty=%v", nt, nt != nil && nt.dirty())
	}
}

// 回归:输入框里是完整文件名,改扩展名 / 去掉扩展名都按输入原样改,不再自动补回原扩展名
// (此前 router.rs → router.md 会变成 router.md.rs)。
func TestFileTreeRenameChangesExtension(t *testing.T) {
	a, tt := demoIDE(t)
	a.beginRenameFile("src/router.rs")
	tt.Frame()
	a.fo.buf = "router.md"
	a.commitFileOp()
	tt.Frame()
	if !hasPath(a, "src/router.md") || hasPath(a, "src/router.md.rs") || hasPath(a, "src/router.rs") {
		t.Fatal("改扩展名应得到 router.md")
	}
	a.beginRenameFile("src/router.md")
	tt.Frame()
	a.fo.buf = "router"
	a.commitFileOp()
	tt.Frame()
	if !hasPath(a, "src/router") || hasPath(a, "src/router.md") {
		t.Fatal("应能去掉扩展名")
	}
}

// 删除:确认框写明文件夹里的文件数,确认后关掉受影响的标签。
func TestFileTreeDelete(t *testing.T) {
	a, tt := demoIDE(t)
	a.ide.openFile("src/router.rs", true)
	a.askDelete("src", true)
	tt.Frame()
	if a.pd.kind != "confirm" || !strings.Contains(a.pd.title, "连同里面的") || !strings.Contains(a.pd.body, "演示数据") {
		t.Fatalf("文件夹删除确认文案不对:%q / %q", a.pd.title, a.pd.body)
	}
	a.pd.ok()
	toast := a.toast // Frame 会清掉提示,先取
	tt.Frame()
	if a.ide.tab("src/router.rs") != nil || hasPath(a, "src/router.rs") {
		t.Fatal("删除后标签应关闭、文件应不在")
	}
	if !strings.Contains(toast, "已移除") {
		t.Fatalf("toast=%q", toast)
	}
}

// F2 在文件树有选中项时可进入重命名(不必先打开编辑器标签)。
func TestFileTreeF2Rename(t *testing.T) {
	a, tt := demoIDE(t)
	a.nav, a.sideOpen = "files", true
	for _, tb := range append([]*ideTab(nil), a.ide.tabs...) {
		a.closeWsTab(tb.id)
	}
	a.ide.sel = "src/router.rs"
	a.filesFocus = true
	tt.Frame()
	tt.Key(0, ui.KeyF2)
	tt.Frame()
	if a.fo.kind != "rename" || a.fo.path != "src/router.rs" {
		t.Fatalf("F2 应进入重命名:%+v", a.fo)
	}
}

// 工作区顶层文件夹没有重命名 / 删除;新建建在主文件夹里。
func TestFileTreeMultiRootTop(t *testing.T) {
	a, tt := newTestApp(t)
	a.ActivateWorkspace("ws-fullstack")
	a.SetMode(ModeIDE)
	tt.Frame()
	a.beginNew("", false)
	if a.fo.dir != a.repos[0].name {
		t.Fatalf("多根时顶层新建应落在主文件夹:%q", a.fo.dir)
	}
}
