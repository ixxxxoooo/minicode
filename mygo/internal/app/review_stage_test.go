package app

import (
	"testing"

	"github.com/ixxxxoooo/minicode/internal/core"
)

func TestReviewStageAcceptedPaths(t *testing.T) {
	a, _ := newTestApp(t)
	a.rv.init()
	a.rv.hunk["src/a.go#0"] = "ok"
	files := []chFile{{path: "src/a.go", blocks: []*core.DiffBlock{{Path: "src/a.go"}}}}
	paths, n := a.reviewStageAcceptedPaths(files)
	if len(paths) != 1 || n != 1 {
		t.Fatalf("want 1 file, got %v n=%d", paths, n)
	}
	_, why := a.reviewStageAcceptedReason([]chFile{})
	if why == "" {
		t.Fatal("空列表应有置灰原因")
	}
}
