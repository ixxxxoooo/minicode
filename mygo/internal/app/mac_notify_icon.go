package app

import (
	"crypto/sha1"
	"encoding/hex"
	"os"
	"path/filepath"
)

// notifyIconFile 把通知用的应用图标 PNG 落成临时文件并返回路径(按内容哈希命名,同一图标只写一次)。
// 参数 png 为当前样式的图标字节;返回值为文件路径,没有图标或写失败返回空串(通知照发,只是不带图标附件)。
// @author ixxxxoooo
func notifyIconFile(png []byte) string {
	if len(png) == 0 {
		return ""
	}
	sum := sha1.Sum(png)
	p := filepath.Join(os.TempDir(), "minicode-notify-icon-"+hex.EncodeToString(sum[:6])+".png")
	if st, err := os.Stat(p); err == nil && st.Size() == int64(len(png)) {
		return p
	}
	if err := os.WriteFile(p, png, 0o600); err != nil {
		log.Warn("写通知图标临时文件失败", "err", err)
		return ""
	}
	return p
}
