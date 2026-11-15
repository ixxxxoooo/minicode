package app

import (
	"testing"

	"github.com/ixxxxoooo/minicode/internal/term"
)

// 含宽字符的行按格子切段:窄字符连续成段,每个宽字符单独一段,列号按「宽字符 2 格」累计。
func TestTermSegmentsWideChars(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	ln := term.Line{{Text: "ab你好c"}, {Text: "d", Bold: true}}
	if !lineHasWide(ln) {
		t.Fatal("应识别出宽字符")
	}
	if lineHasWide(term.Line{{Text: "plain ascii ─ box"}}) {
		t.Fatal("制表符等单宽字符不该算宽")
	}
	var got [][2]string
	for _, s := range termSegments(a.tk, ln) {
		got = append(got, [2]string{itoa(s.col), s.span.Text})
	}
	want := [][2]string{{"0", "ab"}, {"2", "你"}, {"4", "好"}, {"6", "c"}, {"7", "d"}}
	if len(got) != len(want) {
		t.Fatalf("分段 %v,想要 %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("第 %d 段 %v,想要 %v", i, got[i], want[i])
		}
	}
}
