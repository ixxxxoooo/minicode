package app

import "testing"

func TestHashTextStable(t *testing.T) {
	a := hashText("hello")
	b := hashText("hello")
	if a != b || a == "" {
		t.Fatalf("hashText 应稳定: %q vs %q", a, b)
	}
	if hashText("x") == hashText("y") {
		t.Fatal("不同文本不应同哈希")
	}
}
