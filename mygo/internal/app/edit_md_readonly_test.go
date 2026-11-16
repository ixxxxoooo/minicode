package app

import "testing"

// 手动只读预览开关与自动只读判定。
func TestMarkdownManualReadOnly(t *testing.T) {
	a, tt := demoIDE(t)
	a.ide.openFile("docs/guide.md", true)
	tt.Frame()
	tb := a.ide.cur()
	if tb.md == nil || tb.md.unsafe != "" || a.mdPreviewReadOnly(tb) {
		t.Fatal("guide 默认可编辑预览")
	}
	a.toggleMdReadOnly(tb)
	if !tb.md.readOnly || !a.mdPreviewReadOnly(tb) {
		t.Fatalf("应进入手动只读:%+v", tb.md)
	}
	tt.Frame()
	if a.mdPreviewReadOnly(tb) && tb.md.doc != nil {
		// 只读视图走渲染分支,doc 可仍在内存里
	}
	a.toggleMdReadOnly(tb)
	if tb.md.readOnly || a.mdPreviewReadOnly(tb) {
		t.Fatal("再次切换应恢复可编辑")
	}

	a.ide.openFile("docs/embed.md", true)
	tt.Frame()
	emb := a.ide.cur()
	if emb.md.unsafe != "HTML" || !a.mdPreviewReadOnly(emb) {
		t.Fatalf("HTML 应自动只读:%+v", emb.md)
	}
	emb.md.readOnly = true
	if !a.mdPreviewReadOnly(emb) {
		t.Fatal("自动只读与手动标记都应锁定预览")
	}
}
