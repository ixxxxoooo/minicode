package app

import (
	"testing"

	"github.com/ixxxxoooo/minicode/internal/theme"
)

func TestAuthorAbbr(t *testing.T) {
	if authorAbbr("") != "?" {
		t.Fatal("空作者")
	}
	if authorAbbr("Lin") != "Lin" {
		t.Fatal("短名")
	}
	if authorAbbr("Alice") != "Al" {
		t.Fatal("缩写")
	}
}

func TestBlameHashColorStable(t *testing.T) {
	tk := theme.Resolve(theme.Find("paper"), theme.FindScheme("a"), "")
	c1 := blameHashColor("abc123", tk)
	c2 := blameHashColor("abc123", tk)
	if c1 != c2 {
		t.Fatal("同色哈希应同色")
	}
}
