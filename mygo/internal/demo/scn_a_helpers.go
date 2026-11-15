package demo

import (
	"fmt"
	"strings"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// patchTool 在剧本里直接追加已构造的工具卡(不走 Tool 辅助,便于预设状态)。
func patchTool(s *Script, tc *core.ToolCall, st core.ToolStatus, durMs int64, content ...core.ToolContent) {
	s.Add(ms(120), core.EvAppend{Item: tc})
	if st == core.ToolRunning || st == core.ToolPending {
		return
	}
	s.Add(ms(350), core.EvToolUpdate{ID: tc.ID, Patch: core.ToolPatch{
		Status: core.Status(st), DurationMs: durMs, AppendContent: content,
	}})
}

func readSnippet(start, n int) string {
	var b strings.Builder
	for i := start; i < start+n; i++ {
		fmt.Fprintf(&b, "%4d|    let line_%d = %d;\n", i, i, i)
	}
	return b.String()
}

func grepFilesResult() string {
	return "Found 6 files\nsrc/router.rs\nsrc/server.rs\nsrc/handlers/mod.rs\ntests/router_test.rs\ntests/integration/ci.rs\nsrc/lib.rs"
}

func grepContentResult() string {
	return "src/router.rs:41:    pub fn match_path(&self, path: &str)\nsrc/router.rs:88:        self.routes.get(path)\ntests/router_test.rs:12:    assert!(r.match_path(\"/Users\").is_some());"
}

func longGrepResult() string {
	var b strings.Builder
	b.WriteString("Found 48 files\n")
	for i := 1; i <= 48; i++ {
		fmt.Fprintf(&b, "src/pkg/module_%02d.rs\n", i)
	}
	return b.String()
}

func lsListing() string {
	return "src/\n  router.rs\n  server.rs\n  handlers/\n    mod.rs\n    error.rs\ntests/\n  router_test.rs\nCargo.toml"
}

const bashANSI = "\x1b[32m   Compiling\x1b[0m router-demo v0.3.1\n\x1b[33mwarning\x1b[0m: unused import `PathBuf`\n\x1b[32m    Finished\x1b[0m test [unoptimized] \x1b[1;32mok\x1b[0m in 0.41s"

func longBashOutput() string {
	var b strings.Builder
	for i := 1; i <= 28; i++ {
		fmt.Fprintf(&b, "running test_case_%03d ... ok\n", i)
	}
	b.WriteString("test result: ok. 28 passed; 0 failed\n")
	return b.String()
}

const tabCJKOld = "\tfn greet() -> String {\n\t    \"hello\".to_string()\n\t}"
const tabCJKNew = "\tfn greet() -> String {\n\t    \"你好，世界\".to_string() // 含 CJK 与\ttab\n\t}"

const ultraLongLine = "let row = \"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\"; // 演示超长行折行"