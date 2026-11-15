package app

import (
	"testing"

	"github.com/egoist/mygo/ui/editor"
	"github.com/ixxxxoooo/minicode/internal/lsp"
)

func TestLspPosEndUTF16LineEnd(t *testing.T) {
	// 中文行尾:一个字 = 1 字节列、2 个 UTF-16 单元
	src := "你好\n"
	buf := editor.NewBuffer(src)
	tb := &ideTab{buf: buf, st: &editor.State{Buffer: buf}}
	line := buf.Line(0)
	want := lsp.ColToUTF16(line, len(line))
	off := buf.Offset(editor.Pos{Line: 1, Col: 0})
	end := lspPosEnd(tb, off)
	if end.Line != 0 || end.Character != want {
		t.Fatalf("行尾 End 期望 (0,%d) 得 (%d,%d)", want, end.Line, end.Character)
	}
}
