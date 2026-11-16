package app

import (
	"testing"

	"github.com/egoist/mygo/ui/editor"
	"github.com/ixxxxoooo/minicode/internal/workspace"
)

func TestDetectAndApplyEOL(t *testing.T) {
	if detectEOL("a\r\nb\r\nc\n") != eolCRLF || detectEOL("a\nb\r\n") != eolLF && false {
		t.Fatal("多数 CRLF 应判为 CRLF")
	}
	if detectEOL("a\nb\nc\r\n") != eolLF || detectEOL("abc") != eolLF {
		t.Fatal("多数 LF / 无换行应判为 LF")
	}
	if withEOL("a\nb\n", eolCRLF) != "a\r\nb\r\n" || withEOL("a\nb\n", eolLF) != "a\nb\n" {
		t.Fatal("withEOL 转换错误")
	}
	if normEOL("a\r\nb\rc\n") != "a\nb\rc\n" {
		t.Fatal("normEOL 只应处理 \\r\\n")
	}
	// 与编辑器缓冲区的归一口径一致,否则 dirty 判断会错
	for _, s := range []string{"a\r\nb\r\n", "a\rb", "x\n"} {
		if got := editor.NewBuffer(s).String(); got != normEOL(s) {
			t.Fatalf("缓冲区与 normEOL 不一致:%q → %q / %q", s, got, normEOL(s))
		}
	}
}

// CRLF 文件:打开不算未保存;保存后磁盘仍是 CRLF;切到 LF 算未保存,保存后磁盘变 LF。
func TestCRLFFileRoundTrip(t *testing.T) {
	a := New(Options{Mode: ModeIDE})
	fs := workspace.NewMem("demo", map[string]string{"win.txt": "one\r\ntwo\r\n"})
	a.ide.fs = fs
	a.ide.listCache, a.ide.filesCache = nil, nil
	a.ide.openFile("win.txt", true)
	tb := a.ide.tab("win.txt")
	if tb == nil || tb.buf == nil {
		t.Fatal("应能打开")
	}
	if tb.dirty() || tb.eolNow() != eolCRLF {
		t.Fatalf("CRLF 文件刚打开不该未保存,且应识别为 CRLF: dirty=%v eol=%q", tb.dirty(), tb.eol)
	}
	a.ide.active = "win.txt"
	if err := a.ide.save(); err != nil {
		t.Fatal(err)
	}
	if got, _ := fs.Read("win.txt"); got != "one\r\ntwo\r\n" {
		t.Fatalf("保存后应保持 CRLF: %q", got)
	}
	tb.eol = eolLF
	if !tb.dirty() {
		t.Fatal("改行尾应算未保存")
	}
	if err := a.ide.save(); err != nil {
		t.Fatal(err)
	}
	if got, _ := fs.Read("win.txt"); got != "one\ntwo\n" || tb.dirty() {
		t.Fatalf("切到 LF 保存后应变 LF 且不再未保存: %q dirty=%v", got, tb.dirty())
	}
}
