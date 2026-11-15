package kit

import (
	"strings"
	"sync"

	"github.com/egoist/mygo/ui"
)

// 模型品牌 logo。图形与匹配规则在 logos_gen.go(scripts/gen-logos.py 生成,与设计稿 js/logos.js 同源)。

var (
	logoMu    sync.Mutex
	logoCache = map[string]*ui.SVG{}
)

// BrandOf 按名称猜品牌 id(不区分大小写);认不出返回空串。
// 模型用整个模型 ID;服务商用「名称 + 域名」(不要传带路径的 Base URL,例如 DeepSeek 的地址里有 /anthropic)。
func BrandOf(names ...string) string {
	for _, n := range names {
		s := strings.ToLower(strings.TrimSpace(n))
		if s == "" {
			continue
		}
		for _, r := range logoRules {
			for _, k := range r.Contains {
				if strings.Contains(s, k) {
					return r.Brand
				}
			}
			for _, k := range r.Prefixes {
				if strings.HasPrefix(s, k) {
					return r.Brand
				}
			}
		}
	}
	return ""
}

// logoColor 返回品牌色;没有(黑白品牌)返回 false,调用方用文字色。
func logoColor(brand string) (ui.Color, bool) {
	for _, r := range logoRules {
		if r.Brand == brand && r.Color != "" {
			return ui.Hex(r.Color), true
		}
	}
	return ui.Color{}, false
}

func logoSVG(brand string) *ui.SVG {
	logoMu.Lock()
	defer logoMu.Unlock()
	if s, ok := logoCache[brand]; ok {
		return s
	}
	body, ok := logoBodies[brand]
	if !ok {
		return nil
	}
	src := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" fill-rule="evenodd" stroke="none">` + body + `</svg>`
	s := ui.MustParseSVG([]byte(src))
	logoCache[brand] = s
	return s
}

// Logo 画一个品牌 logo(size 为边长)。brand 为空或没有对应图形时画统一的占位图标(芯片),颜色取 fallback;
// 有品牌色的品牌用品牌色,黑白品牌(OpenAI / Grok 等)用 fallback。
func Logo(c *ui.Context, brand string, size float32, fallback ui.Color) ui.Element {
	if s := logoSVG(brand); s != nil {
		col := fallback
		if bc, ok := logoColor(brand); ok {
			col = bc
		}
		return ui.Icon(c, s).Size(size, size).TextColor(col).Shrink(0)
	}
	return Ic(c, "cpu", size).TextColor(fallback).Shrink(0)
}
