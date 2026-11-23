//go:build !darwin || !cgo

package macos

func DockRequestAttention(bool) int { return 0 }
func DockCancelAttention(int)       {}
func DockSetProgress(float64)       {}
