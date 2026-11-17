package claude

import (
	"strings"
	"testing"

	"github.com/ixxxxoooo/minicode/internal/core"
)

func TestResumeOmitsPermissionModeFlag(t *testing.T) {
	m := core.ModePlan
	args := Options{Resume: "s1", Mode: &m}.Args()
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--permission-mode" {
			t.Fatalf("恢复会话不应带 --permission-mode: %v", args)
		}
	}
}

func TestNewSessionKeepsPermissionModeFlag(t *testing.T) {
	m := core.ModeBypassPermissions
	args := Options{SessionID: "s1", Mode: &m}.Args()
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--permission-mode bypassPermissions") {
		t.Fatalf("新会话应带模式: %s", joined)
	}
	if !strings.Contains(joined, "--allow-dangerously-skip-permissions") {
		t.Fatal("应放行绕过选项")
	}
}
