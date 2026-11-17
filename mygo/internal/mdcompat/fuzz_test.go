package mdcompat

import (
	"strings"
	"testing"
	"testing/quick"
)

// 截断与随机输入:不 panic,且内容性字符不丢失。
func TestParseNeverPanics(t *testing.T) {
	f := func(s string) bool {
		if len(s) > 12000 {
			s = s[:12000]
		}
		func() {
			defer func() {
				if recover() != nil {
					t.Fatalf("panic on %q", truncate(s, 80))
				}
			}()
			d := Parse(s)
			plain := PlainText(d)
			sub := LetterMultiset(Prepare(s))
			sup := LetterMultiset(plain)
			if len(sub) > 0 && !MultisetContains(sup, sub) {
				t.Fatalf("丢内容 len=%d\nsrc=%q\nplain=%q", len(s), truncate(s, 120), truncate(plain, 120))
			}
		}()
		return true
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 200}); err != nil {
		t.Fatal(err)
	}
}

func TestTruncatedInputs(t *testing.T) {
	seeds := []string{
		"| a | b |\n| --- | --- |\n| 1 | 2 |",
		"```\nunclosed\n",
		"**bold",
		"<div>html\n",
		"| col1 | col2 |\n| --- |",
		"1. item\n   ```\n   code",
		"混合　全角\t制表符",
		"\u200b零宽\u200b",
	}
	for _, seed := range seeds {
		for cut := 1; cut <= len(seed); cut++ {
			part := seed[:cut]
			d := Parse(part)
			plain := PlainText(d)
			sub := LetterMultiset(Prepare(part))
			sup := LetterMultiset(plain)
			if len(sub) > 0 && !MultisetContains(sup, sub) {
				t.Fatalf("截断 %d/%d 丢内容:\npart=%q\nplain=%q", cut, len(seed), part, plain)
			}
		}
	}
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", "\\n")
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
