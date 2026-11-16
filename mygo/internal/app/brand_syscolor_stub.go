//go:build !darwin || !cgo

package app

func systemIsDark() bool { return false }
