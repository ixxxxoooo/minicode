package demo

import (
	"fmt"
	"strings"

	"github.com/ixxxxoooo/minicode/internal/vcs"
)

func gitMassiveHead() string {
	var b strings.Builder
	b.WriteString("// HEAD: 模拟 5000 行中的约 3200 行\n")
	for i := 1; i <= 3200; i++ {
		fmt.Fprintf(&b, "const H_%d: u32 = %d;\n", i, i)
	}
	return b.String()
}

func gitMassiveWork() string {
	var b strings.Builder
	b.WriteString("// WORK: 改动约 2000 行,删除旧常量\n")
	for i := 1; i <= 2000; i++ {
		fmt.Fprintf(&b, "pub const W_%d: u32 = %d;\n", i, i*2)
	}
	return b.String()
}

func gitWideLine() string {
	return "let x = \"" + strings.Repeat("超长行用于横向滚动与迷你地图标尺 ", 40) + "\";\n"
}

func gitWordHighlightHead() string {
	return "let message = \"保存用户配置到磁盘\";\nlet message = \"写入用户设置\";\n"
}

func gitWordHighlightWork() string {
	return "let message = \"保存用户配置到磁盘\";\nlet message = \"持久化用户偏好\";\n"
}

func gitEOLHead() string {
	return "alpha\r\nbeta\r\n"
}

func gitEOLWork() string {
	return "alpha\nbeta\n"
}

func gitWhitespaceHead() string {
	return "fn pad() {\n    let x = 1;\n}\n"
}

func gitWhitespaceWork() string {
	return "fn pad() {\n\tlet x = 1;\n}\n"
}

func gitMultiHunkWork() string {
	return `use std::collections::HashMap;

pub struct Cache {
    inner: HashMap<String, String>,
    ttl: u64,
}

impl Cache {
    pub fn new() -> Self {
        Self { inner: HashMap::new(), ttl: 300 }
    }

    pub fn get(&self, k: &str) -> Option<&String> {
        self.inner.get(k)
    }

    pub fn set(&mut self, k: String, v: String) {
        self.inner.insert(k, v);
    }

    pub fn expire(&mut self, ttl: u64) {
        self.ttl = ttl;
    }
}
`
}

func gitMultiHunkHead() string {
	return `use std::collections::HashMap;

pub struct Cache {
    data: HashMap<String, String>,
}

impl Cache {
    pub fn new() -> Self {
        Self { data: HashMap::new() }
    }

    pub fn get(&self, k: &str) -> Option<&String> {
        self.data.get(k)
    }
}
`
}

func gitMultiHunkIndex() string {
	return strings.ReplaceAll(gitMultiHunkHead(), "data", "inner")
}

func gitRouterHeadContent() map[string]string {
	head := map[string]string{
		"src/router.rs":                    routerHead,
		"tests/router_test.rs":             "use router_demo::router::Router;\n",
		"tests/case_test.rs":               "",
		"src/server.rs":                    gitMultiHunkHead(),
		"src/handlers/users.rs":            "pub fn list() -> &'static str {\n    \"[]\"\n}\n",
		"src/legacy.rs":                    "pub fn legacy() {}\n// 40 lines trimmed in demo\n",
		"src/config_old.rs":                "pub const LEGACY: &str = \"old\";\n",
		"src/config_new.rs":                "",
		"src/config_copy.rs":               "",
		"src/util.rs":                      "",
		"demo/massive_diff.rs":             gitMassiveHead(),
		"demo/eol_crlf.txt":                gitEOLHead(),
		"demo/whitespace_only.rs":          gitWhitespaceHead(),
		"demo/word_highlight.rs":           gitWordHighlightHead(),
		"demo/wide_line.rs":                "let x = \"short\";\n",
		"demo/bin/app.ico":                 "FAKE_ICO_HEAD\n",
		"demo/conflicts/two.rs":            conflictTwo,
		"demo/conflicts/five.toml":         conflictFive,
		"demo/conflicts/broken.md":         conflictBroken,
		"demo/conflicts/binary.dat":        "BINARY_V1",
		"demo/conflicts/both_added.json":   "",
		"demo/conflicts/delete_modify.txt": "ours kept\n",
		"Cargo.lock":                       "# lock v1\n",
	}
	for i := 1; i <= 300; i++ {
		head[fmt.Sprintf("demo/bulk/file_%03d.txt", i)] = fmt.Sprintf("baseline %d\n", i)
	}
	return head
}

