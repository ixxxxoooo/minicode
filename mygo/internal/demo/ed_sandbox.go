package demo

import (
	"fmt"
	"strings"
)

// editorEmptyDirs 空目录占位(node_modules / target 等)。
func editorEmptyDirs() []string {
	return []string{"node_modules", "target", "bulk/many"}
}

func editorSandboxFiles() map[string]string {
	out := map[string]string{
		// ── 多语言 ──
		"samples/langs/go_demo.go": `package demo

import "fmt"

// Normalize 归一化路径演示(搜索关键字 normalize)。
func Normalize(path string) string {
	return stringsTrim(path)
}

func stringsTrim(s string) string { return s }

// Greet 用于悬停 / 转到定义 / 引用。
func Greet(name string) string {
	return "hi " + name
}

func main() {
	fmt.Println(Greet("world"))
	var n int = "type error for demo problems"
	_ = Normalize("/API/")
	_ = n
}
`,
		"samples/langs/main.ts": `// normalize 搜索命中
export interface User { id: number; name: string }

export function normalizePath(p: string): string {
  return p.replace(/\/+$/, "").toLowerCase();
}

export const DEFAULT = normalizePath("/");
`,
		"samples/langs/App.tsx": `import React from "react";
import { normalizePath } from "./main";

/** 演示 TSX 高亮与 normalize */
export function App() {
  const route = normalizePath("/Users");
  return <div className="app">{route}</div>;
}
`,
		"samples/langs/demo.py": `"""normalize 搜索与符号"""

def normalize(text: str) -> str:
    return text.strip().lower()

class Worker:
    def run(self):
        return normalize("  OK  ")
`,
		"samples/langs/Demo.swift": `// Swift 高亮(演示模式无 SourceKit)
struct Demo {
    func normalize(_ s: String) -> String { s.trimmingCharacters(in: .whitespaces) }
}
`,
		"samples/langs/hello.c": `#include <stdio.h>
/* normalize demo */
static char *normalize(const char *p) { return (char *)p; }
int main(void) { puts(normalize("hi")); return 0; }
`,
		"samples/langs/Main.java": `public class Main {
    static String normalize(String s) { return s.trim(); }
    public static void main(String[] args) {
        System.out.println(normalize(" normalize "));
    }
}
`,
		"samples/langs/query.sql": `-- normalize in SQL
SELECT id, normalize(name) AS n FROM users WHERE normalize(email) = normalize(:email);
`,
		"samples/langs/index.html": `<!DOCTYPE html>
<html lang="zh-Hans"><head><meta charset="utf-8"><title>normalize</title>
<link rel="stylesheet" href="styles.css"></head>
<body><p>normalize 演示</p></body></html>
`,
		"samples/langs/styles.css": `/* normalize token for search */
.app { display: flex; }
.normalize { text-transform: lowercase; }
`,
		"samples/langs/config.json": `{
  "name": "router-demo",
  "normalize": true,
  "routes": ["/api", "/v1"]
}
`,
		"samples/langs/config.yaml": `name: router-demo
normalize: true
routes:
  - /api
`,
		"samples/langs/config.toml": `[package]
name = "demo"

[normalize]
enabled = true
`,
		"samples/langs/run.sh": `#!/usr/bin/env bash
# normalize 搜索
normalize() { echo "$1" | tr '[:upper:]' '[:lower:]'; }
normalize "PATH"
`,
		"samples/langs/Dockerfile": `FROM alpine:3.20
# normalize label for search
RUN echo normalize > /tmp/tag
`,
		"samples/langs/Makefile": `.PHONY: test
# normalize
test:
	@echo normalize
`,
		"samples/langs/env.example": `# 演示用,非真实密钥
API_KEY=sk-••••••••
NORMALIZE_PATHS=1
`,
		"samples/langs/normalize_lib.rs": normalizeLibRS(),
		"samples/README.md":              "# 样例文件\n\n各子目录用于演示多语言、搜索关键字 **normalize**、问题面板与编辑器形态。\n",
		"samples/search/hits.txt":        genSearchHits(),
		// ── 编码边界 ──
		"edge/unicode.txt":         "中文 Tokyo 東京 🚀🎉\n零宽:\u200b\u200b看不见\nemoji 组合: 👨‍👩‍👧\n",
		"edge/mixed_eol.txt":       "LF 行\n" + "CRLF 行\r\n" + "再 LF\n",
		"edge/tabs_spaces.go":      "package edge\n\nfunc Mixed() {\n\tTabIndent := 1\n    SpaceIndent := 2\n}\n",
		"edge/no_newline_eof.txt":  "末尾没有换行符",
		"edge/bom_utf8.txt":        "\uFEFFBOM 开头文件\n第二行 normalize\n",
		"edge/empty.txt":           "",
		"edge/whitespace_only.txt": "   \t  \n  \t\n",
		"edge/binary.dat":          "BIN\x00NOT\x00TEXT",
		"edge/long_line.js":        genLongLine(50000),
		"edge/brackets.rs":         bracketNestRS(),
		"edge/fold_blocks.go":      foldBlocksGo(),
		"edge/syntax_errors.rs":    syntaxErrorsRS(),
		"edge/big_source.go":       genPrefixedLines("// line", 8000),
		"edge/big.log":             genPrefixedLines("[info]", 20000),
		"edge/big.json":            genBigJSON(),
		"edge/minified.js":         genLongLine(50000),
		"edge/wrap_scroll.ts": `// 行长与自动换行 normalize
export const oneLine = "` + strings.Repeat("normalize ", 400) + `";
`,
		// ── 资源 ──
		"assets/placeholder.png.txt": "# PNG 占位\n真实 PNG 在打包资源 assets/icon/;此处为文本占位。打开本文件可看说明;二进制应用 edge/binary.dat 触发「不能用文本打开」。\n",
		"assets/diagram.svg":         `<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg" width="64" height="64"><text x="4" y="20">normalize</text></svg>`,
		"data/sample.csv":            "id,name,normalize\n1,alpha,yes\n2,beta,yes\n",
		"data/sample.tsv":            "id\tname\tnormalize\n1\talpha\tyes\n",
		deepPathLeaf():               "深层目录第 10 层 · normalize\n",
		"compact/only/a/b/chain.rs":    "// 紧凑文件夹演示:compact/only/a/b 合并成一行\n",
		".gitignore":                 "target/\nnode_modules/\n.env\n",
		"samples/lints/multi_diag.rs": `fn main() {
    let fallback = 0;
    let h = 1;
    let normalize = normalize_path("x");
    let _ = normalize;
}
fn normalize_path(p: &str) -> String { p.into() }
`,
	}
	for k, v := range genBulkFiles() {
		out[k] = v
	}
	return out
}

