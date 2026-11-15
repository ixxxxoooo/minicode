package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/richtext"
)

const mdPasteImageMax = 10 << 20 // 10 MiB

// mdTryPasteImage 从剪贴板读图片,落盘并插入 Markdown(源码写语法,WYSIWYG 写图片语法进文档)。
// 返回 true 表示已处理,调用方不要再走默认粘贴。
func (a *App) mdTryPasteImage(c *ui.Context, tb *ideTab, md *mdState) bool {
	raw, ext, err := mdReadClipboardImage(c)
	if err != nil || len(raw) == 0 {
		return false
	}
	if len(raw) > mdPasteImageMax {
		a.toast = "剪贴板图片超过 10 MB,未粘贴"
		mdImgLog.Warn("粘贴图片过大", "bytes", len(raw))
		return true
	}
	rel, saveErr := a.mdSaveClipboardImage(raw, ext)
	if saveErr != nil {
		a.toast = "无法保存图片:" + saveErr.Error()
		return true
	}
	if a.mdSource(tb) {
		a.insertMdImage(tb, rel, "")
		a.toast = "已粘贴图片 " + rel
		mdImgLog.Info("源码粘贴图片", "dest", rel)
		return true
	}
	if md == nil || md.doc == nil {
		return false
	}
	anchor, caret := md.doc.Selection()
	snip := "![](" + rel + ")\n"
	md.doc.Edit(func(e *richtext.Edit) {
		e.ReplaceString(anchor, caret, snip, richtext.Attributes{})
	})
	a.mdWrite(tb, md)
	a.toast = "已粘贴图片 " + rel
	mdImgLog.Info("预览粘贴图片", "dest", rel)
	return true
}

func mdReadClipboardImage(c *ui.Context) (raw []byte, ext string, err error) {
	const jpegFmt = transfer.Format("image/jpeg")
	svc := c.Services().ReadClipboardData(transfer.PNG, jpegFmt)
	raw, err = svc.Read(transfer.PNG)
	ext = ".png"
	if err != nil || len(raw) == 0 {
		raw, err = svc.Read(jpegFmt)
		if err != nil || len(raw) == 0 {
			return nil, "", err
		}
		ext = ".jpg"
	}
	return raw, ext, nil
}

func (a *App) mdSaveClipboardImage(raw []byte, ext string) (rel string, err error) {
	dir := a.mdAssetDir()
	name := mdUniqueAssetName(a.ide.fs.Abs(dir), "paste-"+itoa(int(time.Now().Unix()%1e9))+ext)
	outAbs := a.ide.fs.Abs(filepath.Join(dir, name))
	if err := os.MkdirAll(filepath.Dir(outAbs), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(outAbs, raw, 0o644); err != nil {
		return "", err
	}
	a.ide.invalidate()
	return filepath.ToSlash(filepath.Join(dir, name)), nil
}

// mdUniqueAssetName 在目录中生成不冲突的文件名。
func mdUniqueAssetName(dirAbs, base string) string {
	if _, err := os.Stat(filepath.Join(dirAbs, base)); os.IsNotExist(err) {
		return base
	}
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	for n := 2; n < 10000; n++ {
		cand := fmt.Sprintf("%s-%d%s", stem, n, ext)
		if _, err := os.Stat(filepath.Join(dirAbs, cand)); os.IsNotExist(err) {
			return cand
		}
	}
	return base
}
