package app

import (
	"strings"
	"testing"
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
