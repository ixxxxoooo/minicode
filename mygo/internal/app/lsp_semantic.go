package app

// 语义高亮:MyGo editor.State 仅有 TextBackgrounds(底色叠层),没有按字节范围改前景 Token 的 API;
// tree-sitter 语义色也在库内走 Highlighter 管线,应用层不能安全注入 LSP semanticTokens/full。
// 括号着色仍用 ide_brackets.go 深度染色;设计稿 lsp-spec 已标为明确取舍(不假装已接)。
