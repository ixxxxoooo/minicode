//go:build !darwin || !cgo

package macos

// LoadIcnsAsPNG 非 macOS 或无 cgo 时不可用。
func LoadIcnsAsPNG(path string) ([]byte, bool) { return nil, false }

// SetFinderIcon 非 macOS 或无 cgo 时不可用。
func SetFinderIcon(path string, png []byte) bool { return false }
