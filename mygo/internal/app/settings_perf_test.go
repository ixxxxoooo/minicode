package app

import (
	"strings"
	"testing"

	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/prefs"
)

func TestSettingsPerfRenders(t *testing.T) {
	_, tt, _ := settingsApp(t, "perf")
	body := tt.Text()
	if !strings.Contains(body, "悬停") {
		t.Fatal("趋势区应包含悬停提示")
	}
	if !strings.Contains(body, "帧率") && !strings.Contains(body, "数据不足") {
		t.Fatalf("性能页应渲染概览或空态,得到: %q", trunc(body, 240))
	}
}

func TestPerfTabAlias(t *testing.T) {
	if got := resolveSetTabID("diagnostics"); got != "perf" {
		t.Fatalf("diagnostics → perf, got %q", got)
	}
}

func TestLogLevelPersistsViaPrefs(t *testing.T) {
	_, _, path := settingsApp(t, "perf")
	p := prefs.Open(path)
	p.Set(pPrivacyLvl, 3)
	a, tt, _ := settingsApp(t, "perf")
	a.prefs = p
	a.ApplyDiagPrefs()
	tt.Frame()
	if diag.CurrentLevel() != diag.LevelDebug {
		t.Fatalf("应为 DEBUG,得到 %v", diag.CurrentLevel())
	}
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
