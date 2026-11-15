package demo

import "testing"

// 默认(不开极端数据)的主演示仓库要与设计稿一致:三项改动、一条储藏、六条提交。
func TestRouterDemoRepoDefaultMatchesDesign(t *testing.T) {
	r := Repo()
	cs, _ := r.Status()
	if len(cs) != 3 {
		t.Fatalf("默认应只有设计稿的三项改动,实际 %d", len(cs))
	}
	if st, _ := r.Stashes(); len(st) != 1 {
		t.Fatalf("默认应只有 1 条储藏:%d", len(st))
	}
	if log, _ := r.Log(300); len(log) != 6 {
		t.Fatalf("默认应只有 6 条提交:%d", len(log))
	}
}

func TestRouterDemoRepoPreservesSnapshotCommits(t *testing.T) {
	SetStress(true)
	defer SetStress(false)
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
