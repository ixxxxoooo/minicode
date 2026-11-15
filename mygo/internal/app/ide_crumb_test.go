package app

import "testing"

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
