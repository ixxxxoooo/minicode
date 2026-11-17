package app

import (
	"testing"
	"time"

	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/prefs"
)

func TestSettingsPerfRenders(t *testing.T) {
	for i := 0; i < 24; i++ {
		diag.RecordUIFrame(9*time.Millisecond, 16*time.Millisecond)
	}
	_, tt, _ := settingsApp(t, "perf")
	if !tt.HasText("悬停") {
		t.Fatal("趋势区应包含悬停提示")
	}
	if !tt.HasText("帧率") && !tt.HasText("数据不足") {
		t.Fatal("性能页应渲染概览或空态")
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
