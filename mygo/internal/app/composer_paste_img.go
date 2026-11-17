package app

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/diag"
)

// 会话输入框粘贴剪贴板图片:存到 ~/.minicode/paste/,作为图片附件加进草稿(输入框上方出缩略图卡,
// 发送时拆成 Images),与「添加图片…」/ 拖入图片同一种附件。
//
// @author ixxxxoooo

const composerPasteImageMax = 10 << 20 // 10 MiB,与 Markdown 粘贴图片一致

// composerClipboardImage 从剪贴板读 PNG / JPEG。
// 参数:svc 构建界面时取的 c.Services()(回调里不能用 *ui.Context)。
// 返回:图片字节、扩展名;剪贴板里没有图片时 raw 为空。
func composerClipboardImage(svc ui.Services) (raw []byte, ext string) {
	const jpegFmt = transfer.Format("image/jpeg")
	d := svc.ReadClipboardData(transfer.PNG, jpegFmt)
	if b, err := d.Read(transfer.PNG); err == nil && len(b) > 0 {
		return b, ".png"
	}
	if b, err := d.Read(jpegFmt); err == nil && len(b) > 0 {
		return b, ".jpg"
	}
	return nil, ""
}

// composerPasteImage 输入框的粘贴回调里调用:剪贴板有图片就落盘并排队插入。
// 参数:svc 构建界面时取的 c.Services()。
// 返回:true = 已接管这次粘贴(默认粘贴取消);false = 不是图片,照常粘贴文本。
func (a *App) composerPasteImage(svc ui.Services) bool {
	raw, ext := composerClipboardImage(svc)
	if len(raw) == 0 {
		return false
	}
	if len(raw) > composerPasteImageMax {
		a.toast = "剪贴板图片超过 10 MB,未粘贴"
		attachLog.Warn("粘贴图片过大", "bytes", len(raw))
		return true
	}
	dir := filepath.Join(diag.Home(), "paste")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		a.toast = "无法保存图片:" + err.Error()
		attachLog.Warn("创建粘贴目录失败", "err", err)
		return true
	}
	path := filepath.Join(dir, fmt.Sprintf("%d%s", time.Now().UnixNano(), ext))
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		a.toast = "无法保存图片:" + err.Error()
		attachLog.Warn("写入粘贴图片失败", "err", err)
		return true
	}
	a.addDraftImage(path) // 缩略图卡显示在输入框上方,正文里不露路径
	attachLog.Info("粘贴图片为附件", "bytes", len(raw), "path", path)
	return true
}
