package app

import (
	"strings"
	"testing"
)

func TestThinkPreviewPlain(t *testing.T) {
	s := "a\nb\n" + strings.Repeat("长", 300)
	got := thinkPreviewPlain(s)
	if got == "" {
		t.Fatal("empty")
	}
	if strings.Contains(got, "\n") {
		t.Fatal("newlines remain")
	}
	if len([]rune(got)) > thinkPreviewRunes {
		t.Fatalf("too long %d", len([]rune(got)))
	}
}

func TestThinkMqTarget(t *testing.T) {
	a := &App{}
	if a.thinkMqTarget("short", 10) != 0 {
		t.Fatal("short")
	}
	off := a.thinkMqTarget("12345678901234567890", 10)
	if off <= 0 {
		t.Fatal("want scroll")
	}
}