func normalizeLibRS() string {
	var b strings.Builder
	b.WriteString("// 搜索演示:大量 normalize 命中\n")
	for i := 0; i < 120; i++ {
		fmtLine(&b, i)
	}
	return b.String()
}

func fmtLine(b *strings.Builder, i int) {
	fmt.Fprintf(b, "pub fn normalize_%d(x: &str) -> String { x.to_string() /* normalize */ }\n", i)
}

func bracketNestRS() string {
	return `fn main() {
    let v = [[[[[[[[[1]]]]]]]]];
    if (1 < (2 < (3 < (4 < (5 < 6)))))) {
        println!("{v:?}");
    }
}
`
}

func foldBlocksGo() string {
	var b strings.Builder
	b.WriteString("package edge\n\nfunc Big() {\n")
	for i := 0; i < 80; i++ {
		fmt.Fprintf(&b, "\t_ = %d // normalize\n", i)
	}
	b.WriteString("}\n")
	return b.String()
}

func syntaxErrorsRS() string {
	return `fn broken( {
    let x = ;
    normalize(
}
`
}

// ExtraGitStatus 供沙盒 Git 标记扩展(迷你地图刻度),不动原有三个路径。
func ExtraGitStatus() map[string]string {
	return map[string]string{
		"samples/langs/go_demo.go":    "M",
		"edge/big_source.go":          "M",
		"samples/lints/multi_diag.rs": "A",
	}
}
