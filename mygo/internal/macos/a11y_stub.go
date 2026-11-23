//go:build !darwin || !cgo

package macos

func ReduceTransparency() bool { return false }
func IncreaseContrast() bool   { return false }
func ReduceMotion() bool       { return false }
func PostAnnouncement(string)  {}
