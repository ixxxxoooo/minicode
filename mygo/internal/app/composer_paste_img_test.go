package app

import (
	"os"
	"strings"
	"testing"

	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
)

// 会话输入框粘贴剪贴板图片:落盘到 MINI_HOME/paste,草稿里出现 `![](路径)`,缩略图行能解析到它。
// @author ixxxxoooo
func TestComposerPasteImage(t *testing.T) {
	t.Setenv("MINI_HOME", t.TempDir())
	a, tt := sessionApp(t, "flow-gaps-b4")
	a.draft = ""
	a.ux.focusDraft = true // 下一帧把焦点给输入框
	tt.Frame()
	png := []byte("\x89PNG\r\n\x1a\nfake")
	tt.SetClipboardData(transfer.New(transfer.NewItem(transfer.Bytes(transfer.PNG, png))))
	tt.Key(ui.Cmd, ui.KeyV)
	tt.Frame()
	atts := parseDraftAttaches(a.draft)
	if len(atts) != 1 {
		t.Fatalf("粘贴图片后草稿应有 1 个图片附件,草稿=%q", a.draft)
	}
	if !strings.HasSuffix(atts[0].Path, ".png") {
		t.Fatalf("应存成 png: %q", atts[0].Path)
	}
	got, err := os.ReadFile(a.absAttach(atts[0].Path))
	if err != nil || string(got) != string(png) {
		t.Fatalf("落盘内容不符: %v %q", err, got)
	}
}
