// 演示模式不启动真实语言服务器(a.proj == "")。
// 「假 LSP」能力由以下来源覆盖:
//   - 本地符号索引(symbols.go):悬停 / 定义 / 引用 / 补全 / 重命名 / 大纲
//   - demo.Problems():诊断波浪线与问题面板
//   - 沙盒内多语言源文件:高亮、折叠、括号配对
//
// 若日后要在演示中模拟 InlayHint / 真实 LSP 格式化,可在此包实现 FakeLSP Client 并在 lsp_app 里对 proj=="" 分支接线。
package demo
