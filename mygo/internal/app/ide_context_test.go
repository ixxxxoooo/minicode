package app

import (
	"strings"
	"testing"

	"github.com/ixxxxoooo/minicode/internal/core"
)

func openAnyFile(t *testing.T, a *App) *ideTab {
	t.Helper()
	for _, p := range []string{"src/main.rs", "src/router.rs", "README.md"} {
		a.ide.openFile(p, true)
		if tb := a.ide.cur(); tb != nil && tb.buf != nil && tb.path == p {
			return tb
		}
	}
	t.Skip("演示沙盒里没有可打开的文件")
	return nil
}

// 打开了文件:上下文含当前文件与选区;相同内容只发一次;关闭偏好后不发。
func TestIDEContextAttachOnce(t *testing.T) {
	a, tt := newTestApp(t)
	tb := openAnyFile(t, a)
	tt.Frame()
	tb.st.Select(0, 5)
	ctx := a.ideContext()
	if !strings.Contains(ctx, tb.path) || !strings.Contains(ctx, "选中的代码") {
		t.Fatalf("上下文应含文件与选区:%q", ctx)
	}
	p := a.attachIDEContext(core.CmdPrompt{Text: "hi"})
	if p.Context == "" {
		t.Fatal("第一次应附带上下文")
	}
	if p2 := a.attachIDEContext(core.CmdPrompt{Text: "again"}); p2.Context != "" {
		t.Fatal("内容没变不应重复发送")
	}
	tb.st.Select(1, 8)
	if p3 := a.attachIDEContext(core.CmdPrompt{Text: "changed"}); p3.Context == "" {
		t.Fatal("选区变了应重新发送")
	}
	a.prefs.Set(pIDECtx, false)
	a.ctxSent = ""
	if a.ideContext() != "" {
		t.Fatal("关闭偏好后不应有上下文")
	}
}

// 选区超长要截断。
func TestIDEContextTruncates(t *testing.T) {
	a, tt := newTestApp(t)
	tb := openAnyFile(t, a)
	tt.Frame()
	n := tb.buf.Len()
	tb.st.Select(0, n)
	ctx := a.ideContext()
	if len([]rune(ctx)) > ideCtxMaxSel+1500 {
		t.Fatalf("上下文过长:%d", len([]rune(ctx)))
	}
}
