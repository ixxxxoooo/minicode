package app

import (
	"testing"

	"github.com/ixxxxoooo/minicode/internal/prefs"
)

func TestScmViewModePref(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true, Prefs: prefs.Memory()})
	a.prefs.Set(pScmView, "tree")
	if a.scmViewMode() != "tree" {
		t.Fatal("应为树形")
	}
	a.toggleScmViewMode()
	if a.scmViewMode() != "list" {
		t.Fatal("切换后应为列表")
	}
}
