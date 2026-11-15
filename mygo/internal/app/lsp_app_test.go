package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui/editor"
	"github.com/ixxxxoooo/minicode/internal/lsp"
)

const goDemoSrc = "package main\n\nimport \"fmt\"\n\n// Greet 打招呼。\nfunc Greet(name string) string { return \"hi \" + name }\n\nfunc main() {\n\tfmt.Println(Greet(\"x\"))\n\tvar n int = \"bad\"\n\t_ = n\n}\n"

func lspProject(t *testing.T) (*App, *testTB, string) {
	t.Helper()
	dir := t.TempDir()
	dir, _ = filepath.EvalSymlinks(dir)
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module demo\n\ngo 1.21\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "main.go"), []byte(goDemoSrc), 0o644)
	a, tt := newTestApp(t)
	a.SetProject(dir)
	a.SetMode(ModeIDE)
	t.Cleanup(a.CloseTerminals)
	a.ide.openFile("main.go", true)
	return a, &testTB{tt: tt}, dir
}

type testTB struct{ tt interface{ Frame() } }

func (b *testTB) frames(n int) {
	for i := 0; i < n; i++ {
		b.tt.Frame()
	}
}

// 真实 gopls 走完整条链:启动 → didOpen → 诊断进问题面板与波浪线 → 转到定义 / 引用 / 补全 / 悬停。
func TestLSPRealGopls(t *testing.T) {
	if _, ok := lsp.ForFile("a.go").Find(); !ok {
		t.Skip("没有安装 gopls")
	}
	a, b, _ := lspProject(t)
	pump := func(until func() bool, what string) {
		t.Helper()
		deadline := time.Now().Add(60 * time.Second)
		for time.Now().Before(deadline) {
			b.frames(1)
			a.drainUI()
			if until() {
				return
			}
			time.Sleep(40 * time.Millisecond)
		}
		t.Fatalf("等不到:%s", what)
	}
	pump(func() bool { s := a.lspFor("main.go"); return s != nil && s.state == lspReady }, "gopls 就绪")
	pump(func() bool { ne, _ := a.lspCounts(); return ne > 0 }, "类型错误诊断")
	if ds := a.lspH.diags["main.go"]; len(ds) == 0 || ds[0].Range.Start.Line != 9 {
		t.Fatalf("诊断应在第 10 行:%+v", ds)
	}
	tb := a.ide.cur()
	b.frames(2)
	if n := len(tb.st.Diagnostics()); n == 0 {
		t.Fatal("诊断应铺成编辑器里的波浪线")
	}

	// 转到定义:调用处 Greet → 声明(第 6 行)
	off := strings.Index(goDemoSrc, "Greet(\"x\")") + 2
	a.goDefAt(tb, off)
	a.drainUI()
	if p := tb.st.CursorPos(); a.ide.cur().path != "main.go" || p.Line != 5 {
		// 定义是异步回到界面线程的:再泵一轮
		pump(func() bool { return a.ide.cur().st != nil && a.ide.cur().st.CursorPos().Line == 5 }, "跳到定义所在行")
	}

	// 引用:Greet 声明 + 调用 = 2 处
	decl := strings.Index(goDemoSrc, "func Greet") + 6
	tb.st.Select(decl, decl)
	a.showRefs("Greet")
	a.drainUI()
	pump(func() bool { return a.ide.refs != nil && len(a.ide.refs.items) >= 2 }, "引用列表")
	if a.ide.bottomTab != "refs" {
		t.Fatalf("应切到「引用」分段:%s", a.ide.bottomTab)
	}

	// 补全:fmt. 之后
	src := goDemoSrc
	at := strings.Index(src, "fmt.Println") + len("fmt.")
	got := make(chan int, 1)
	_ = got
	a.completeAt(tb, editor.CompletionRequest{Offset: at, Prefix: "", Version: tb.buf.Version(), Explicit: true})
	a.drainUI()
	// 悬停:不报错即可(内容在弹层里),这里只验证请求链路不崩
	w := strings.Index(goDemoSrc, "Greet(\"x\")") + 1
	a.hoverAt(tb, editor.HoverRequest{Offset: w, Version: tb.buf.Version()})
	a.drainUI()
}

func TestLSPMissingServerShowsInstall(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // 找不到任何服务器
	t.Setenv("HOME", t.TempDir())
	a, b, _ := lspProject(t)
	b.frames(3)
	a.drainUI()
	s := a.lspFor("main.go")
	if s == nil || s.state != lspMissing {
		t.Fatalf("没装应是 missing:%+v", s)
	}
	a.ide.bottomOpen, a.ide.bottomTab = true, "problems"
	b.frames(2)
	tt := b.tt.(interface{ HasText(string) bool })
	if !tt.HasText("未安装 gopls") || !tt.HasText("在终端安装") {
		t.Fatal("问题面板应给出安装入口")
	}
	a.lspInstall(s)
	if tp := a.ide.tpc(); tp.initCmd != s.spec.Install || a.ide.bottomTab != "term" {
		t.Fatalf("安装应在新终端里排上命令:%q tab=%s", a.ide.tpc().initCmd, a.ide.bottomTab)
	}
	// 重新检测:丢掉会话,下一次取用重建
	a.lspRetry(s)
	if a.lspH.sess[s.root+"|go"] != nil {
		t.Fatal("重新检测应丢掉旧会话")
	}
}

// 重命名(含文档注释共 3 处)与格式化走服务器,结果落到缓冲区且可撤销。
func TestLSPRenameAndFormat(t *testing.T) {
	if _, ok := lsp.ForFile("a.go").Find(); !ok {
		t.Skip("没有安装 gopls")
	}
	a, b, _ := lspProject(t)
	pump := func(until func() bool, what string) {
		t.Helper()
		deadline := time.Now().Add(60 * time.Second)
		for time.Now().Before(deadline) {
			b.frames(1)
			a.drainUI()
			if until() {
				return
			}
			time.Sleep(40 * time.Millisecond)
		}
		t.Fatalf("等不到:%s", what)
	}
	pump(func() bool {
		s := a.lspFor("main.go")
		return s != nil && s.state == lspReady && s.docs["main.go"] != nil
	}, "gopls 就绪并已同步文档")
	tb := a.ide.cur()
	decl := strings.Index(goDemoSrc, "func Greet") + 6
	tb.st.Select(decl, decl)
	a.applyRename("Greet", "Hello")
	pump(func() bool { return strings.Count(tb.buf.String(), "Hello") == 3 }, "三处都改成 Hello")
	if strings.Contains(tb.buf.String(), "Greet") {
		t.Fatalf("不应残留旧名:%s", tb.buf.String())
	}
	if !tb.buf.Undo() || !strings.Contains(tb.buf.String(), "Greet") {
		t.Fatal("重命名应可一次撤销")
	}

	// 格式化:把缩进弄乱
	tb.buf.Set("package main\n\nfunc  main( ) {\nx:=1\n_=x\n}\n")
	pump(func() bool {
		d := a.lspH.sess[a.lspFor("main.go").root+"|go"].docs["main.go"]
		return d != nil && d.bufVer == tb.buf.Version()
	}, "改动已同步")
	a.formatDoc()
	pump(func() bool { return strings.Contains(tb.buf.String(), "func main() {") }, "格式化完成")
	if !strings.Contains(tb.buf.String(), "\tx := 1") {
		t.Fatalf("格式化结果:%q", tb.buf.String())
	}
}
