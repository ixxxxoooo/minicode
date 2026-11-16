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
		"区块标题「<工作区名>」与 node_modules、~28 个、[…]",
		"画布 ` ```canvas ` 围栏",
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

// 序列化器把「**中文**:全角标点」整块写成 <strong> 时,换回 **;原文自己写的 <strong> 不动。
func TestMdFixMarks(t *testing.T) {
	if got := mdFixMarks("- **看门狗**:x", "- <strong>看门狗</strong>:x"); got != "- **看门狗**:x" {
		t.Errorf("应换回 **:%q", got)
	}
	if got := mdFixMarks("原文 <strong>a</strong>", "原文 <strong>a</strong>"); got != "原文 <strong>a</strong>" {
		t.Errorf("原文自带的标签不应被改:%q", got)
	}
	if mdBlockWouldNormalize("- **看门狗会记栈**:主线程停滞超过 400 ms,`diag.Watchdog` 每 100 ms 记一次。") {
		t.Error("加粗被序列化成 HTML 后换回,不应再算规范化")
	}
}
