package customize

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ixxxxoooo/minicode/internal/diag"
)

var alog = diag.L("customize")

// writeAtomic 先备份再原子写入;父目录不存在会自动创建。
func writeAtomic(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		bak := path + ".bak-" + time.Now().Format("20060102-150405")
		if err := os.Rename(path, bak); err != nil {
			return fmt.Errorf("备份失败: %w", err)
		}
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, mode); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// quarantine 把损坏的 JSON 改名留底。
func quarantine(path string, reason error) {
	if path == "" {
		return
	}
	if _, err := os.Stat(path); err != nil {
		return
	}
	bad := path + ".bad-" + time.Now().Format("20060102-150405")
	if err := os.Rename(path, bad); err == nil {
		alog.Warn("配置损坏,已改名留底", "path", path, "reason", reason)
	}
}
