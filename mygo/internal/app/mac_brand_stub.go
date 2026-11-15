//go:build !darwin || nocgo

package app

// loadIcnsAsPNG 非 macOS 或无 cgo 时不可用。
func loadIcnsAsPNG(path string) ([]byte, bool) { return nil, false }
