package app

import (
	"path/filepath"
	"testing"

	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/prefs"
)

func TestApplyDiagPrefsLogLevel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "p.json")
	p := prefs.Open(path)
	p.Set(pPrivacyLvl, 0)
	a := New(Options{Prefs: p})
	a.ApplyDiagPrefs()
	if diag.CurrentLevel() != diag.LevelError {
		t.Fatalf("want error, got %v", diag.CurrentLevel())
	}
}
