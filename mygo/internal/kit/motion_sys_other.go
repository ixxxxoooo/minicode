//go:build !darwin

package kit

// SystemReduceMotion 非 macOS 无系统「减弱动态」设置。
func SystemReduceMotion() bool { return false }
