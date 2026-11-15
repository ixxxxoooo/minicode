package app

import (
	"os"
	"path/filepath"
	"time"

	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/richtext"
)

// mdTryPasteImage 从剪贴板读图片,落盘并插入 Markdown(源码写语法,WYSIWYG 写图片语法进文档)。
// 返回 true 表示已处理,调用方不要再走默认粘贴。
func (a *App) mdTryPasteImage(c *ui.Context, tb *ideTab, md *mdState) bool {
	if tb == nil || md == nil || md.doc == nil {
		return false
	}
	raw, err := c.Services().ReadClipboardData(transfer.PNG).Read(transfer.PNG)
	if err != nil || len(raw) == 0 {
		return false
	}
	ext := ".png"
	name := "paste-" + itoa(int(time.Now().Unix()%1e9)) + ext
	dir := a.mdAssetDir()
	outAbs := a.ide.fs.Abs(filepath.Join(dir, name))
	if err := os.MkdirAll(filepath.Dir(outAbs), 0o755); err != nil {
		a.toast = "无法保存图片:" + err.Error()
		return true
	}
	if err := os.WriteFile(outAbs, raw, 0o644); err != nil {
		a.toast = "无法保存图片:" + err.Error()
		return true
	}
	a.ide.invalidate()
	rel := filepath.ToSlash(filepath.Join(dir, name))
	if a.mdSource(tb) {
		a.insertMdImage(tb, rel, "")
		return true
	}
	anchor, caret := md.doc.Selection()
	snip := "![](" + rel + ")\n"
	md.doc.Edit(func(e *richtext.Edit) {
		e.ReplaceString(anchor, caret, snip, richtext.Attributes{})
	})
	a.mdWrite(tb, md)
	a.toast = "已粘贴图片 " + rel
	mdImgLog.Info("剪贴板粘贴图片", "dest", rel)
	return true
}
