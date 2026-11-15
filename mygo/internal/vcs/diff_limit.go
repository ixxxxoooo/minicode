package vcs

import (
	"strconv"
	"strings"
)

// ParseDiffMaxBytes 解析设置里的 Diff 大小上限(如 "512 KB"、"2 MB");无法解析时返回 512*1024。
func ParseDiffMaxBytes(s string) int64 {
	s = strings.TrimSpace(strings.ToUpper(s))
	if s == "" {
		return 512 * 1024
	}
	mult := int64(1)
	switch {
	case strings.HasSuffix(s, "GB"):
		mult = 1 << 30
		s = strings.TrimSuffix(s, "GB")
	case strings.HasSuffix(s, "MB"):
		mult = 1 << 20
		s = strings.TrimSuffix(s, "MB")
	case strings.HasSuffix(s, "KB"):
		mult = 1 << 10
		s = strings.TrimSuffix(s, "KB")
	case strings.HasSuffix(s, "B"):
		s = strings.TrimSuffix(s, "B")
	}
	s = strings.TrimSpace(s)
	n, err := strconv.ParseFloat(s, 64)
	if err != nil || n <= 0 {
		return 512 * 1024
	}
	return int64(n * float64(mult))
}

// ExceedsDiffLimit 判断文件大小是否超过上限。
func ExceedsDiffLimit(size int64, maxStr string) bool {
	if size <= 0 {
		return false
	}
	return size > ParseDiffMaxBytes(maxStr)
}
