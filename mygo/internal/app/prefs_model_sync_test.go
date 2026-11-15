package app

import (
	"testing"

	"github.com/ixxxxoooo/minicode/internal/core"
)

func TestSyncLegacyModelPrefs(t *testing.T) {
	a, _ := newTestApp(t)
	a.prefs.Set(legacyModelIx, 1)
	a.prefs.Set(legacyEffort, int(core.EffortHigh))
	a.syncModelEffortPrefs()
	if a.prefs.Str(prefModel, "") != "opus" {
		t.Fatalf("应迁移到 claude.model=opus,得 %q", a.prefs.Str(prefModel, ""))
	}
	if a.prefs.Int(prefEffort, -1) != int(core.EffortHigh) {
		t.Fatalf("应迁移 effort:%d", a.prefs.Int(prefEffort, -1))
	}
	if a.prefs.Has(legacyModelIx) || a.prefs.Has(legacyEffort) {
		t.Fatal("迁移后应删除旧键")
	}
}
