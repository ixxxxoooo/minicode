package app

import "github.com/ixxxxoooo/minicode/internal/symbols"

// 过渡别名:符号索引已搬到 internal/symbols(纯逻辑,自带测试),这里保留旧名,
// 让 ide_* / lsp_* 里的十几个文件暂时不用改调用点。各文件迁到 symbols.X 之后,逐个删掉对应别名,最后删掉本文件。
type symbol = symbols.Symbol

var (
	parseSymbols = symbols.Parse
	symbolLang   = symbols.Lang
	enclosing    = symbols.Enclosing
	wordAt       = symbols.WordAt
	wordHits     = symbols.WordHits
	isKeyword    = symbols.IsKeyword
	isWordByte   = symbols.IsWordByte
	indentOf     = symbols.IndentOf
	reMd         = symbols.MarkdownHeading
)
