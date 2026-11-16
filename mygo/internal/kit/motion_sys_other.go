//go:build !darwin || !cgo

package kit

// SystemReduceMotion 非 macOS(或无 cgo)时没有系统「减弱动态」设置。
func SystemReduceMotion() bool { return false }
