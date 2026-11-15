package claude

import (
	"os"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// ReadTail 读文件末尾最多 max 字节(用于任务 output_file 的增量同步)。
func ReadTail(path string, max int) (string, error) {
	if path == "" {
		return "", nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if len(b) <= max {
		return string(b), nil
	}
	cut := len(b) - max
	for cut < len(b) && b[cut]&0xC0 == 0x80 {
		cut++
	}
	return string(b[cut:]), nil
}

// TidyTaskLog 整理输出后再用于终端卡展示。
func TidyTaskLog(s string) string { return core.TidyLog(s) }
