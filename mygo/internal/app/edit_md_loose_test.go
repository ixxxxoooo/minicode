package app

import "testing"

// 「写法与预览序列化不完全一致」的提示只统计真会被改写的写法:
// 段内软换行、引用 > 前缀、围栏语言标注、分隔线、有序列表编号都不算。
func TestMdNormalizedIgnoresEquivalentSyntax(t *testing.T) {
	same := []string{
		"第一行\n第二行",
		"> 引用第一行\n> 引用第二行",
		"```sh\ncd mygo\n```",
		"---",
		"2. 第二项",
	}
	for _, s := range same {
		if mdBlockWouldNormalize(s) {
			t.Errorf("%q 内容不会变,不应算「会被规范化」", s)
		}
	}
	if !mdBlockWouldNormalize("* 用星号的列表项") {
		t.Error("列表符 * → - 会被改写,应算规范化")
	}
	if got := mdNormalized("段一\n续行\n\n---\n\n```go\nx\n```\n"); got != 0 {
		t.Errorf("只含等价写法的文档应为 0,实际 %d", got)
	}
}
