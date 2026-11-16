package app

import (
	"path/filepath"
	"testing"

	"github.com/ixxxxoooo/minicode/internal/prefs"
)

func TestMigrateWaitAnim(t *testing.T) {
	p := prefs.Open(filepath.Join(t.TempDir(), "a.json"))
	p.Set(pWaitAnim, 0)
	migrateWaitAnim(p)
	if p.Int(pWaitAnim, -1) != 1 {
		t.Fatalf("old ring -> 1, got %d", p.Int(pWaitAnim, -1))
	}
	p2 := prefs.Open(filepath.Join(t.TempDir(), "b.json"))
	migrateWaitAnim(p2)
	if p2.Has(pWaitAnim) {
		t.Fatal("default should stay unset (沙漏)")
	}
	if p2.Int(pWaitAnimRev, 0) != 1 {
		t.Fatal("rev")
	}
}