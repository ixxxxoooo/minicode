//go:build !darwin || !cgo

package macos

func SystemIsDark() bool { return false }
