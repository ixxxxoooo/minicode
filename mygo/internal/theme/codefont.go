package theme

import "strings"

// @author ixxxxoooo

// DefaultCodeFamilies 是「代码文字」默认字体链(与 BaoCode 的 CodeFont.defaultFamilies 一致):
// 取第一个已安装的字体,其余依次兜底;没装的会被框架跳过。Menlo 保证 macOS 上一定有字形。
const DefaultCodeFamilies = "JetBrains Mono, Fira Code, Menlo, Consolas, Cascadia Mono"

// DefaultCodeSize 是代码文字默认字号(BaoCode 在 macOS 上是 13)。
const DefaultCodeSize = 13

// codeChain 是当前生效的代码字体链(含末尾的系统等宽兜底),只在主线程读写。
var codeChain = DefaultCodeFamilies + ", monospace"

// CodeFont 返回代码文字应使用的字体链,传给 Element.Font / ui.Font.Family / ui.Span.Font。
// 返回:逗号分隔的字体链(设置里没改过时是默认链)。
func CodeFont() string { return codeChain }

// codeFeat 是当前代码文字的 OpenType 特性(开连字为空串,关时显式关 calt / liga)。
var codeFeat string

// SetCodeFont 设置代码字体链与连字开关(来自「设置 → 编辑器 → 字体」);chain 为空恢复默认。
// 参数:chain 逗号分隔的字体名,支持带引号;末尾总会补上系统等宽。ligatures 是否开连字。
func SetCodeFont(chain string, ligatures bool) {
	chain = strings.TrimSpace(chain)
	if chain == "" {
		chain = DefaultCodeFamilies
	}
	if !strings.Contains(strings.ToLower(chain), "monospace") {
		chain += ", monospace"
	}
	codeChain = chain
	codeFeat = ""
	if !ligatures {
		codeFeat = "calt=0,liga=0"
	}
}

// CodeFeatures 返回代码文字的 OpenType 特性(传给 ui.Font.Features / Element.FontFeatures;空串 = 字体默认)。
func CodeFeatures() string { return codeFeat }
