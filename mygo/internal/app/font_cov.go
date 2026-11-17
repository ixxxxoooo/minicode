package app

import (
	"strings"
	"sync"
)

// @author ixxxxoooo

// 终端字体的字形覆盖检查:把终端里常见的几类字符各取一组码点,问字体链里谁有字形。
// 没有字形的字符会画成方框(「豆腐块」)——设置里的字体预览据此逐类给出「自带 / 回退补全 / 缺字形」,
// 选字体时一眼就能知道哪些字符会出问题。字形查询走 CoreText(font_cov_darwin.go),其它平台没有查询、不显示结论。

// glyphCat 是一类要检查的字符。
type glyphCat struct {
	id     string
	name   string
	sample string // 预览里显示的样例(含空格分隔)
	nerd   bool   // 属于 Nerd Font 的私有区图标:缺失时给「安装 Nerd Font」
}

var glyphCats = []glyphCat{
	{id: "latin", name: "字母数字", sample: "Aa Bb Qg 0O 1lI {}[]()<>|"},
	{id: "cjk", name: "中日韩", sample: "中文 日本語 한국어"},
	{id: "box", name: "盒绘 / 方块", sample: "╭─┬─╮ ├─┼─┤ ╰─┴─╯ ▀▄█▓░"},
	{id: "sym", name: "符号 / 箭头", sample: "→ ← ↑ ↓ ≠ ≤ ≥ ★ ✓ ✗ ① • …"},
	{id: "pl", name: "Powerline", sample: "    ", nerd: true},
	{id: "nerd", name: "Nerd 图标", sample: "    ", nerd: true},
	{id: "emoji", name: "Emoji", sample: "😀 🚀 ✅"},
}

// covLevel 是一类字符在字体链里的覆盖程度。
type covLevel int

const (
	covUnknown  covLevel = iota // 没有字形查询(非 macOS)
	covOwn                      // 首选字体自带
	covFallback                 // 首选字体没有,回退链里有字体补全
	covMissing                  // 整条链都没有:会显示成方框
)

// catResult 是一类字符的检查结论。
type catResult struct {
	level covLevel
	by    string // covFallback 时,由哪个字体补
	miss  int    // covMissing 时缺字形的字符个数
}

var covCache = struct {
	sync.Mutex
	m map[string][]bool // 字体名|类别 → 各码点是否有字形;installed=false 存 nil
}{m: map[string][]bool{}}

// catRunes 取样例里要检查的码点(去空白、去重复)。
func catRunes(c glyphCat) []rune {
	seen := map[rune]bool{}
	var out []rune
	for _, r := range c.sample {
		if r == ' ' || seen[r] {
			continue
		}
		seen[r] = true
		out = append(out, r)
	}
	return out
}

// fontHas 返回某个字体对某类字符各码点的字形情况;installed=false 表示字体没装,known=false 表示这个平台没有字形查询。
func fontHas(family string, c glyphCat) (has []bool, installed, known bool) {
	key := strings.ToLower(family) + "|" + c.id
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

// resetFontCoverage 清掉字形缓存(装了新字体之后调用)。
func resetFontCoverage() {
	covCache.Lock()
	covCache.m = map[string][]bool{}
	covCache.Unlock()
}

// chainCoverage 检查字体链(首选在前、回退在后)对一类字符的覆盖:
// 首选字体全部有 = 自带;首选缺、但整条链合起来都有 = 回退补全;链上仍有字符没人有 = 缺字形。
func chainCoverage(chain []string, c glyphCat) catResult {
	cps := catRunes(c)
	have := make([]bool, len(cps))
	by := ""
	primaryAll := false
	for i, fam := range chain {
		if strings.EqualFold(fam, "monospace") {
			fam = "Menlo"
		}
		h, inst, known := fontHas(fam, c)
		if !known {
			return catResult{level: covUnknown}
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
		return catResult{level: covOwn}
	}
	miss := 0
	for _, ok := range have {
		if !ok {
			miss++
		}
	}
	if miss > 0 {
		return catResult{level: covMissing, miss: miss}
	}
	return catResult{level: covFallback, by: by}
}