func gitRouterIndexContent() map[string]string {
	return map[string]string{
		"src/server.rs":         gitMultiHunkIndex(),
		"src/handlers/users.rs": "pub fn list() -> &'static str {\n    \"[\\\"demo\\\"]\"\n}\n",
		"src/util.rs":           "pub fn demo_id() -> u32 { 42 }\n",
		"Cargo.lock":            "# lock v2 staged\n",
	}
}

func gitRouterWorkExtras() map[string]string {
	m := map[string]string{
		"src/server.rs":                    gitMultiHunkWork(),
		"src/handlers/users.rs":            "pub fn list() -> &'static str {\n    \"[\\\"live\\\"]\"\n}\npub fn count() -> usize { 1 }\n",
		"src/config_new.rs":                "pub const LEGACY: &str = \"new\";\n",
		"src/config_copy.rs":               "pub const LEGACY: &str = \"copy\";\n",
		"src/util.rs":                      "pub fn demo_id() -> u32 { 42 }\npub fn extra() {}\n",
		"demo/untracked_notes.md":          "# 未跟踪\n\n草稿说明。\n",
		"demo/license.bin":                 "\x00\x01\x02\x03",
		"demo/massive_diff.rs":             gitMassiveWork(),
		"demo/eol_crlf.txt":                gitEOLWork(),
		"demo/whitespace_only.rs":          gitWhitespaceWork(),
		"demo/word_highlight.rs":           gitWordHighlightWork(),
		"demo/wide_line.rs":                gitWideLine(),
		"demo/bin/app.ico":                 "FAKE_ICO_WORK\n",
		"demo/conflicts/two.rs":            conflictTwo,
		"demo/conflicts/five.toml":         conflictFive,
		"demo/conflicts/broken.md":         conflictBroken,
		"demo/conflicts/binary.dat":        "BINARY_V2",
		"demo/conflicts/both_added.json":   conflictBothAdded,
		"demo/conflicts/delete_modify.txt": conflictDeleteModify,
		"src/cache.rs":                     gitMultiHunkWork(),
	}
	for i := 1; i <= 300; i++ {
		p := fmt.Sprintf("demo/bulk/file_%03d.txt", i)
		if i%73 == 0 {
			continue
		}
		m[p] = fmt.Sprintf("work %d\n", i)
	}
	return m
}

func gitRouterBlame() map[string][]vcs.BlameLine {
	var lines []vcs.BlameLine
	entries := []struct {
		hash, author, when, sum string
		n                       int
	}{
		{"a3f9c21", "你", "刚刚", "fix router match", 8},
		{"c44f8d1", "Lin", "昨天", "auth bearer", 6},
		{"e19a6c0", "王明", "周一", "axum 0.8", 5},
		{"f5521ab", "Wen", "11月8日", "docs", 4},
	}
	for _, e := range entries {
		for i := 0; i < e.n; i++ {
			lines = append(lines, vcs.BlameLine{Hash: e.hash, Author: e.author, When: e.when, Summary: e.sum})
		}
	}
	lines = append(lines, vcs.BlameLine{Author: "你", When: "未提交", Summary: "未提交的改动", Uncommitted: true})
	src := strings.Split(routerSrc, "\n")
	if len(lines) < len(src) {
		for len(lines) < len(src) {
			lines = append(lines, vcs.BlameLine{Hash: "f5521ab", Author: "Wen", When: "11月8日", Summary: "docs"})
		}
	}
	lines = lines[:len(src)]
	return map[string][]vcs.BlameLine{"src/router.rs": lines}
}

func gitRouterFileLog() map[string][]vcs.Commit {
	preserve := routerPreserveLog()
	var lg []vcs.Commit
	for i := 0; i < 30; i++ {
		if i < len(preserve) {
			lg = append(lg, preserve[i])
			continue
		}
		lg = append(lg, vcs.Commit{
			Hash:    fmt.Sprintf("r%05d", i),
			Subject: fmt.Sprintf("touch router.rs #%d", i),
			Author:  []string{"你", "Lin", "王明"}[i%3],
			When:    []string{"刚刚", "昨天", "去年"}[i%3],
		})
	}
	return map[string][]vcs.Commit{"src/router.rs": lg}
}
