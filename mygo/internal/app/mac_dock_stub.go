//go:build !darwin || !cgo

package app

func dockRequestAttention(bool) int { return 0 }
func dockCancelAttention(int)       {}
func dockSetProgress(float64)       {}
