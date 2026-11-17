package app

import (
	"strings"
)

// termFallbackTail 终端字体链末尾:系统等宽 + CJK + Emoji,减少缺字形方框。
var termFallbackTail = []string{
	"SF Mono", "Menlo", "Monaco", "monospace",
	"PingFang SC", "Hiragino Sans GB", "Apple SD Gothic Neo", "Apple Symbols", "Apple Color Emoji",
}

// commonMonoFonts 设置里「常见等宽」分组(顺序即展示顺序)。
var commonMonoFonts = []string{
	"", // 系统等宽
	"SF Mono", "Menlo", "Monaco", "Courier New",
	"JetBrains Mono", "Fira Code", "Cascadia Code", "Source Code Pro", "IBM Plex Mono", "Hack",
	"MesloLGS NF", "JetBrainsMono Nerd Font", "FiraCode Nerd Font", "Hack Nerd Font",
	"Symbols Nerd Font", "MesloLGS Nerd Font", "CaskaydiaCove Nerd Font",
	"Sarasa Mono SC", // 更纱黑体:中英文严格 2:1 等宽,适合中文用户
}

// parseFontFamilies 解析 VSCode 式字体列表(支持逗号与引号)。
func parseFontFamilies(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var out []string
	var cur strings.Builder
	inQuote := false
	flush := func() {
		name := strings.TrimSpace(cur.String())
		name = strings.Trim(name, `"'`)
		if name != "" {
			out = append(out, name)
		}
		cur.Reset()
	}
	for _, r := range s {
		switch r {
		case '"', '\'':
			inQuote = !inQuote
			continue
		case ',':
			if !inQuote {
				flush()
				continue
			}
		}
		cur.WriteRune(r)
	}
	flush()
	return out
}

func familyNamesEqual(a, b string) bool {
	return normFont(a) == normFont(b)
}

func chainHasFamily(chain []string, name string) bool {
	for _, f := range chain {
		if familyNamesEqual(f, name) {
			return true
		}
	}
	return false
}

// buildTermFontChain 根据偏好拼出 MyGo 可用的逗号分隔字体链。
func buildTermFontChain(pref string, nerdFallback bool) string {
	raw := parseFontFamilies(pref)
	out := make([]string, 0, len(raw)+len(termFallbackTail))
	for _, fam := range raw {
		if fam == "" {
			continue
		}
		if nerdFallback && looksNerdFont(fam) {
			if ok, known := fontInstalled(fam); known && !ok {
				if !chainHasFamily(out, "monospace") {
					out = append(out, "monospace")
				}
				continue
			}
		}
		if !chainHasFamily(out, fam) {
			out = append(out, fam)
		}
	}
	if len(out) == 0 {
		out = append(out, "monospace")
	}
	for _, tail := range termFallbackTail {
		if !chainHasFamily(out, tail) {
			out = append(out, tail)
		}
	}
	return strings.Join(out, ", ")
}
