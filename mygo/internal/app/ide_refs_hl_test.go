package app

import (
	"strings"
	"testing"
)

func TestRefLineSpansSyntax(t *testing.T) {
	a, _ := newTestApp(t)
	line := `func Greet(name string) {`
	sp := a.refLineSpans("main.go", line, 5, 10)
	joined := ""
	for _, s := range sp {
		joined += s.Text
	}
	if joined != line {
		t.Fatalf("span 应覆盖整行:%q", joined)
	}
	hasKw := false
	for _, s := range sp {
		if s.Text == "func" && s.Color.A > 0 {
			hasKw = true
		}
	}
	if !hasKw {
		t.Fatal("应给关键字着色")
	}
	hit := false
	for _, s := range sp {
		if strings.Contains(s.Text, "Greet") && s.Background.A > 0 {
			hit = true
		}
	}
	if !hit {
		t.Fatal("命中词应有底色")
	}
}