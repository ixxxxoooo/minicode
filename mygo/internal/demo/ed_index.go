// 演示文件索引 · 编辑器 / 终端 / 搜索 / 问题面板
//
// 沙盒在 demo.Sandbox() 中装配;展开逻辑见 ed_sandbox.go / ed_gen.go。
// 点开路径均在 router-demo 项目根下(演示模式侧栏「router-demo」会话)。
//
// ── 保留(与 Git / 设计稿截图绑定,勿改内容)──
//
//	src/router.rs, src/main.rs, tests/*, docs/guide.md, docs/api.md, .cursor/rules/style.mdc, Cargo.toml 等
//
// ── samples/langs · 多语言与高亮 / 大纲 / 补全──
//
//	samples/langs/go_demo.go       Go:符号、normalize 搜索命中、故意类型错误(问题面板)
//	samples/langs/App.tsx          TSX + normalize
//	samples/langs/main.ts          TypeScript 接口与 normalize
//	samples/langs/demo.py          Python
//	samples/langs/Demo.swift       Swift(无语言服务,仅高亮)
//	samples/langs/hello.c          C
//	samples/langs/Main.java        Java
//	samples/langs/query.sql        SQL
//	samples/langs/index.html       HTML
//	samples/langs/styles.css       CSS
//	samples/langs/config.json      JSON
//	samples/langs/config.yaml      YAML
//	samples/langs/config.toml      TOML
//	samples/langs/run.sh           Shell
//	samples/langs/Dockerfile       Docker
//	samples/langs/Makefile         Make
//	samples/langs/env.example      环境变量示例(无真实密钥)
//	samples/langs/normalize_lib.rs 大量 normalize 命中(搜索「大量结果」)
//
// ── edge · 边界编码与编辑器形态──
//
//	edge/unicode.txt               CJK、emoji、零宽字符 U+200B
//	edge/mixed_eol.txt             CRLF 与 LF 混排
//	edge/tabs_spaces.go            Tab 与空格混用缩进
//	edge/no_newline_eof.txt        末尾无换行
//	edge/bom_utf8.txt              UTF-8 BOM 开头
//	edge/empty.txt                 空文件
//	edge/whitespace_only.txt       仅空白
//	edge/binary.dat                含 NUL,打开应提示二进制
//	edge/long_line.js              单行约 5 万字符(横向滚动 / 换行)
//	edge/brackets.rs               深括号嵌套(配对竖线 / 彩虹括号)
//	edge/fold_blocks.go            长函数体(折叠)
//	edge/syntax_errors.rs          大量语法问题(本地诊断 + rust 名)
//	edge/big_source.go             约 8000 行(滚动 / 迷你地图)
//	edge/big.log                   约 2 万行日志
//	edge/big.json                  约 200KB JSON
//	edge/minified.js               压缩单行 JS
//	assets/placeholder.png.txt     PNG 占位说明(真二进制请用 .dat)
//	assets/diagram.svg             SVG
//
// ── data · 表格──
//
//	data/sample.csv, data/sample.tsv
//
// ── deep · 深层目录──
//
//	deep/l1/…/l10/leaf.txt         10 层嵌套
//
// ── bulk/many · 大目录──
//
//	bulk/many/file_0001.txt …      1500 个文件(列表性能)
//
// ── 默认忽略(空目录占位)──
//
//	node_modules/, target/
//
// ── 演示接线──
//
//	问题/波浪线: demo.Problems() → ide_overlay / ide_panel
//	终端静态行: demo.TermDemoLines() → newIDE
//	命令只读标签: demo.CmdViews() → seedWorkspace
//	输出面板预置: demo.SeedOutput(func) → SeedDemoExtras
package demo
