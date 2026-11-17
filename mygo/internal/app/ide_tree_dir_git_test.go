package app

import (
	"testing"

	"github.com/ixxxxoooo/minicode/internal/vcs"
)

func TestDirGitAggPriorityAndCounts(t *testing.T) {
	a := New(Options{Mode: ModeIDE})
	m := map[string]string{
		"src/a.go": "U",
		"src/b.go": "M",
		"src/c.go": "D",
	}
	a.rebuildTreeDirGit(m)
	agg := a.ide.dirGit["src"]
	if agg.worst != 'D' {
		t.Fatalf("目录最严重状态应为 D,得 %q", agg.worst)
	}
	if agg.untracked != 1 || agg.modified != 1 || agg.deleted != 1 {
		t.Fatalf("计数不对:%+v", agg)
	}
	if tip := agg.tooltip(); tip == "" {
		t.Fatal("应有 tooltip")
	}
}

func TestDirGitConflictBeatsModify(t *testing.T) {
	a := New(Options{Mode: ModeIDE})
	m := map[string]string{
		"pkg/x.go": "M",
		"pkg/y.go": string(vcs.StatusConflict),
	}
	a.rebuildTreeDirGit(m)
	if a.ide.dirGit["pkg"].worst != vcs.StatusConflict {
		t.Fatalf("冲突应优先,得 %q", a.ide.dirGit["pkg"].worst)
	}
}
