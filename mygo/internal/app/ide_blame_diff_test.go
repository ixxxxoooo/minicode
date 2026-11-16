package app

import (
	"testing"

	"github.com/ixxxxoooo/minicode/internal/vcs"
)

// 追溯卡「差异」:未提交的行直接打开 HEAD ↔ 工作区差异标签并激活,不再置灰。
func TestBlameCardDiffForUncommittedLine(t *testing.T) {
	a, tt := demoIDE(t)
	d := a.ide
	tb := d.tab("src/router.rs")
	if tb == nil {
		t.Fatal("演示项目里应有 src/router.rs")
	}
	id := diffID(dkHead, tb.path)
	if d.tab(id) != nil {
		t.Fatalf("前置条件:差异标签 %s 不该已存在", id)
	}
	a.blameOpenDiff(tb, vcs.BlameLine{Uncommitted: true})
	tt.Frame()
	got := d.tab(id)
	if got == nil {
		t.Fatalf("未提交行点「差异」应打开 %s", id)
	}
	if d.active != id {
		t.Fatalf("差异标签应被激活:active=%q", d.active)
	}
	if got.preview {
		t.Fatal("从追溯卡明确打开的差异应是固定标签,不是预览")
	}
}

// 既不是未提交、也没有哈希(读取中 / 异常):什么都不做,不 panic;没有标签时同样安全。
func TestBlameOpenDiffNoopCases(t *testing.T) {
	a, _ := demoIDE(t)
	d := a.ide
	tb := d.tab("src/router.rs")
	n := len(d.tabs)
	a.blameOpenDiff(tb, vcs.BlameLine{})
	a.blameOpenDiff(nil, vcs.BlameLine{Uncommitted: true})
	if len(d.tabs) != n {
		t.Fatalf("无效输入不该开标签:%d → %d", n, len(d.tabs))
	}
}
