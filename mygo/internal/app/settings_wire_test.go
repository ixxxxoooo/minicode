package app

import (
	"testing"

	"github.com/ixxxxoooo/minicode/internal/prefs"
)

func TestMigrateModelPrefs(t *testing.T) {
	st := prefs.Memory()
	st.Set("models.model", 1)
	a := &App{prefs: st}
	a.MigrateSettingsPrefs()
	if a.prefs.Str(prefModel, "") != "opus" {
		t.Fatalf("应迁移到 claude.model=opus,得 %q", a.prefs.Str(prefModel, ""))
	}
}
