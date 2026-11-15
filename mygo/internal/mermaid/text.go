package mermaid

import (
	"strconv"
	"strings"
)

func charEM(c rune) float32 {
	switch c {
	case ' ':
		return 0.3
	case 'i', 'l', 'j', 'I', '.', ',', ':', ';', '|', '\'', '!', '`':
		return 0.3
	case 'f', 't', 'r', '(', ')', '[', ']', '{', '}', '-', '/', '\\':
		return 0.4
	case 'm', 'w', 'M', 'W', '@', '%':
		return 0.85
	default:
		if c >= 'A' && c <= 'Z' {
			return 0.68
		}
		if c < 128 {
			return 0.56
		}
		return 1.0
	}
}

// TextW 估算文字宽度。
func TextW(s string, size float32) float32 {
	var sum float32
	for _, c := range s {
		sum += charEM(c)
	}
	return sum * size
}

// SplitLines 拆标签行并清洗。
func SplitLines(s string) []string {
	s = strings.Trim(s, "\" \t")
	s = strings.ReplaceAll(s, "\\n", "\n")
	for _, br := range []string{"<br/>", "<br />", "<br>", "<BR/>", "<BR>"} {
		s = strings.ReplaceAll(s, br, "\n")
	}
	var lines []string
	for _, l := range strings.Split(s, "\n") {
		lines = append(lines, strings.TrimSpace(stripTags(l)))
	}
	empty := true
	for _, l := range lines {
		if l != "" {
			empty = false
			break
		}
	}
	if empty {
		return []string{""}
	}
	return lines
}

func stripTags(s string) string {
	var out strings.Builder
	inTag := false
	for _, c := range s {
		switch {
		case c == '<' && strings.Contains(s, ">"):
			inTag = true
		case c == '>' && inTag:
			inTag = false
		case inTag:
		default:
			out.WriteRune(c)
		}
	}
	t := out.String()
	for _, pair := range [][2]string{
		{"&quot;", "\""}, {"&lt;", "<"}, {"&gt;", ">"}, {"&amp;", "&"},
		{"#35;", "#"}, {"#quot;", "\""},
	} {
		t = strings.ReplaceAll(t, pair[0], pair[1])
	}
	return t
}

// LinesW 多行最大宽度。
func LinesW(lines []string, size float32) float32 {
	var max float32
	for _, l := range lines {
		if w := TextW(l, size); w > max {
			max = w
		}
	}
	return max
}

// ParseColor 解析颜色。
func ParseColor(s string) (uint32, bool) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "#") {
		h := s[1:]
		switch len(h) {
		case 3:
			v, err := strconv.ParseUint(h, 16, 12)
			if err != nil {
				return 0, false
			}
			r, g, b := (v>>8)&0xf, (v>>4)&0xf, v&0xf
			return uint32(r*17)<<16 | uint32(g*17)<<8 | uint32(b*17), true
		case 6:
			v, err := strconv.ParseUint(h, 16, 24)
			if err != nil {
				return 0, false
			}
			return uint32(v), true
		}
		return 0, false
	}
	switch strings.ToLower(s) {
	case "white":
		return 0xffffff, true
	case "black":
		return 0x000000, true
	case "red":
		return 0xe5484d, true
	case "green":
		return 0x30a46c, true
	case "blue":
		return 0x3e63dd, true
	case "yellow":
		return 0xf5d90a, true
	case "orange":
		return 0xf76b15, true
	case "gray", "grey":
		return 0x8b8d98, true
	case "pink":
		return 0xd6409f, true
	case "purple":
		return 0x8e4ec6, true
	default:
		return 0, false
	}
}

// Contrast 给定底色上的文字色。
func Contrast(rgb uint32) uint32 {
	r, g, b := (rgb>>16)&0xff, (rgb>>8)&0xff, rgb&0xff
	if (r*299+g*587+b*114)/1000 > 140 {
		return 0x1a1a1a
	}
	return 0xffffff
}
