package demo

import "testing"

func TestRouterDemoRepoPreservesSnapshotCommits(t *testing.T) {
	r := Repo()
	log, err := r.Log(10)
	if err != nil || len(log) < 6 {
		t.Fatalf("log: %v len=%d", err, len(log))
	}
	if log[0].Hash != "a3f9c21" || log[5].Hash != "f5521ab" {
		t.Fatalf("preserve commits: %s … %s", log[0].Hash, log[5].Hash)
	}
	cs, _ := r.Status()
	if len(cs) < 300 {
		t.Fatalf("expected bulk changes, got %d", len(cs))
	}
	long, _ := r.Log(300)
	if len(long) < 200 {
		t.Fatalf("expected 200+ commits, got %d", len(long))
	}
	st, _ := r.Stashes()
	if len(st) < 5 {
		t.Fatalf("stashes: %d", len(st))
	}
}
