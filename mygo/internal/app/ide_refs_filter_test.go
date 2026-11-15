package app

import "testing"

// 引用计数只数代码,不数注释与 package 声明。
func TestCodePartSkipsComments(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"// add 计算两个数", ""},
		{"  /* add */", ""},
		{" * add", ""},
		{"package main", ""},
		{"# add 注释", ""},
		{"x := add(1, 2) // add 的调用", "x := add(1, 2)"},
		{`s := "a // b" + add()`, `s := "a // b" + add()`},
		{"func add() {}", "func add() {}"},
	} {
		if got := codePart(c.in); got != c.want {
			t.Fatalf("codePart(%q)=%q,期望 %q", c.in, got, c.want)
		}
	}
}
