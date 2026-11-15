package app

// 语义高亮:MyGo editor 只有语法 Tokenizer(Highlighter),State 上没有按范围叠加语义色的 API,
// LSP semanticTokens/full 无法在不改库的情况下接入编辑器着色。
// 诊断 / 括号着色 / 同词高亮仍可用;设计稿如实标注为 part,见 docs/design js/status.js lsp-spec。
