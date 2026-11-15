//go:build darwin

package kit

import (
	"os/exec"
	"strings"
)

// SystemReduceMotion 读 macOS「减弱动态效果」;读不到则视为 false。
func SystemReduceMotion() bool {
	out, err := exec.Command("defaults", "read", "com.apple.universalaccess", "reduceMotion").Output()
	if err != nil {
		return false
	}
	s := strings.TrimSpace(string(out))
	return s == "1" || strings.EqualFold(s, "true")
}
