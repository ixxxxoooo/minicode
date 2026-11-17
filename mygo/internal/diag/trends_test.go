package diag

import (
	"testing"
)

func TestReadLogLinesFilter(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MINI_HOME", dir)
	openLog(LogPath())
	L("alpha").Info("可见")
	L("beta").Debug("隐藏")
	L("alpha").Warn("警告行")
	got, err := ReadLogLines(50, LevelInfo, "alpha", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 lines, got %d", len(got))
	}
}

func TestPerformanceSnapshotJSON(t *testing.T) {
	ResetAllMetrics()
	b := PerformanceSnapshotJSON()
	if len(b) < 10 {
		t.Fatal("snapshot too short")
	}
}
