package vcs

import "testing"

func TestMergeLineRange(t *testing.T) {
	base := "a\nb\nc\nd\n"
	src := "A\nB\nC\nD\n"
	got := MergeLineRange(base, src, 1, 2)
	want := "a\nB\nC\nd\n"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
