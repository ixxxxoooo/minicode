package app

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui/richtext"
)

func TestMdPatchFenceLangs(t *testing.T) {
	orig := "# t\n\n```rust\nfn main() {}\n```\n"
	neu := "# t\n\n```\nfn main() {}\n```\n"
	out := mdPatchFenceLangs(orig, neu)
	if !strings.Contains(out, "```rust") {
		t.Fatalf("应补回 rust 标注,got:\n%s", out)
	}
}

func TestMdPreserveRoundTrip(t *testing.T) {
	orig := "---\na: 1\n---\n```go\nx\n```\n"
	neu := "---\na: 1\n---\n```\ny\n```\n"
	got := mdPreserveRoundTrip(orig, neu)
	if !strings.Contains(got, "```go") {
		t.Fatalf("whole doc preserve: %q", got)
	}
}

func TestMdPreserveBodyOnlyChangedBlock(t *testing.T) {
	orig := "# 快速开始\n\n第一段:设计稿先行。\n\n第二段:先改这一\n段。\n\n第三段:段内硬换行\n仍在本段。\n\n```rust\nfn main() {}\n```\n\n- 用 * 列表\n"
	sp := mdSplitBodyBlocks(orig)
	// 模拟只改第二段
	newDoc := strings.Replace(orig, "第二段:先改这一\n段。", "第二段:先改预览里这一段。", 1)
	// 文档序列化会规范化列表符等,但 merge 应保留未改块
	ser := richtext.ParseMarkdown(newDoc).Markdown()
	got := mdPreserveBody(orig, sp, ser)
	if !strings.Contains(got, "第一段:设计稿先行。") {
		t.Fatalf("第一段应原样保留:\n%s", got)
	}
	if !strings.Contains(got, "段内硬换行\n仍在本段。") {
		t.Fatalf("第三段硬换行应保留:\n%s", got)
	}
	if !strings.Contains(got, "```rust") {
		t.Fatalf("代码块语言应保留:\n%s", got)
	}
	if !strings.Contains(got, "- 用 * 列表") {
		t.Fatalf("列表符应保留:\n%s", got)
	}
	if !strings.Contains(got, "先改预览里这一段") {
		t.Fatalf("第二段应更新:\n%s", got)
	}
}

func TestMdSplitBodyBlocksListStar(t *testing.T) {
	body := "- 用 * 混排\n"
	sp := mdSplitBodyBlocks(body)
	if len(sp.parts) != 1 || sp.parts[0].raw != "- 用 * 混排" {
		t.Fatalf("列表项切块: %+v", sp.parts)
	}
}
