package app

import "testing"

func TestParseConflictsKeepsBase(t *testing.T) {
	src := `a
<<<<<<< HEAD
cur
||||||| base
ancestor
=======
inc
>>>>>>> other
b`
	segs := parseConflicts(src)
	if len(segs) != 3 {
		t.Fatalf("segments %d", len(segs))
	}
	if !segs[1].conflict || len(segs[1].base) != 1 || segs[1].base[0] != "ancestor" {
		t.Fatalf("base %+v", segs[1].base)
	}
}
