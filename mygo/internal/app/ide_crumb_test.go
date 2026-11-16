package app

import (
	"strings"
	"testing"

)

func TestSiblingSymbols(t *testing.T) {
	syms := []symbol{
		{Name: "Router", Kind: "struct", Line: 0, Parent: -1},
		{Name: "add", Kind: "fn", Line: 5, Parent: 0},
		{Name: "match_path", Kind: "fn", Line: 10, Parent: 0},
		{Name: "other", Kind: "fn", Line: 20, Parent: -1},
	}
	sibs := siblingSymbols(syms, 1)
	if len(sibs) != 2 || sibs[0] != 1 || sibs[1] != 2 {
		t.Fatalf("siblings under impl: %v", sibs)
	}
	top := siblingSymbols(syms, 0)
	if len(top) != 2 {
		t.Fatalf("top-level siblings: %v", top)
	}
}

func TestCrumbPathMenuTitle(t *testing.T) {
	a, _ := demoIDE(t)
	fs := a.ide.fs
	parts := strings.Split("src/router.rs", "/")
	if got := crumbPathMenuTitle(fs, parts, 0); got != fs.Name() {
		t.Fatalf("根段标题=%q,想要 %q", got, fs.Name())
	}
	if got := crumbPathMenuTitle(fs, parts, 1); got != "src" {
		t.Fatalf("文件段标题=%q", got)
	}
}

