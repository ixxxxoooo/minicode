package app

import "testing"

// 用户文字里的图片占位标记只用于显示时去掉,原文不变。
func TestUserDisplayText(t *testing.T) {
	cases := map[string]string{
		"[Image #16] 这里耗时写在前面":            "这里耗时写在前面",
		"看这个 [Image #3] 和 [Image #4] 的差异": "看这个 和 的差异",
		"[Image #1]":       "",
		"没有图片":             "没有图片",
		"数组 a[Image] 不是标记": "数组 a[Image] 不是标记",
	}
	for in, want := range cases {
		if got := userDisplayText(in); got != want {
			t.Errorf("userDisplayText(%q) = %q,期望 %q", in, got, want)
		}
	}
}
