// Package fonts 是字体的字形覆盖检查:问一条字体链里谁有某类字符的字形。
// 不依赖界面;字形查询在 macOS 上走 CoreText(coverage_darwin.go),其它平台没有查询、不给结论。
// 字体链的解析与拼装依赖应用里的字体扫描,仍在 internal/app(term_font_chain.go)。
//
// @author ixxxxoooo
package fonts

import (
	"strings"
	"sync"
)

// 终端字体的字形覆盖检查:把终端里常见的几类字符各取一组码点,问字体链里谁有字形。
// 没有字形的字符会画成方框(「豆腐块」)——设置里的字体预览据此逐类给出「自带 / 回退补全 / 缺字形」,
// 选字体时一眼就能知道哪些字符会出问题。

// GlyphCat 是一类要检查的字符。
type GlyphCat struct {
	ID     string
	Name   string
	Sample string // 预览里显示的样例(含空格分隔)
	Nerd   bool   // 属于 Nerd Font 的私有区图标:缺失时给「安装 Nerd Font」
}

// GlyphCats 是预览里逐类检查的字符类别(顺序即显示顺序;第一类用来判断字体是否已安装)。
var GlyphCats = []GlyphCat{
	{ID: "latin", Name: "字母数字", Sample: "Aa Bb Qg 0O 1lI {}[]()<>|"},
	{ID: "cjk", Name: "中日韩", Sample: "中文 日本語 한국어"},
	{ID: "box", Name: "盒绘 / 方块", Sample: "╭─┬─╮ ├─┼─┤ ╰─┴─╯ ▀▄█▓░"},
	{ID: "sym", Name: "符号 / 箭头", Sample: "→ ← ↑ ↓ ≠ ≤ ≥ ★ ✓ ✗ ① • …"},
	{ID: "pl", Name: "Powerline", Sample: "    ", Nerd: true},
	{ID: "nerd", Name: "Nerd 图标", Sample: "    ", Nerd: true},
	{ID: "emoji", Name: "Emoji", Sample: "😀 🚀 ✅"},
}

// CovLevel 是一类字符在字体链里的覆盖程度。
type CovLevel int

const (
	CovUnknown  CovLevel = iota // 没有字形查询(非 macOS)
	CovOwn                      // 首选字体自带
	CovFallback                 // 首选字体没有,回退链里有字体补全
	CovMissing                  // 整条链都没有:会显示成方框
)

// CatResult 是一类字符的检查结论。
type CatResult struct {
	Level CovLevel
	By    string // CovFallback 时,由哪个字体补
	Miss  int    // CovMissing 时缺字形的字符个数
}

var covCache = struct {
	sync.Mutex
	m map[string][]bool // 字体名|类别 → 各码点是否有字形;installed=false 存 nil
}{m: map[string][]bool{}}

// catRunes 取样例里要检查的码点(去空白、去重复)。
func catRunes(c GlyphCat) []rune {
	seen := map[rune]bool{}
	var out []rune
	for _, r := range c.Sample {
		if r == ' ' || seen[r] {
			continue
		}
		seen[r] = true
		out = append(out, r)
	}
	return out
}

// Has 返回某个字体对某类字符各码点的字形情况;installed=false 表示字体没装,known=false 表示这个平台没有字形查询。
func Has(family string, c GlyphCat) (has []bool, installed, known bool) {
	key := strings.ToLower(family) + "|" + c.ID
	covCache.Lock()
	h, hit := covCache.m[key]
	covCache.Unlock()
	if hit {
		return h, h != nil, true
	}
	h, inst := platformFontCoverage(family, catRunes(c))
	if h == nil && inst { // 平台不支持查询
		return nil, true, false
	}
	covCache.Lock()
	covCache.m[key] = h
	covCache.Unlock()
	return h, inst, true
}

// ResetCoverage 清掉字形缓存(装了新字体之后调用)。
func ResetCoverage() {
	covCache.Lock()
	covCache.m = map[string][]bool{}
	covCache.Unlock()
}

// ChainCoverage 检查字体链(首选在前、回退在后)对一类字符的覆盖:
// 首选字体全部有 = 自带;首选缺、但整条链合起来都有 = 回退补全;链上仍有字符没人有 = 缺字形。
func ChainCoverage(chain []string, c GlyphCat) CatResult {
	cps := catRunes(c)
	have := make([]bool, len(cps))
	by := ""
	primaryAll := false
	for i, fam := range chain {
		if strings.EqualFold(fam, "monospace") {
			fam = "Menlo"
		}
		h, inst, known := Has(fam, c)
		if !known {
			return CatResult{Level: CovUnknown}
		}
		if !inst {
			continue
		}
		all := true
		for j := range cps {
			if h[j] && !have[j] {
				have[j] = true
				if i > 0 && by == "" {
					by = fam
				}
			}
			if !h[j] {
				all = false
			}
		}
		if i == 0 && all {
			primaryAll = true
		}
	}
	if primaryAll {
		return CatResult{Level: CovOwn}
	}
	miss := 0
	for _, ok := range have {
		if !ok {
			miss++
		}
	}
	if miss > 0 {
		return CatResult{Level: CovMissing, Miss: miss}
	}
	return CatResult{Level: CovFallback, By: by}
}
