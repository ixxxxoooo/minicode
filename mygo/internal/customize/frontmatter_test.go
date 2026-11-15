package customize

import "testing"

func TestSplitFrontmatter(t *testing.T) {
	fm, body := splitFrontmatter("---\ndescription: hi\n---\n# Title\n")
	if fm == "" || body == "" {
		t.Fatal("expected frontmatter split")
	}
	if got := fmLine(fm, "description"); got != "hi" {
		t.Fatalf("desc=%q", got)
	}
}

func TestValidNameCommand(t *testing.T) {
	s := NewStore(map[Cat][]*Item{}, nil)
	if err := s.ValidName(Commands, "bad", nil); err == nil {
		t.Fatal("command must start with /")
	}
}
