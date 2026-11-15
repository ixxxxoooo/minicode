//go:build !darwin || !cgo

package app

func a11yReduceTransparency() bool { return false }
func a11yIncreaseContrast() bool   { return false }
func a11yReduceMotion() bool       { return false }
func postA11yAnnouncement(string)  {}
