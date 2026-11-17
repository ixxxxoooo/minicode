package app

import (
	"testing"

	"github.com/ixxxxoooo/minicode/internal/workspace"
)

func TestUniqueTreeDest(t *testing.T) {
	fs := workspace.NewMem("t", map[string]string{"a/x.txt": "1"}, nil)
	dest := uniqueTreeDest("a", "x.txt", fs)
	if dest != "a/x 副本.txt" {
		t.Fatalf("首次副本 %q", dest)
	}
	fs.Write(dest, "2")
	dest2 := uniqueTreeDest("a", "x.txt", fs)
	if dest2 != "a/x 副本 2.txt" {
		t.Fatalf("第二次副本 %q", dest2)
	}
}